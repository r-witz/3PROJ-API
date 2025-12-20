package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CollectionItemRepository interface {
	Create(ctx context.Context, item *domain.CollectionItem) error
	GetByCollectionID(ctx context.Context, collectionID uuid.UUID) ([]*domain.CollectionItem, error)
	GetByCollectionIDAndTMDBID(ctx context.Context, collectionID uuid.UUID, tmdbID int) (*domain.CollectionItem, error)
	Delete(ctx context.Context, collectionID uuid.UUID, tmdbID int) error
}
