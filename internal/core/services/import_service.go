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
	"duskforge-api/pkg/logger"
	"duskforge-api/pkg/tmdb"
	ws "duskforge-api/pkg/websocket"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type importService struct {
	collectionRepo     ports.CollectionRepository
	collectionItemRepo ports.CollectionItemRepository
	reviewRepo         ports.ReviewRepository
	tmdbClient         ports.TMDBClient
	hub                *ws.Hub
	achievementSvc     ports.AchievementService

	progress sync.Map
}

func NewImportService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	reviewRepo ports.ReviewRepository,
	tmdbClient ports.TMDBClient,
	hub *ws.Hub,
	achievementSvc ports.AchievementService,
) ports.ImportService {
	return &importService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		reviewRepo:         reviewRepo,
		tmdbClient:         tmdbClient,
		hub:                hub,
		achievementSvc:     achievementSvc,
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

type backfillItem struct {
	collectionID uuid.UUID
	tmdbID       int
}

func filmKey(name string, year int) string {
	return strings.ToLower(name) + "|" + strconv.Itoa(year)
}

func (s *importService) StartImportLetterboxd(ctx context.Context, userID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (*ports.ImportProgress, error) {
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

	watched := parseWatched(archive)
	ratings := parseRatings(archive)
	reviews := parseReviews(archive)
	watchlist := parseWatchlist(archive)

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

	if len(uniqueFilms) == 0 {
		return nil, domain.ErrImportFileEmpty
	}

	progress := &ports.ImportProgress{
		Status: ports.ImportStatusProcessing,
		Phase:  "resolving",
		Total:  len(uniqueFilms),
	}
	s.setProgress(userID, progress)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("import-processor panic", zap.Any("panic", r))
			}
		}()
		s.processImport(userID, uniqueFilms, watched, ratings, reviews, watchlist)
	}()

	return progress, nil
}

func (s *importService) GetImportStatus(userID uuid.UUID) *ports.ImportProgress {
	if p, ok := s.progress.Load(userID); ok {
		return p.(*ports.ImportProgress)
	}
	return nil
}

func (s *importService) setProgress(userID uuid.UUID, p *ports.ImportProgress) {
	s.progress.Store(userID, p)
	s.hub.SendToUser(userID, ws.Event{
		Type: ws.EventImportProgress,
		Data: p,
	})
}

func (s *importService) setProgressQuiet(userID uuid.UUID, p *ports.ImportProgress) {
	s.progress.Store(userID, p)
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

	wsInterval := max(total/100, 1)
	resolved, failed := s.resolveFilms(ctx, uniqueFilms, func(count int) {
		p := &ports.ImportProgress{
			Status:   ports.ImportStatusProcessing,
			Phase:    "resolving",
			Total:    total,
			Resolved: count,
		}
		if count%wsInterval == 0 || count == total {
			s.setProgress(userID, p)
		} else {
			s.setProgressQuiet(userID, p)
		}
	})

	s.setProgress(userID, &ports.ImportProgress{
		Status:   ports.ImportStatusProcessing,
		Phase:    "importing",
		Total:    total,
		Resolved: len(resolved),
	})

	result := &ports.ImportResult{
		Failed: failed,
	}

	watchedCol, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, domain.SystemCollectionWatched)
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

	tmdbIDSet := make(map[int]struct{}, len(merged))
	for key, m := range merged {
		film, ok := resolved[key]
		if !ok {
			continue
		}
		if m.Rating < 0.5 || m.Rating > 5.0 {
			continue
		}
		tmdbIDSet[film.tmdbID] = struct{}{}
	}
	tmdbIDs := make([]int, 0, len(tmdbIDSet))
	for id := range tmdbIDSet {
		tmdbIDs = append(tmdbIDs, id)
	}
	existingReviews, batchErr := s.reviewRepo.GetByUserIDAndTMDBIDs(ctx, userID, tmdbIDs)

	for key, m := range merged {
		film, ok := resolved[key]
		if !ok {
			continue
		}

		if m.Rating < 0.5 || m.Rating > 5.0 {
			continue
		}

		var existing *domain.Review
		if batchErr != nil {
			e, err := s.reviewRepo.GetByUserIDAndTMDBID(ctx, userID, film.tmdbID)
			if err != nil {
				continue
			}
			existing = e
		} else {
			existing = existingReviews[film.tmdbID]
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

		s.addCollectionItem(ctx, watchedCol.ID, film.tmdbID, film.runtime)
	}

	if result.Failed == nil {
		result.Failed = []ports.ImportFailure{}
	}

	if s.achievementSvc != nil {
		_, _ = s.achievementSvc.EvaluateAllForUser(ctx, userID)
	}

	s.setProgress(userID, &ports.ImportProgress{
		Status:   ports.ImportStatusCompleted,
		Phase:    "done",
		Total:    total,
		Resolved: len(resolved),
		Result:   result,
	})

	seen := make(map[int]bool)
	var backfillItems []backfillItem

	for _, w := range watched {
		key := filmKey(w.Name, w.Year)
		film, ok := resolved[key]
		if !ok || seen[film.tmdbID] {
			continue
		}
		seen[film.tmdbID] = true
		backfillItems = append(backfillItems, backfillItem{collectionID: watchedCol.ID, tmdbID: film.tmdbID})
	}
	for _, w := range watchlist {
		key := filmKey(w.Name, w.Year)
		film, ok := resolved[key]
		if !ok || seen[film.tmdbID] {
			continue
		}
		seen[film.tmdbID] = true
		backfillItems = append(backfillItems, backfillItem{collectionID: toWatchCol.ID, tmdbID: film.tmdbID})
	}

	s.backfillRuntimes(ctx, userID, backfillItems)
}

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

