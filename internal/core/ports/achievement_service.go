package ports

import (
	"context"
	"encoding/json"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type CreateAchievementInput struct {
	Code        string
	Name        string
	Description string
	Category    domain.AchievementCategory
	Tier        domain.AchievementTier
	IconURL     *string
	Criterion   json.RawMessage
	Secret      bool
	Active      bool
	SortOrder   int
}

type UpdateAchievementInput struct {
	Name        *string
	Description *string
	Category    *domain.AchievementCategory
	Tier        *domain.AchievementTier
	IconURL     *string
	Criterion   json.RawMessage
	Secret      *bool
	Active      *bool
	SortOrder   *int
}

type AchievementProgress struct {
	Current int `json:"current"`
	Target  int `json:"target"`
}

type AchievementWithProgress struct {
	Achievement *domain.Achievement
	Unlocked    bool
	UnlockedAt  *domain.UserAchievement
	Progress    AchievementProgress
}

type UnlockedAchievement struct {
	Achievement *domain.Achievement
	UnlockedAt  *domain.UserAchievement
}

type AchievementService interface {
	// Catalog CRUD (admin-only callers enforced at route layer).
	Create(ctx context.Context, input CreateAchievementInput) (*domain.Achievement, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateAchievementInput) (*domain.Achievement, error)
	Delete(ctx context.Context, id uuid.UUID) error

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
