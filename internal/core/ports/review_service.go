package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CreateReviewInput struct {
	Rating           float64 `json:"rating"`
	Content          *string `json:"content"`
	ContainsSpoilers bool    `json:"contains_spoilers"`
}

type UpdateReviewInput struct {
	Rating           *float64 `json:"rating"`
	Content          *string  `json:"content"`
	ContainsSpoilers *bool    `json:"contains_spoilers"`
}

type ReviewWithMeta struct {
	Review       *domain.Review
	LikeCount    int
	LikedByUser  bool
	CommentCount int
	User         *domain.User
}

type ReviewService interface {
	Create(ctx context.Context, userID uuid.UUID, tmdbID int, input CreateReviewInput) (*domain.Review, error)
	GetByID(ctx context.Context, id uuid.UUID, requestingUserID *uuid.UUID) (*ReviewWithMeta, error)
	GetByTMDBID(ctx context.Context, tmdbID int, requestingUserID *uuid.UUID, offset, limit int, sort ReviewSort) ([]*ReviewWithMeta, int, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, requestingUserID *uuid.UUID, offset, limit int, sort ReviewSort) ([]*ReviewWithMeta, int, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateReviewInput) (*ReviewWithMeta, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	Like(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID) error
	Unlike(ctx context.Context, reviewID uuid.UUID, userID uuid.UUID) error
}
