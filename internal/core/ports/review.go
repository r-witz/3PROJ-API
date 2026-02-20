package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type RatingStats struct {
	Rating float64
	Count  int
}

type ReviewSortField string

const (
	ReviewSortByCreatedAt ReviewSortField = "created_at"
	ReviewSortByLikes     ReviewSortField = "likes"
	ReviewSortByRating    ReviewSortField = "rating"
)

type ReviewSort struct {
	Field ReviewSortField
	Asc   bool
}

type ReviewRepository interface {
	Create(ctx context.Context, review *domain.Review) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Review, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, tmdbID *int, offset, limit int, sort ReviewSort) ([]*domain.Review, error)
	GetByTMDBID(ctx context.Context, tmdbID int, offset, limit int, sort ReviewSort) ([]*domain.Review, error)
	GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int) (*domain.Review, error)
	GetAverageRatingsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]float64, error)
	GetRatingStatsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]RatingStats, error)
	CountByTMDBID(ctx context.Context, tmdbID int) (int, error)
	CountByUserID(ctx context.Context, userID uuid.UUID, tmdbID *int) (int, error)
	Update(ctx context.Context, review *domain.Review) error
	Delete(ctx context.Context, id uuid.UUID) error
}
