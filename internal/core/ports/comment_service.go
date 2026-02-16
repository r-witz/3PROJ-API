package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CreateCommentInput struct {
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"contains_spoilers"`
}

type UpdateCommentInput struct {
	Content          *string `json:"content"`
	ContainsSpoilers *bool   `json:"contains_spoilers"`
}

type CommentWithMeta struct {
	Comment     *domain.Comment
	LikeCount   int
	LikedByUser bool
}

type CommentService interface {
	Create(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID, input CreateCommentInput) (*domain.Comment, error)
	GetByReviewID(ctx context.Context, reviewID uuid.UUID, requestingUserID *uuid.UUID) ([]*CommentWithMeta, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateCommentInput) (*domain.Comment, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	Like(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error
	Unlike(ctx context.Context, commentID uuid.UUID, userID uuid.UUID) error
}
