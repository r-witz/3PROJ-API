package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type AchievementProgress struct {
	Current int `json:"current"`
	Target  int `json:"target"`
}

type AchievementWithProgress struct {
	Achievement *domain.Achievement
	Unlocked    bool
	UnlockedAt  *domain.UserAchievement
	Progress    AchievementProgress

	Family string
}

type UnlockedAchievement struct {
	Achievement *domain.Achievement
	UnlockedAt  *domain.UserAchievement
}

type AchievementService interface {
	List(ctx context.Context, requesterID *uuid.UUID, category *domain.AchievementCategory) ([]*AchievementWithProgress, error)
	GetByID(ctx context.Context, id uuid.UUID, requesterID *uuid.UUID) (*AchievementWithProgress, error)
	ListUnlockedByUser(ctx context.Context, userID uuid.UUID) ([]*UnlockedAchievement, error)
	ListRecentUnlocksByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*UnlockedAchievement, error)

	EvaluateForEvent(ctx context.Context, userID uuid.UUID, category domain.AchievementCategory) ([]*domain.Achievement, error)

	EvaluateAllForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Achievement, error)
}
