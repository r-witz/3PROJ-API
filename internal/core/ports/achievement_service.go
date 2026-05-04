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

	// Family is a stable identifier for the progression ladder this entry
	// belongs to. Set by List so frontends can key each row to its ladder;
	// left empty on endpoints that return individual badges.
	Family string
}

type UnlockedAchievement struct {
	Achievement *domain.Achievement
	UnlockedAt  *domain.UserAchievement
}

type AchievementService interface {
	// Read — requesterID determines unlock/progress and secret visibility.
	List(ctx context.Context, requesterID *uuid.UUID, category *domain.AchievementCategory) ([]*AchievementWithProgress, error)
	GetByID(ctx context.Context, id uuid.UUID, requesterID *uuid.UUID) (*AchievementWithProgress, error)
	ListUnlockedByUser(ctx context.Context, userID uuid.UUID) ([]*UnlockedAchievement, error)
	ListRecentUnlocksByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*UnlockedAchievement, error)

	// Evaluation — called from activity middleware post-handler.
	EvaluateForEvent(ctx context.Context, userID uuid.UUID, category domain.AchievementCategory) ([]*domain.Achievement, error)

	// EvaluateAllForUser runs evaluation across every category. Use this from
	// flows that bypass the activity middleware (bulk imports, admin backfills)
	// so any newly-eligible badges unlock.
	EvaluateAllForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Achievement, error)
}
