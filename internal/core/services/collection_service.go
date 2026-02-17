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
}

func NewCollectionService(
	collectionRepo ports.CollectionRepository,
	collectionItemRepo ports.CollectionItemRepository,
	tmdbClient ports.TMDBClient,
) ports.CollectionService {
	return &collectionService{
		collectionRepo:     collectionRepo,
		collectionItemRepo: collectionItemRepo,
		tmdbClient:         tmdbClient,
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
		Visibility: domain.CollectionVisibilityPrivate,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	toWatch := &domain.Collection{
		ID:         uuid.New(),
		UserID:     userID,
		Name:       "To Watch",
		Slug:       "to-watch",
		Type:       domain.CollectionTypeSystem,
		Visibility: domain.CollectionVisibilityPrivate,
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

func (s *collectionService) GetBySlug(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID) (*domain.Collection, error) {
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

	return collection, nil
}

func (s *collectionService) GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID) ([]*domain.Collection, error) {
	collections, err := s.collectionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	isOwner := requestingUserID != nil && *requestingUserID == userID

	if isOwner {
		return collections, nil
	}

	var visible []*domain.Collection
	for _, c := range collections {
		if c.Visibility == domain.CollectionVisibilityPublic {
			visible = append(visible, c)
		}
	}
	return visible, nil
}

func (s *collectionService) GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int, requestingUserID *uuid.UUID) ([]*domain.Collection, error) {
	collections, err := s.collectionRepo.GetByUserIDAndTMDBID(ctx, userID, tmdbID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	isOwner := requestingUserID != nil && *requestingUserID == userID

	if isOwner {
		return collections, nil
	}

	var visible []*domain.Collection
	for _, c := range collections {
		if c.Visibility == domain.CollectionVisibilityPublic {
			visible = append(visible, c)
		}
	}
	return visible, nil
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

	// Fetch runtime from TMDB
	var runtime int16
	details, err := s.tmdbClient.GetMovieDetails(ctx, tmdbID, "en")
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

func (s *collectionService) GetItems(ctx context.Context, userID uuid.UUID, slug string, requestingUserID *uuid.UUID, offset, limit int, language string) ([]*ports.CollectionItemWithDetails, int, error) {
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

	// Enrich items with TMDB details concurrently
	result := make([]*ports.CollectionItemWithDetails, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, item := range items {
		wg.Add(1)
		go func(idx int, it *domain.CollectionItem) {
			defer wg.Done()

			detail := &ports.CollectionItemWithDetails{
				Item: it,
			}

			details, err := s.tmdbClient.GetMovieDetails(ctx, it.TMDBID, language)
			if err == nil && details != nil {
				detail.Title = details.Title
				detail.Poster = details.PosterPath
				detail.ReleaseDate = details.ReleaseDate
				detail.Runtime = details.Runtime
			}

			mu.Lock()
			result[idx] = detail
			mu.Unlock()
		}(i, item)
	}

	wg.Wait()

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
