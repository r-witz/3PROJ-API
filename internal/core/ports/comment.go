package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	GetByReviewID(ctx context.Context, reviewID uuid.UUID, offset, limit int) ([]*domain.Comment, error)
	CountByReviewID(ctx context.Context, reviewID uuid.UUID) (int, error)
	CountByReviewIDs(ctx context.Context, reviewIDs []uuid.UUID) (map[uuid.UUID]int, error)
	Update(ctx context.Context, comment *domain.Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}
