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
}
