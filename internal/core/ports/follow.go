package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type FollowRepository interface {
	Create(ctx context.Context, follow *domain.Follow) error
	GetFollowers(ctx context.Context, userID uuid.UUID) ([]*domain.Follow, error)
	GetFollowing(ctx context.Context, userID uuid.UUID) ([]*domain.Follow, error)
	GetFollowersPaginated(ctx context.Context, userID uuid.UUID, search string, offset, limit int) ([]*domain.Follow, int, error)
	GetFollowingPaginated(ctx context.Context, userID uuid.UUID, search string, offset, limit int) ([]*domain.Follow, int, error)
	GetByFollowerIDAndFollowingID(ctx context.Context, followerID, followingID uuid.UUID) (*domain.Follow, error)
	Delete(ctx context.Context, followerID, followingID uuid.UUID) error
	CountFollowers(ctx context.Context, userID uuid.UUID) (int, error)
	CountFollowing(ctx context.Context, userID uuid.UUID) (int, error)
}
