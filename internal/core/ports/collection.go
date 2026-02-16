package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CollectionRepository interface {
	Create(ctx context.Context, collection *domain.Collection) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Collection, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Collection, error)
	GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int) ([]*domain.Collection, error)
	GetByUserIDAndSlug(ctx context.Context, userID uuid.UUID, slug string) (*domain.Collection, error)
	Update(ctx context.Context, collection *domain.Collection) error
	Delete(ctx context.Context, id uuid.UUID) error
}
