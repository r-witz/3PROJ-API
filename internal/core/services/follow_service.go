package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type followService struct {
	followRepo   ports.FollowRepository
	userRepo     ports.UserRepository
	activityRepo ports.ActivityRepository
}

func NewFollowService(followRepo ports.FollowRepository, userRepo ports.UserRepository, activityRepo ports.ActivityRepository) ports.FollowService {
	return &followService{followRepo: followRepo, userRepo: userRepo, activityRepo: activityRepo}
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

	now := time.Now()
	follow := &domain.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
		CreatedAt:   now,
	}
	if err := s.followRepo.Create(ctx, follow); err != nil {
		return err
	}

	_ = s.activityRepo.Create(ctx, &domain.Activity{
		ID:           uuid.New(),
		UserID:       followerID,
		Type:         domain.ActivityTypeUserFollowed,
		TargetUserID: &followingID,
		CreatedAt:    now,
	})

	return nil
}

func (s *followService) Unfollow(ctx context.Context, followerID, followingID uuid.UUID) error {
	existing, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, followerID, followingID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFollowing
	}

	if err := s.followRepo.Delete(ctx, followerID, followingID); err != nil {
		return err
	}

	_ = s.activityRepo.Create(ctx, &domain.Activity{
		ID:           uuid.New(),
		UserID:       followerID,
		Type:         domain.ActivityTypeUserUnfollowed,
		TargetUserID: &followingID,
		CreatedAt:    time.Now(),
	})

	return nil
}

func (s *followService) RemoveFollower(ctx context.Context, userID, followerID uuid.UUID) error {
	existing, err := s.followRepo.GetByFollowerIDAndFollowingID(ctx, followerID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFollowing
	}

	return s.followRepo.Delete(ctx, followerID, userID)
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
