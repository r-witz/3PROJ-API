package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Notification, error)
	GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, error)
	GetUnreadByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Notification, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	Update(ctx context.Context, notification *domain.Notification) error
	Delete(ctx context.Context, id uuid.UUID) error
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}
