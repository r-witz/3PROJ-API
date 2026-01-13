package services

import (
	"context"

	"github.com/google/uuid"
)

type FollowStats struct {
	FollowersCount int
	FollowingCount int
}

type FollowService interface {
	GetStats(ctx context.Context, userID uuid.UUID) (*FollowStats, error)
	IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error)
}
