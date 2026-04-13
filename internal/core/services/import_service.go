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
	"sync/atomic"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/tmdb"
	ws "duskforge-api/pkg/websocket"

	"github.com/google/uuid"
)

type importService struct {
	collectionRepo     ports.CollectionRepository
	collectionItemRepo ports.CollectionItemRepository
	reviewRepo         ports.ReviewRepository
	tmdbClient         ports.TMDBClient
	hub                *ws.Hub

	// In-memory progress tracking per user
	progress sync.Map // map[uuid.UUID]*ports.ImportProgress
}

func NewImportService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	reviewRepo ports.ReviewRepository,
	tmdbClient ports.TMDBClient,
	hub *ws.Hub,
) ports.ImportService {
	return &importService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		reviewRepo:         reviewRepo,
		tmdbClient:         tmdbClient,
		hub:                hub,
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

type resolvedFilm struct {
	tmdbID  int
	runtime int16
}

func filmKey(name string, year int) string {
	return strings.ToLower(name) + "|" + strconv.Itoa(year)
}

func (s *importService) StartImportLetterboxd(ctx context.Context, userID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (*ports.ImportProgress, error) {
	// Check if an import is already running for this user
	if existing, ok := s.progress.Load(userID); ok {
		p := existing.(*ports.ImportProgress)
		if p.Status == ports.ImportStatusProcessing {
			return p, nil
		}
	}

	archive, err := zip.NewReader(zipReader, zipSize)
	if err != nil {
		return nil, domain.ErrInvalidImportFile
	}

	// Parse CSVs synchronously (fast, no API calls)
	watched := parseWatched(archive)
	ratings := parseRatings(archive)
	reviews := parseReviews(archive)
	watchlist := parseWatchlist(archive)

	// Collect all unique films
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

	progress := &ports.ImportProgress{
		Status: ports.ImportStatusProcessing,
		Phase:  "resolving",
		Total:  len(uniqueFilms),
	}
	s.setProgress(userID, progress)

	// Launch the import in the background
	go s.processImport(userID, uniqueFilms, watched, ratings, reviews, watchlist)

	return progress, nil
}

func (s *importService) GetImportStatus(userID uuid.UUID) *ports.ImportProgress {
	if p, ok := s.progress.Load(userID); ok {
		return p.(*ports.ImportProgress)
	}
	return nil
}

// setProgress stores progress and pushes it to the user via WebSocket.
func (s *importService) setProgress(userID uuid.UUID, p *ports.ImportProgress) {
	s.progress.Store(userID, p)
	s.hub.SendToUser(userID, ws.Event{
		Type: ws.EventImportProgress,
		Data: p,
	})
}

func (s *importService) processImport(
	userID uuid.UUID,
	uniqueFilms map[string]watchedEntry,
	watched []watchedEntry,
	ratings []ratingEntry,
	reviews []reviewEntry,
	watchlist []watchlistEntry,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	total := len(uniqueFilms)

	// Phase 1: Resolve all films to TMDB IDs and fetch runtimes
	resolved, failed := s.resolveFilms(ctx, uniqueFilms, func(count int) {
		s.setProgress(userID, &ports.ImportProgress{
			Status:   ports.ImportStatusProcessing,
			Phase:    "resolving",
			Total:    total,
			Resolved: count,
		})
	})

	s.setProgress(userID, &ports.ImportProgress{
		Status:   ports.ImportStatusProcessing,
		Phase:    "importing",
		Total:    total,
		Resolved: len(resolved),
	})

	// Phase 2: Import into collections and create reviews
	result := &ports.ImportResult{
		Failed: failed,
	}

	watchedCol, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, "watched")
	if err != nil || watchedCol == nil {
		s.setProgress(userID, &ports.ImportProgress{
			Status: ports.ImportStatusFailed,
			Phase:  "importing",
			Total:  total,
			Error:  "failed to find watched collection",
		})
		return
	}
	toWatchCol, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, "to-watch")
	if err != nil || toWatchCol == nil {
		s.setProgress(userID, &ports.ImportProgress{
			Status: ports.ImportStatusFailed,
			Phase:  "importing",
			Total:  total,
			Error:  "failed to find to-watch collection",
		})
		return
	}

	// Import watched films
	for _, w := range watched {
		key := filmKey(w.Name, w.Year)
		film, ok := resolved[key]
		if !ok {
			continue
		}
		if s.addCollectionItem(ctx, watchedCol.ID, film.tmdbID, film.runtime) {
			result.Watched.Imported++
		} else {
			result.Watched.Skipped++
		}
	}

	// Import watchlist
	for _, w := range watchlist {
		key := filmKey(w.Name, w.Year)
		film, ok := resolved[key]
		if !ok {
			continue
		}
		if s.addCollectionItem(ctx, toWatchCol.ID, film.tmdbID, film.runtime) {
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
		film, ok := resolved[key]
		if !ok {
			continue
		}

		if m.Rating < 0.5 || m.Rating > 5.0 {
			continue
		}

		existing, err := s.reviewRepo.GetByUserIDAndTMDBID(ctx, userID, film.tmdbID)
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
			TMDBID:           film.tmdbID,
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
		s.addCollectionItem(ctx, watchedCol.ID, film.tmdbID, film.runtime)
	}

	if result.Failed == nil {
		result.Failed = []ports.ImportFailure{}
	}

	// Mark as completed
	s.setProgress(userID, &ports.ImportProgress{
		Status:   ports.ImportStatusCompleted,
		Phase:    "done",
		Total:    total,
		Resolved: len(resolved),
		Result:   result,
	})
}

// addCollectionItem adds a film to a collection directly.
// Returns true if the item was added, false if it already existed or on error.
func (s *importService) addCollectionItem(ctx context.Context, collectionID uuid.UUID, tmdbID int, runtime int16) bool {
	existing, err := s.collectionItemRepo.GetByCollectionIDAndTMDBID(ctx, collectionID, tmdbID)
	if err != nil || existing != nil {
		return false
	}

	item := &domain.CollectionItem{
		CollectionID: collectionID,
		TMDBID:       tmdbID,
		AddedAt:      time.Now(),
		Runtime:      runtime,
		Metadata:     json.RawMessage("{}"),
	}

	if err := s.collectionItemRepo.Create(ctx, item); err != nil {
		return false
	}
	return true
}

// bestMatch picks the most accurate result from a TMDB search by comparing
// title and release year against the Letterboxd entry.
func bestMatch(results []tmdb.MovieSummary, name string, year int) *tmdb.MovieSummary {
	nameLower := strings.ToLower(strings.TrimSpace(name))

	// Pass 1: exact title (or original title) + exact year
	for i, r := range results {
		ry := releaseYear(r.ReleaseDate)
		if ry == year && (strings.ToLower(r.Title) == nameLower || strings.ToLower(r.OriginalTitle) == nameLower) {
			return &results[i]
		}
	}

	// Pass 2: exact year, any title (TMDB already filtered by query)
	for i, r := range results {
		if releaseYear(r.ReleaseDate) == year {
			return &results[i]
		}
	}

	// Pass 3: exact title match regardless of year
	for i, r := range results {
		if strings.ToLower(r.Title) == nameLower || strings.ToLower(r.OriginalTitle) == nameLower {
			return &results[i]
		}
	}

	// Fallback: first result
	return &results[0]
}

func releaseYear(date string) int {
	if len(date) >= 4 {
		y, err := strconv.Atoi(date[:4])
		if err == nil {
			return y
		}
	}
	return 0
}

// resolveFilms resolves film names to TMDB IDs and fetches runtimes concurrently.
func (s *importService) resolveFilms(ctx context.Context, films map[string]watchedEntry, onProgress func(int)) (map[string]resolvedFilm, []ports.ImportFailure) {
	resolved := make(map[string]resolvedFilm)
	var failed []ports.ImportFailure
	var mu sync.Mutex
	var wg sync.WaitGroup
	var count atomic.Int32
	sem := make(chan struct{}, 10)

	for key, film := range films {
		wg.Add(1)
		go func(key string, film watchedEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			match := s.searchFilm(ctx, film.Name, film.Year)
			if match == nil {
				mu.Lock()
				failed = append(failed, ports.ImportFailure{
					Name:   film.Name,
					Year:   film.Year,
					Reason: "no TMDB match found",
				})
				mu.Unlock()
				newCount := int(count.Add(1))
				onProgress(newCount)
				return
			}

			tmdbID := match.ID

			// Fetch runtime from movie details
			var runtime int16
			details, err := s.tmdbClient.GetMovieDetails(ctx, tmdbID, "en-US")
			if err == nil && details != nil && details.Runtime != nil {
				runtime = int16(*details.Runtime)
			}

			mu.Lock()
			resolved[key] = resolvedFilm{tmdbID: tmdbID, runtime: runtime}
			mu.Unlock()

			newCount := int(count.Add(1))
			onProgress(newCount)
		}(key, film)
	}

	wg.Wait()
	return resolved, failed
}

// searchFilm tries to find the best TMDB match for a film name + year.
// It first searches with PrimaryReleaseYear for strict matching, then
// falls back to a broader Year search if needed.
func (s *importService) searchFilm(ctx context.Context, name string, year int) *tmdb.MovieSummary {
	// Try strict search with primary_release_year
	resp, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:              name,
		PrimaryReleaseYear: year,
		Language:           "en-US",
	})
	if err == nil && resp != nil && len(resp.Results) > 0 {
		return bestMatch(resp.Results, name, year)
	}

	// Fallback: broader year search
	resp, err = s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    name,
		Year:     year,
		Language: "en-US",
	})
	if err == nil && resp != nil && len(resp.Results) > 0 {
		return bestMatch(resp.Results, name, year)
	}

	// Last resort: search without year
	resp, err = s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    name,
		Language: "en-US",
	})
	if err == nil && resp != nil && len(resp.Results) > 0 {
		return bestMatch(resp.Results, name, year)
	}

	return nil
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
