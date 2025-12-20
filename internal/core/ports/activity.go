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
	GetFeedForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Activity, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
