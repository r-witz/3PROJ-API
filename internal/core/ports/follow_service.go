package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type FollowStats struct {
	FollowersCount int
	FollowingCount int
}

type FollowUserSummary struct {
	User      *domain.User
	FollowedAt string
}

type FollowListResult struct {
	Users  []*FollowUserSummary
	Total  int
	Offset int
	Limit  int
}

type FollowService interface {
	Follow(ctx context.Context, followerID, followingID uuid.UUID) error
	Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error
	RemoveFollower(ctx context.Context, userID, followerID uuid.UUID) error
	GetFollowers(ctx context.Context, userID uuid.UUID, search string, offset, limit int) (*FollowListResult, error)
	GetFollowing(ctx context.Context, userID uuid.UUID, search string, offset, limit int) (*FollowListResult, error)
	GetStats(ctx context.Context, userID uuid.UUID) (*FollowStats, error)
	IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
}
