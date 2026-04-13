package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/tmdb"

	"github.com/google/uuid"
)

type importService struct {
	collectionRepo     ports.CollectionRepository
	collectionItemRepo ports.CollectionItemRepository
	reviewRepo         ports.ReviewRepository
	tmdbClient         ports.TMDBClient
}

func NewImportService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	reviewRepo ports.ReviewRepository,
	tmdbClient ports.TMDBClient,
) ports.ImportService {
	return &importService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		reviewRepo:         reviewRepo,
		tmdbClient:         tmdbClient,
	}
}

type watchedEntry struct {
	Name string
	Year int
}

type ratingEntry struct {
	Name   string
	Year   int
	Rating float64
}

type reviewEntry struct {
	Name   string
	Year   int
	Rating float64
	Review string
}

type watchlistEntry struct {
	Name string
	Year int
}

func filmKey(name string, year int) string {
	return strings.ToLower(name) + "|" + strconv.Itoa(year)
}

func (s *importService) ImportLetterboxd(ctx context.Context, userID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (*ports.ImportResult, error) {
	archive, err := zip.NewReader(zipReader, zipSize)
	if err != nil {
		return nil, domain.ErrInvalidImportFile
	}

	watched := parseWatched(archive)
	ratings := parseRatings(archive)
	reviews := parseReviews(archive)
	watchlist := parseWatchlist(archive)

	// Collect all unique films across all CSVs
	uniqueFilms := make(map[string]watchedEntry)
	for _, w := range watched {
		key := filmKey(w.Name, w.Year)
		uniqueFilms[key] = watchedEntry{Name: w.Name, Year: w.Year}
	}
	for _, r := range ratings {
		key := filmKey(r.Name, r.Year)
		if _, exists := uniqueFilms[key]; !exists {
			uniqueFilms[key] = watchedEntry{Name: r.Name, Year: r.Year}
		}
	}
	for _, r := range reviews {
		key := filmKey(r.Name, r.Year)
		if _, exists := uniqueFilms[key]; !exists {
			uniqueFilms[key] = watchedEntry{Name: r.Name, Year: r.Year}
		}
	}
	for _, w := range watchlist {
		key := filmKey(w.Name, w.Year)
		if _, exists := uniqueFilms[key]; !exists {
			uniqueFilms[key] = watchedEntry{Name: w.Name, Year: w.Year}
		}
	}

	// Resolve all films to TMDB IDs concurrently
	resolved, failed := s.resolveFilms(ctx, uniqueFilms)

	result := &ports.ImportResult{
		Failed: failed,
	}

	// Look up the "watched" and "to-watch" collection IDs once
	watchedCol, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, "watched")
	if err != nil || watchedCol == nil {
		return nil, domain.ErrInternal
	}
	toWatchCol, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, "to-watch")
	if err != nil || toWatchCol == nil {
		return nil, domain.ErrInternal
	}

	// Import watched films — add directly to collection, skip TMDB detail fetch
	for _, w := range watched {
		key := filmKey(w.Name, w.Year)
		tmdbID, ok := resolved[key]
		if !ok {
			continue
		}
		if s.addCollectionItem(ctx, watchedCol.ID, tmdbID) {
			result.Watched.Imported++
		} else {
			result.Watched.Skipped++
		}
	}

	// Import watchlist
	for _, w := range watchlist {
		key := filmKey(w.Name, w.Year)
		tmdbID, ok := resolved[key]
		if !ok {
			continue
		}
		if s.addCollectionItem(ctx, toWatchCol.ID, tmdbID) {
			result.Watchlist.Imported++
		} else {
			result.Watchlist.Skipped++
		}
	}

	// Merge ratings and reviews by film key
	type mergedReview struct {
		Rating float64
		Review string
	}
	merged := make(map[string]mergedReview)
	for _, r := range ratings {
		key := filmKey(r.Name, r.Year)
		merged[key] = mergedReview{Rating: r.Rating}
	}
	for _, r := range reviews {
		key := filmKey(r.Name, r.Year)
		entry := merged[key]
		entry.Review = r.Review
		if r.Rating > 0 && entry.Rating == 0 {
			entry.Rating = r.Rating
		}
		merged[key] = entry
	}

	now := time.Now()
	for key, m := range merged {
		tmdbID, ok := resolved[key]
		if !ok {
			continue
		}

		if m.Rating < 0.5 || m.Rating > 5.0 {
			continue
		}

		existing, err := s.reviewRepo.GetByUserIDAndTMDBID(ctx, userID, tmdbID)
		if err != nil {
			continue
		}

		hasReviewText := m.Review != ""

		if existing != nil {
			if hasReviewText {
				result.Reviews.Skipped++
			} else {
				result.Ratings.Skipped++
			}
			continue
		}

		review := &domain.Review{
			ID:               uuid.New(),
			UserID:           userID,
			TMDBID:           tmdbID,
			Rating:           m.Rating,
			ContainsSpoilers: false,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if hasReviewText {
			review.Content = &m.Review
		}

		if err := s.reviewRepo.Create(ctx, review); err != nil {
			continue
		}

		if hasReviewText {
			result.Reviews.Imported++
		} else {
			result.Ratings.Imported++
		}

		// Ensure reviewed films are in the watched collection
		s.addCollectionItem(ctx, watchedCol.ID, tmdbID)
	}

	if result.Failed == nil {
		result.Failed = []ports.ImportFailure{}
	}

	return result, nil
}

