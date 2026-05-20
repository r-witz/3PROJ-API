package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type AchievementListFilter struct {
	Category *domain.AchievementCategory
	OnlyActive bool
}

type AchievementRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Achievement, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Achievement, error)
	List(ctx context.Context, filter AchievementListFilter) ([]*domain.Achievement, error)

	GetUnlockedIDsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]struct{}, error)
	GetUnlockedByUser(ctx context.Context, userID uuid.UUID) ([]*domain.UserAchievement, error)
	GetRecentUnlocksByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UserAchievement, error)
	CountUnlockedByUser(ctx context.Context, userID uuid.UUID) (int, error)

	Unlock(ctx context.Context, userID, achievementID uuid.UUID) (bool, error)

	CountCommentsByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountWrittenReviewsByUser(ctx context.Context, userID uuid.UUID) (int, error)
	CountReviewsByUserWithRating(ctx context.Context, userID uuid.UUID, rating float64) (int, error)
	CountCustomCollectionsByUser(ctx context.Context, userID uuid.UUID) (int, error)
}
