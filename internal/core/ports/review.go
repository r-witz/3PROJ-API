package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ReviewRepository interface {
	Create(ctx context.Context, review *domain.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Review, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Review, error)
	GetByTMDBID(ctx context.Context, tmdbID int) ([]*domain.Review, error)
	GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int) (*domain.Review, error)
	GetAverageRatingsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]float64, error)
	Update(ctx context.Context, review *domain.Review) error
	Delete(ctx context.Context, id uuid.UUID) error
}