// addCollectionItem adds a film to a collection directly, skipping TMDB detail fetch.
// Returns true if the item was added, false if it already existed or on error.
func (s *importService) addCollectionItem(ctx context.Context, collectionID uuid.UUID, tmdbID int) bool {
	existing, err := s.collectionItemRepo.GetByCollectionIDAndTMDBID(ctx, collectionID, tmdbID)
	if err != nil || existing != nil {
		return false
	}

	item := &domain.CollectionItem{
		CollectionID: collectionID,
		TMDBID:       tmdbID,
		AddedAt:      time.Now(),
		Runtime:      0,
		Metadata:     json.RawMessage("{}"),
	}

	if err := s.collectionItemRepo.Create(ctx, item); err != nil {
		return false
	}
	return true
}

func (s *importService) resolveFilms(ctx context.Context, films map[string]watchedEntry) (map[string]int, []ports.ImportFailure) {
	resolved := make(map[string]int)
	var failed []ports.ImportFailure
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for key, film := range films {
		wg.Add(1)
		go func(key string, film watchedEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
				Query:    film.Name,
				Year:     film.Year,
				Language: "en-US",
			})

			mu.Lock()
			defer mu.Unlock()

			if err != nil || resp == nil || len(resp.Results) == 0 {
				failed = append(failed, ports.ImportFailure{
					Name:   film.Name,
					Year:   film.Year,
					Reason: "no TMDB match found",
				})
				return
			}

			resolved[key] = resp.Results[0].ID
		}(key, film)
	}

	wg.Wait()
	return resolved, failed
}

// CSV parsing helpers

func findFileInZip(archive *zip.Reader, filename string) *zip.File {
	// Prefer exact root-level match first
	for _, f := range archive.File {
		if strings.EqualFold(f.Name, filename) {
			return f
		}
	}
	// Fallback: match by filename in any subdirectory
	for _, f := range archive.File {
		if idx := strings.LastIndex(f.Name, "/"); idx >= 0 {
			if strings.EqualFold(f.Name[idx+1:], filename) {
				return f
			}
		}
	}
	return nil
}

func readCSV(f *zip.File) ([][]string, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	return reader.ReadAll()
}

func parseWatched(archive *zip.Reader) []watchedEntry {
	f := findFileInZip(archive, "watched.csv")
	if f == nil {
		return nil
	}

	records, err := readCSV(f)
	if err != nil || len(records) < 2 {
		return nil
	}

	var entries []watchedEntry
	for _, row := range records[1:] {
		if len(row) < 3 {
			continue
		}
		year, err := strconv.Atoi(row[2])
		if err != nil {
			continue
		}
		entries = append(entries, watchedEntry{
			Name: row[1],
			Year: year,
		})
	}
	return entries
}

func parseRatings(archive *zip.Reader) []ratingEntry {
	f := findFileInZip(archive, "ratings.csv")
	if f == nil {
		return nil
	}

	records, err := readCSV(f)
	if err != nil || len(records) < 2 {
		return nil
	}

	var entries []ratingEntry
	for _, row := range records[1:] {
		if len(row) < 5 {
			continue
		}
		year, err := strconv.Atoi(row[2])
		if err != nil {
			continue
		}
		rating, err := strconv.ParseFloat(row[4], 64)
		if err != nil || rating < 0.5 {
			continue
		}
		entries = append(entries, ratingEntry{
			Name:   row[1],
			Year:   year,
			Rating: rating,
		})
	}
	return entries
}

func parseReviews(archive *zip.Reader) []reviewEntry {
	f := findFileInZip(archive, "reviews.csv")
	if f == nil {
		return nil
	}

	records, err := readCSV(f)
	if err != nil || len(records) < 2 {
		return nil
	}

	var entries []reviewEntry
	for _, row := range records[1:] {
		// reviews.csv columns: Date, Name, Year, Letterboxd URI, Rating, Rewatch, Review, Tags, Watched Date
		if len(row) < 7 {
			continue
		}
		year, err := strconv.Atoi(row[2])
		if err != nil {
			continue
		}
		reviewText := strings.TrimSpace(row[6])
		if reviewText == "" {
			continue
		}
		rating, _ := strconv.ParseFloat(row[4], 64)
		entries = append(entries, reviewEntry{
			Name:   row[1],
			Year:   year,
			Rating: rating,
			Review: reviewText,
		})
	}
	return entries
}

func parseWatchlist(archive *zip.Reader) []watchlistEntry {
	f := findFileInZip(archive, "watchlist.csv")
	if f == nil {
		return nil
	}

	records, err := readCSV(f)
	if err != nil || len(records) < 2 {
		return nil
	}

	var entries []watchlistEntry
	for _, row := range records[1:] {
		if len(row) < 3 {
			continue
		}
		year, err := strconv.Atoi(row[2])
		if err != nil {
			continue
		}
		entries = append(entries, watchlistEntry{
			Name: row[1],
			Year: year,
		})
	}
	return entries
}
