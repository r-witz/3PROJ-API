package domain

import (
	"time"

	"github.com/google/uuid"
)

type NotificationPreferences struct {
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	LikeReview          bool      `json:"like_review" db:"like_review"`
	LikeComment         bool      `json:"like_comment" db:"like_comment"`
	NewComment          bool      `json:"new_comment" db:"new_comment"`
	NewFollow           bool      `json:"new_follow" db:"new_follow"`
	System              bool      `json:"system" db:"system"`
	AchievementUnlocked bool      `json:"achievement_unlocked" db:"achievement_unlocked"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

func DefaultNotificationPreferences(userID uuid.UUID) *NotificationPreferences {
	return &NotificationPreferences{
		UserID:              userID,
		LikeReview:          true,
		LikeComment:         true,
		NewComment:          true,
		NewFollow:           true,
		System:              true,
		AchievementUnlocked: true,
		UpdatedAt:           time.Now(),
	}
}

func (p *NotificationPreferences) IsEnabled(notifType NotificationType) bool {
	switch notifType {
	case NotificationTypeLikeReview:
		return p.LikeReview
	case NotificationTypeLikeComment:
		return p.LikeComment
	case NotificationTypeNewComment:
		return p.NewComment
	case NotificationTypeNewFollow:
		return p.NewFollow
	case NotificationTypeSystem:
		return p.System
	case NotificationTypeAchievementUnlocked:
		return p.AchievementUnlocked
	default:
		return false
	}
}