func (s *importService) backfillRuntimes(ctx context.Context, userID uuid.UUID, items []backfillItem) {
	if len(items) == 0 {
		return
	}

	s.setProgress(userID, &ports.ImportProgress{
		Status: ports.ImportStatusCompleted,
		Phase:  "enriching",
	})

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, item := range items {
		wg.Add(1)
		go func(it backfillItem) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.Logger.Error("import-backfill-worker panic", zap.Any("panic", r))
				}
			}()
			sem <- struct{}{}
			defer func() { <-sem }()

			details, err := s.tmdbClient.GetMovieDetails(ctx, it.tmdbID, "en-US")
			if err != nil || details == nil || details.Runtime == nil {
				return
			}

			runtime := int16(*details.Runtime)
			if runtime > 0 {
				_ = s.collectionItemRepo.UpdateRuntime(ctx, it.collectionID, it.tmdbID, runtime)
			}
		}(item)
	}

	wg.Wait()

	if s.achievementSvc != nil {
		_, _ = s.achievementSvc.EvaluateForEvent(ctx, userID, domain.AchievementCategoryWatching)
	}
}

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
			defer func() {
				if r := recover(); r != nil {
					logger.Logger.Error("import-resolve-worker panic", zap.Any("panic", r))
				}
			}()
			sem <- struct{}{}
			defer func() { <-sem }()

			tmdbID, ok := s.searchAndMatch(ctx, film.Name, film.Year)
			if !ok {
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

			mu.Lock()
			resolved[key] = resolvedFilm{tmdbID: tmdbID, runtime: 0}
			mu.Unlock()

			newCount := int(count.Add(1))
			onProgress(newCount)
		}(key, film)
	}

	wg.Wait()
	return resolved, failed
}

func (s *importService) searchAndMatch(ctx context.Context, title string, year int) (int, bool) {
	resp, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:              title,
		PrimaryReleaseYear: year,
		Language:           "en-US",
	})
	if err == nil && resp != nil && len(resp.Results) > 0 {
		match, score := bestMatch(title, year, resp.Results)
		if score >= matchThreshold {
			return match.ID, true
		}
	}

	resp, err = s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    title,
		Language: "en-US",
	})
	if err == nil && resp != nil && len(resp.Results) > 0 {
		match, score := bestMatch(title, year, resp.Results)
		if score >= matchThreshold {
			return match.ID, true
		}
	}

	return 0, false
}


func findFileInZip(archive *zip.Reader, filename string) *zip.File {
	for _, f := range archive.File {
		if strings.EqualFold(f.Name, filename) {
			return f
		}
	}
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

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	return reader.ReadAll()
}

func validLetterboxdHeader(header []string, expected []string) bool {
	if len(header) < len(expected) {
		return false
	}
	for i, col := range expected {
		if !strings.EqualFold(strings.TrimSpace(header[i]), col) {
			return false
		}
	}
	return true
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

	if !validLetterboxdHeader(records[0], []string{"Date", "Name", "Year", "Letterboxd URI"}) {
		return nil
	}

	entries := make([]watchedEntry, 0, len(records)-1)
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

	if !validLetterboxdHeader(records[0], []string{"Date", "Name", "Year", "Letterboxd URI", "Rating"}) {
		return nil
	}

	entries := make([]ratingEntry, 0, len(records)-1)
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

	if !validLetterboxdHeader(records[0], []string{"Date", "Name", "Year", "Letterboxd URI", "Rating", "Rewatch", "Review"}) {
		return nil
	}

	entries := make([]reviewEntry, 0, len(records)-1)
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

	if !validLetterboxdHeader(records[0], []string{"Date", "Name", "Year", "Letterboxd URI"}) {
		return nil
	}

	entries := make([]watchlistEntry, 0, len(records)-1)
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
