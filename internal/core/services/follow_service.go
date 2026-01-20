package services

import (
	"context"

	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type followService struct {
	followRepo ports.FollowRepository
}

func NewFollowService(followRepo ports.FollowRepository) ports.FollowService {
	return &followService{followRepo: followRepo}
}

func (s *followService) GetStats(ctx context.Context, userID uuid.UUID) (*ports.FollowStats, error) {
	followersCount, err := s.followRepo.CountFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}

	followingCount, err := s.followRepo.CountFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &ports.FollowStats{
		FollowersCount: followersCount,
		FollowingCount: followingCount,
	}, nil
}

func (s *followService) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	follow, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, followerID, followingID)
	if err != nil {
		return false, err
	}
	return follow != nil, nil
}
