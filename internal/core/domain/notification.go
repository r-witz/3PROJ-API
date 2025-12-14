package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeLikeReview  NotificationType = "like_review"
	NotificationTypeLikeComment NotificationType = "like_comment"
	NotificationTypeNewComment  NotificationType = "new_comment"
	NotificationTypeNewFollow   NotificationType = "new_follow"
	NotificationTypeSystem      NotificationType = "system"
)

type Notification struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"user_id" db:"user_id"`
	ActorID   *uuid.UUID       `json:"actor_id,omitempty" db:"actor_id"`
	Type      NotificationType `json:"type" db:"type"`
	ReviewID  *uuid.UUID       `json:"review_id,omitempty" db:"review_id"`
	CommentID *uuid.UUID       `json:"comment_id,omitempty" db:"comment_id"`
	Message   *string          `json:"message,omitempty" db:"message"`
	ReadAt    *time.Time       `json:"read_at,omitempty" db:"read_at"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}
