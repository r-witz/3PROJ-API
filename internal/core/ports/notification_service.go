package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type NotifyInput struct {
	UserID    uuid.UUID
	ActorID   uuid.UUID
	Type      domain.NotificationType
	ReviewID  *uuid.UUID
	CommentID *uuid.UUID
	Message   *string
}

type UpdateNotificationPreferencesInput struct {
	LikeReview  *bool
	LikeComment *bool
	NewComment  *bool
	NewFollow   *bool
	System      *bool
}

type NotificationService interface {
	Notify(ctx context.Context, input NotifyInput) (*domain.Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, int, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkAsRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	Delete(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*domain.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, input UpdateNotificationPreferencesInput) (*domain.NotificationPreferences, error)
}
