package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CommentLikeRepository interface {
	Create(ctx context.Context, like *domain.CommentLike) error
	GetByCommentID(ctx context.Context, commentID uuid.UUID) ([]*domain.CommentLike, error)
	GetByUserIDAndCommentID(ctx context.Context, userID, commentID uuid.UUID) (*domain.CommentLike, error)
	Delete(ctx context.Context, userID, commentID uuid.UUID) error
	CountByCommentID(ctx context.Context, commentID uuid.UUID) (int, error)
}
