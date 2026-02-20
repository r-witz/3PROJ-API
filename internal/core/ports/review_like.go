package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ReviewLikeRepository interface {
	Create(ctx context.Context, like *domain.ReviewLike) error
	GetByReviewID(ctx context.Context, reviewID uuid.UUID) ([]*domain.ReviewLike, error)
	GetByUserIDAndReviewID(ctx context.Context, userID, reviewID uuid.UUID) (*domain.ReviewLike, error)
	Delete(ctx context.Context, userID, reviewID uuid.UUID) error
	CountByReviewID(ctx context.Context, reviewID uuid.UUID) (int, error)
	CountByReviewIDs(ctx context.Context, reviewIDs []uuid.UUID) (map[uuid.UUID]int, error)
	GetLikedByUser(ctx context.Context, userID uuid.UUID, reviewIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}
