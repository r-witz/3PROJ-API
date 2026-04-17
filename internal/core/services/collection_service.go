package services

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9-]+`)

type collectionService struct {
	collectionRepo     ports.CollectionRepository
	collectionItemRepo ports.CollectionItemRepository
	tmdbClient         ports.TMDBClient
	reviewRepo         ports.ReviewRepository
}

func NewCollectionService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	tmdbClient ports.TMDBClient,
	reviewRepo ports.ReviewRepository,
) ports.CollectionService {
	return &collectionService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		tmdbClient:         tmdbClient,
		reviewRepo:         reviewRepo,
	}
}

func (s *collectionService) CreateDefaultCollections(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()

	watched := &domain.Collection{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "Watched",
		Slug:       "watched",
		Type:       domain.CollectionTypeSystem,
		Visibility: domain.CollectionVisibilityPublic,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	toWatch := &domain.Collection{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "To Watch",
		Slug:       "to-watch",
		Type:       domain.CollectionTypeSystem,
		Visibility: domain.CollectionVisibilityPublic,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.collectionRepo.Create(ctx, watched); err != nil {
		return domain.ErrInternal
	}

	if err := s.collectionRepo.Create(ctx, toWatch); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *collectionService) Create(ctx context.Context, userID uuid.UUID, input ports.CreateCollectionInput) (*domain.Collection, error) {
	if input.Name == "" {
		return nil, domain.ErrInvalidInput
	}

	slug := generateSlug(input.Name)
	if slug == "" {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, domain.ErrCollectionAlreadyExists
	}

	visibility := domain.CollectionVisibilityPrivate
	if input.Visibility == string(domain.CollectionVisibilityPublic) {
		visibility = domain.CollectionVisibilityPublic
	}

	now := time.Now()
	collection := &domain.Collection{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        input.Name,
		Slug:        slug,
		Type:        domain.CollectionTypeCustom,
		Visibility:  visibility,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.collectionRepo.Create(ctx, collection); err != nil {
		return nil, domain.ErrInternal
	}

	return collection, nil
}

func (s *collectionService) GetBySlug(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID) (*ports.CollectionWithPresence, error) {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if collection == nil {
		return nil, domain.ErrCollectionNotFound
	}

	if !canViewCollection(collection, requestingUserID) {
		return nil, domain.ErrCollectionNotFound
	}

	count, err := s.collectionItemRepo.CountByCollectionID(ctx, collection.ID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.CollectionWithPresence{
		Collection: collection,
		ItemCount:  count,
	}, nil
}

func (s *collectionService) GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID, collectionType *domain.CollectionType) ([]ports.CollectionWithPresence, error) {
	collections, err := s.collectionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	isOwner := requestingUserID != nil && *requestingUserID == userID

	var visible []*domain.Collection
	for _, c := range collections {
		if !isOwner && c.Visibility != domain.CollectionVisibilityPublic {
			continue
		}
		if collectionType != nil && c.Type != *collectionType {
			continue
		}
		visible = append(visible, c)
	}

	ids := make([]uuid.UUID, len(visible))
	for i, c := range visible {
		ids[i] = c.ID
	}

	counts, err := s.collectionItemRepo.CountByCollectionIDs(ctx, ids)
	if err != nil {
		return nil, domain.ErrInternal
	}

	result := make([]ports.CollectionWithPresence, len(visible))
	for i, c := range visible {
		result[i] = ports.CollectionWithPresence{
			Collection: c,
			ItemCount:  counts[c.ID],
		}
	}
	return result, nil
}

func (s *collectionService) GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int, requestingUserID *uuid.UUID, collectionType *domain.CollectionType) ([]ports.CollectionWithPresence, error) {
	allCollections, err := s.collectionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	matchingCollections, err := s.collectionRepo.GetByUserIDAndTMDBID(ctx, userID, tmdbID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	matchingIDs := make(map[uuid.UUID]bool, len(matchingCollections))
	for _, c := range matchingCollections {
		matchingIDs[c.ID] = true
	}

	isOwner := requestingUserID != nil && *requestingUserID == userID

	var visible []*domain.Collection
	for _, c := range allCollections {
		if !isOwner && c.Visibility != domain.CollectionVisibilityPublic {
			continue
		}
		if collectionType != nil && c.Type != *collectionType {
			continue
		}
		visible = append(visible, c)
	}

	ids := make([]uuid.UUID, len(visible))
	for i, c := range visible {
		ids[i] = c.ID
	}

	counts, err := s.collectionItemRepo.CountByCollectionIDs(ctx, ids)
	if err != nil {
		return nil, domain.ErrInternal
	}

	result := make([]ports.CollectionWithPresence, len(visible))
	for i, c := range visible {
		result[i] = ports.CollectionWithPresence{
			Collection: c,
			HasMovie:   matchingIDs[c.ID],
			ItemCount:  counts[c.ID],
		}
	}
	return result, nil
}

func (s *collectionService) Update(ctx context.Context, userID uuid.UUID, slug string, input ports.UpdateCollectionInput) (*domain.Collection, error) {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if collection == nil {
		return nil, domain.ErrCollectionNotFound
	}

	if collection.Type == domain.CollectionTypeSystem && input.Name != nil {
		return nil, domain.ErrCannotModifySystemCollection
	}

	if input.Name != nil && *input.Name != "" {
		newSlug := generateSlug(*input.Name)

		existing, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, newSlug)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if existing != nil && existing.ID != collection.ID {
			return nil, domain.ErrCollectionAlreadyExists
		}

		collection.Name = *input.Name
		collection.Slug = newSlug
	}

	if input.Description != nil {
		collection.Description = input.Description
	}

	if input.Visibility != nil {
		if *input.Visibility == string(domain.CollectionVisibilityPublic) {
			collection.Visibility = domain.CollectionVisibilityPublic
		} else {
			collection.Visibility = domain.CollectionVisibilityPrivate
		}
	}

	collection.UpdatedAt = time.Now()

	if err := s.collectionRepo.Update(ctx, collection); err != nil {
		return nil, domain.ErrInternal
	}

	return collection, nil
}

func (s *collectionService) Delete(ctx context.Context, userID uuid.UUID, slug string) error {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return domain.ErrInternal
	}
	if collection == nil {
		return domain.ErrCollectionNotFound
	}

	if collection.Type == domain.CollectionTypeSystem {
		return domain.ErrCannotDeleteSystemCollection
	}

	if err := s.collectionRepo.Delete(ctx, collection.ID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *collectionService) AddItem(ctx context.Context, userID uuid.UUID, slug string, tmdbID int) (*domain.CollectionItem, error) {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if collection == nil {
		return nil, domain.ErrCollectionNotFound
	}

	existing, err := s.collectionItemRepo.GetByCollectionIDAndTMDBID(ctx, collection.ID, tmdbID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, domain.ErrCollectionItemAlreadyExists
	}

	var runtime int16
	details, err := s.tmdbClient.GetMovieDetails(ctx, tmdbID, "en-US")
	if err == nil && details != nil && details.Runtime != nil {
		runtime = int16(*details.Runtime)
	}

	item := &domain.CollectionItem{
		CollectionID: collection.ID,
		TMDBID:       tmdbID,
		AddedAt:      time.Now(),
		Runtime:      runtime,
		Metadata:     json.RawMessage("{}"),
	}

	if err := s.collectionItemRepo.Create(ctx, item); err != nil {
		return nil, domain.ErrInternal
	}

	return item, nil
}

func (s *collectionService) RemoveItem(ctx context.Context, userID uuid.UUID, slug string, tmdbID int) error {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return domain.ErrInternal
	}
	if collection == nil {
		return domain.ErrCollectionNotFound
	}

	existing, err := s.collectionItemRepo.GetByCollectionIDAndTMDBID(ctx, collection.ID, tmdbID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing == nil {
		return domain.ErrCollectionItemNotFound
	}

	if err := s.collectionItemRepo.Delete(ctx, collection.ID, tmdbID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *collectionService) GetItems(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID, offset, limit int, language string, sortOpt ports.CollectionItemSort) ([]ports.MovieSearchResult, int, error) {
	collection, err := s.collectionRepo.GetByUserIDAndSlug(ctx, userID, slug)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}
	if collection == nil {
		return nil, 0, domain.ErrCollectionNotFound
	}

	if !canViewCollection(collection, requestingUserID) {
		return nil, 0, domain.ErrCollectionNotFound
	}

	sortField := sortOpt.Field
	if sortField == "" {
		sortField = ports.CollectionItemSortByAddedAt
	}

	total, err := s.collectionItemRepo.CountByCollectionID(ctx, collection.ID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	// Fast path: default sort by added_at uses SQL-level pagination.
	if sortField == ports.CollectionItemSortByAddedAt && !sortOpt.Asc {
		items, err := s.collectionItemRepo.GetByCollectionIDPaginated(ctx, collection.ID, offset, limit)
		if err != nil {
			return nil, 0, domain.ErrInternal
		}
		result, err := s.buildCollectionItemResults(ctx, items, userID, requestingUserID, language)
		if err != nil {
			return nil, 0, err
		}
		return result, total, nil
	}

	items, err := s.collectionItemRepo.GetByCollectionID(ctx, collection.ID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	if len(items) == 0 {
		return []ports.MovieSearchResult{}, total, nil
	}

	results, err := s.buildCollectionItemResults(ctx, items, userID, requestingUserID, language)
	if err != nil {
		return nil, 0, err
	}

	viewerRatings := map[int]float64{}
	if sortField == ports.CollectionItemSortByOurRating && requestingUserID != nil {
		tmdbIDs := make([]int, len(items))
		for i, item := range items {
			tmdbIDs[i] = item.TMDBID
		}
		vr, err := s.reviewRepo.GetRatingsByUserIDAndTMDBIDs(ctx, *requestingUserID, tmdbIDs)
		if err == nil {
			viewerRatings = vr
		}
	}

	addedAtByID := make(map[int]time.Time, len(items))
	for _, item := range items {
		addedAtByID[item.TMDBID] = item.AddedAt
	}

	sortCollectionItemResults(results, addedAtByID, viewerRatings, sortOpt)

	if offset >= len(results) {
		return []ports.MovieSearchResult{}, total, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}
	return results[offset:end], total, nil
}

func (s *collectionService) buildCollectionItemResults(ctx context.Context, items []*domain.CollectionItem, userID uuid.UUID, requestingUserID *uuid.UUID, language string) ([]ports.MovieSearchResult, error) {
	if len(items) == 0 {
		return []ports.MovieSearchResult{}, nil
	}

	tmdbIDs := make([]int, len(items))
	for i, item := range items {
		tmdbIDs[i] = item.TMDBID
	}

	type movieInfo struct {
		title      string
		poster     *string
		date       string
		tmdbRating *float64
	}
	movieInfos := make([]movieInfo, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for i, item := range items {
		wg.Add(1)
		go func(idx int, tmdbID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			details, err := s.tmdbClient.GetMovieDetails(ctx, tmdbID, language)
			if err != nil || details == nil {
				return
			}

			movieInfos[idx] = movieInfo{
				title:  details.Title,
				poster: details.PosterPath,
				date:   details.ReleaseDate,
			}
			if details.VoteCount > 0 {
				rating := details.VoteAverage / 2
				movieInfos[idx].tmdbRating = &rating
			}
		}(i, item.TMDBID)
	}
	wg.Wait()

	ratings, err := s.reviewRepo.GetAverageRatingsByTMDBIDs(ctx, tmdbIDs)
	if err != nil {
		ratings = make(map[int]float64)
	}

	ownerRatings, err := s.reviewRepo.GetRatingsByUserIDAndTMDBIDs(ctx, userID, tmdbIDs)
	if err != nil {
		ownerRatings = make(map[int]float64)
	}

	result := make([]ports.MovieSearchResult, len(items))
	for i, item := range items {
		var duskforgeRating *float64
		if r, ok := ratings[item.TMDBID]; ok {
			duskforgeRating = &r
		}

		var ownerRating *float64
		if r, ok := ownerRatings[item.TMDBID]; ok {
			ownerRating = &r
		}

		result[i] = ports.MovieSearchResult{
			ID:              item.TMDBID,
			Poster:          movieInfos[i].poster,
			Name:            movieInfos[i].title,
			Date:            movieInfos[i].date,
			TMDBRating:      movieInfos[i].tmdbRating,
			DuskforgeRating: duskforgeRating,
			UserRating:      ownerRating,
		}
	}

	return result, nil
}

func sortCollectionItemResults(results []ports.MovieSearchResult, addedAtByID map[int]time.Time, viewerRatings map[int]float64, sortOpt ports.CollectionItemSort) {
	viewerRatingPtr := func(id int) *float64 {
		if r, ok := viewerRatings[id]; ok {
			return &r
		}
		return nil
	}

	sort.SliceStable(results, func(i, j int) bool {
		switch sortOpt.Field {
		case ports.CollectionItemSortByReleaseDate:
			return cmpString(results[i].Date, results[j].Date, sortOpt.Asc)
		case ports.CollectionItemSortByTMDBRating:
			return cmpFloatPtr(results[i].TMDBRating, results[j].TMDBRating, sortOpt.Asc)
		case ports.CollectionItemSortByDuskforgeRating:
			return cmpFloatPtr(results[i].DuskforgeRating, results[j].DuskforgeRating, sortOpt.Asc)
		case ports.CollectionItemSortByCollectionRating:
			return cmpFloatPtr(results[i].UserRating, results[j].UserRating, sortOpt.Asc)
		case ports.CollectionItemSortByOurRating:
			return cmpFloatPtr(viewerRatingPtr(results[i].ID), viewerRatingPtr(results[j].ID), sortOpt.Asc)
		default:
			return cmpTime(addedAtByID[results[i].ID], addedAtByID[results[j].ID], sortOpt.Asc)
		}
	})
}

// cmpFloatPtr reports whether a should come before b. Nil values always sink to the bottom regardless of direction.
func cmpFloatPtr(a, b *float64, asc bool) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	if asc {
		return *a < *b
	}
	return *a > *b
}

// cmpString orders empty strings last regardless of direction.
func cmpString(a, b string, asc bool) bool {
	if a == "" && b == "" {
		return false
	}
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	if asc {
		return a < b
	}
	return a > b
}

func cmpTime(a, b time.Time, asc bool) bool {
	if asc {
		return a.Before(b)
	}
	return a.After(b)
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = slugRegex.ReplaceAllString(slug, "")
	slug = strings.Trim(slug, "-")
	return slug
}

func canViewCollection(collection *domain.Collection, requestingUserID *uuid.UUID) bool {
	if requestingUserID != nil && *requestingUserID == collection.UserID {
		return true
	}
	return collection.Visibility == domain.CollectionVisibilityPublic
}
