package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CollectionItemRepository interface {
	Create(ctx context.Context, item *domain.CollectionItem) error
	GetByCollectionID(ctx context.Context, collectionID uuid.UUID) ([]*domain.CollectionItem, error)
	GetByCollectionIDPaginated(ctx context.Context, collectionID uuid.UUID, offset, limit int) ([]*domain.CollectionItem, error)
	CountByCollectionID(ctx context.Context, collectionID uuid.UUID) (int, error)
	GetByCollectionIDAndTMDBID(ctx context.Context, collectionID uuid.UUID, tmdbID int) (*domain.CollectionItem, error)
	Delete(ctx context.Context, collectionID uuid.UUID, tmdbID int) error
}
