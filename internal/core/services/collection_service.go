package services

import (
	"context"
	"encoding/json"
	"regexp"
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
	activityRepo       ports.ActivityRepository
}

func NewCollectionService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	tmdbClient ports.TMDBClient,
	reviewRepo ports.ReviewRepository,
	activityRepo ports.ActivityRepository,
) ports.CollectionService {
	return &collectionService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		tmdbClient:         tmdbClient,
		reviewRepo:         reviewRepo,
		activityRepo:       activityRepo,
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

	_ = s.activityRepo.Create(ctx, &domain.Activity{
		ID:           uuid.New(),
		UserID:       userID,
		Type:         domain.ActivityTypeCollectionCreated,
		CollectionID: &collection.ID,
		CreatedAt:    now,
	})

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

	_ = s.activityRepo.DeleteByTypeAndReference(ctx, userID, domain.ActivityTypeCollectionCreated, nil, &collection.ID, nil, nil)

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

	if collection.Type == domain.CollectionTypeSystem && collection.Slug == "to-watch" {
		_ = s.activityRepo.Create(ctx, &domain.Activity{
			ID:           uuid.New(),
			UserID:       userID,
			Type:         domain.ActivityTypeWatchlistItemAdded,
			CollectionID: &collection.ID,
			TMDBID:       &tmdbID,
			CreatedAt:    item.AddedAt,
		})
	} else if collection.Type != domain.CollectionTypeSystem {
		_ = s.activityRepo.Create(ctx, &domain.Activity{
			ID:           uuid.New(),
			UserID:       userID,
			Type:         domain.ActivityTypeCollectionItemAdded,
			CollectionID: &collection.ID,
			TMDBID:       &tmdbID,
			CreatedAt:    item.AddedAt,
		})
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

	if collection.Type == domain.CollectionTypeSystem && collection.Slug == "to-watch" {
		_ = s.activityRepo.DeleteByTypeAndReference(ctx, userID, domain.ActivityTypeWatchlistItemAdded, nil, &collection.ID, nil, &tmdbID)
	} else if collection.Type != domain.CollectionTypeSystem {
		_ = s.activityRepo.DeleteByTypeAndReference(ctx, userID, domain.ActivityTypeCollectionItemAdded, nil, &collection.ID, nil, &tmdbID)
	}

	return nil
}

func (s *collectionService) GetItems(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID, offset, limit int, language string) ([]ports.MovieSearchResult, int, error) {
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

	items, err := s.collectionItemRepo.GetByCollectionIDPaginated(ctx, collection.ID, offset, limit)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.collectionItemRepo.CountByCollectionID(ctx, collection.ID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	if len(items) == 0 {
		return []ports.MovieSearchResult{}, total, nil
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

	result := make([]ports.MovieSearchResult, len(items))
	for i, item := range items {
		var duskforgeRating *float64
		if r, ok := ratings[item.TMDBID]; ok {
			duskforgeRating = &r
		}

		result[i] = ports.MovieSearchResult{
			ID:              item.TMDBID,
			Poster:          movieInfos[i].poster,
			Name:            movieInfos[i].title,
			Date:            movieInfos[i].date,
			TMDBRating:      movieInfos[i].tmdbRating,
			DuskforgeRating: duskforgeRating,
		}
	}

	return result, total, nil
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
