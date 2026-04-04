package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ActivityRepository interface {
	Create(ctx context.Context, activity *domain.Activity) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Activity, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Activity, error)
	GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int, types []domain.ActivityType) ([]*domain.Activity, error)
	GetFeedForUser(ctx context.Context, userID uuid.UUID, limit, offset int, types []domain.ActivityType) ([]*domain.Activity, error)
	CountByUserID(ctx context.Context, userID uuid.UUID, types []domain.ActivityType) (int, error)
	CountFeedForUser(ctx context.Context, userID uuid.UUID, types []domain.ActivityType) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTypeAndReference(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error
	DeleteByFields(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error
}
