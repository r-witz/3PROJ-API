package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ActivityFeedItem struct {
	Activity   *domain.Activity
	User       *domain.User
	Review     *domain.Review
	Collection *domain.Collection
	Comment    *domain.Comment
	TargetUser *domain.User
}

type ActivityService interface {
	GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int, types []domain.ActivityType) ([]*ActivityFeedItem, int, error)
	GetFeedForUser(ctx context.Context, userID uuid.UUID, offset, limit int, types []domain.ActivityType) ([]*ActivityFeedItem, int, error)
	Create(ctx context.Context, activity *domain.Activity) error
	DeleteByTypeAndReference(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error
}
