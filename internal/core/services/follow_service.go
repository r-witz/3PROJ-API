package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type followService struct {
	followRepo ports.FollowRepository
	userRepo   ports.UserRepository
}

func NewFollowService(followRepo ports.FollowRepository, userRepo ports.UserRepository) ports.FollowService {
	return &followService{followRepo: followRepo, userRepo: userRepo}
}

func (s *followService) Follow(ctx context.Context, followerID, followingID uuid.UUID) error {
	if followerID == followingID {
		return domain.ErrCannotFollowSelf
	}

	targetUser, err := s.userRepo.GetByID(ctx, followingID)
	if err != nil {
		return err
	}
	if targetUser == nil {
		return domain.ErrUserNotFound
	}

	existing, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if existing != nil {
		return domain.ErrAlreadyFollowing
	}

	follow := &domain.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
		CreatedAt:   time.Now(),
	}
	return s.followRepo.Create(ctx, follow)
}

func (s *followService) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	existing, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFollowing
	}

	return s.followRepo.Delete(ctx, followerID, followingID)
}

func (s *followService) GetFollowers(ctx context.Context, userID uuid.UUID, search string, offset, limit int) (*ports.FollowListResult, error) {
	follows, total, err := s.followRepo.GetFollowersPaginated(ctx, userID, search, offset, limit)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(follows))
	for i, f := range follows {
		userIDs[i] = f.FollowerID
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	summaries := make([]*ports.FollowUserSummary, 0, len(follows))
	for _, f := range follows {
		if user, ok := userMap[f.FollowerID]; ok {
			summaries = append(summaries, &ports.FollowUserSummary{
				User:       user,
				FollowedAt: f.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return &ports.FollowListResult{
		Users:  summaries,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

func (s *followService) GetFollowing(ctx context.Context, userID uuid.UUID, search string, offset, limit int) (*ports.FollowListResult, error) {
	follows, total, err := s.followRepo.GetFollowingPaginated(ctx, userID, search, offset, limit)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, len(follows))
	for i, f := range follows {
		userIDs[i] = f.FollowingID
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	summaries := make([]*ports.FollowUserSummary, 0, len(follows))
	for _, f := range follows {
		if user, ok := userMap[f.FollowingID]; ok {
			summaries = append(summaries, &ports.FollowUserSummary{
				User:       user,
				FollowedAt: f.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return &ports.FollowListResult{
		Users:  summaries,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
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
