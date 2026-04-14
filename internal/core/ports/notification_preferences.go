package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type NotificationPreferencesRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.NotificationPreferences, error)
	Upsert(ctx context.Context, prefs *domain.NotificationPreferences) error
}
