package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type blockService struct {
	blockRepo  ports.BlockRepository
	followRepo ports.FollowRepository
	userRepo   ports.UserRepository
}

func NewBlockService(blockRepo ports.BlockRepository, followRepo ports.FollowRepository, userRepo ports.UserRepository) ports.BlockService {
	return &blockService{
		blockRepo:  blockRepo,
		followRepo: followRepo,
		userRepo:   userRepo,
	}
}

func (s *blockService) BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return domain.ErrCannotBlockSelf
	}

	user, err := s.userRepo.GetByID(ctx, blockedID)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return domain.ErrUserNotFound
	}

	existing, err := s.blockRepo.GetByBlockerAndBlocked(ctx, blockerID, blockedID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing != nil {
		return domain.ErrAlreadyBlocked
	}

	block := &domain.UserBlock{
		BlockerID: blockerID,
		BlockedID: blockedID,
		CreatedAt: time.Now(),
	}
	if err := s.blockRepo.Create(ctx, block); err != nil {
		return domain.ErrInternal
	}

	// Remove follows in both directions
	_ = s.followRepo.Delete(ctx, blockerID, blockedID)
	_ = s.followRepo.Delete(ctx, blockedID, blockerID)

	return nil
}

func (s *blockService) UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	existing, err := s.blockRepo.GetByBlockerAndBlocked(ctx, blockerID, blockedID)
	if err != nil {
		return domain.ErrInternal
	}
	if existing == nil {
		return domain.ErrNotBlocked
	}

	if err := s.blockRepo.Delete(ctx, blockerID, blockedID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *blockService) GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, offset, limit int) (*ports.BlockListResult, error) {
	blocks, total, err := s.blockRepo.GetBlockedByUser(ctx, blockerID, offset, limit)
	if err != nil {
		return nil, domain.ErrInternal
	}

	userIDs := make([]uuid.UUID, len(blocks))
	for i, b := range blocks {
		userIDs[i] = b.BlockedID
	}

	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, domain.ErrInternal
	}

	userMap := make(map[uuid.UUID]*domain.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	summaries := make([]*ports.BlockedUserSummary, 0, len(blocks))
	for _, b := range blocks {
		summaries = append(summaries, &ports.BlockedUserSummary{
			User:      userMap[b.BlockedID],
			BlockedAt: b.CreatedAt.Format(time.RFC3339),
		})
	}

	return &ports.BlockListResult{
		Users:  summaries,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

func (s *blockService) IsBlocked(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error) {
	return s.blockRepo.IsBlocked(ctx, userID1, userID2)
}

func (s *blockService) IsBlockedBy(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	return s.blockRepo.IsBlockedBy(ctx, blockerID, blockedID)
}

func (s *blockService) GetBlockerIDs(ctx context.Context, blockedID uuid.UUID) ([]uuid.UUID, error) {
	return s.blockRepo.GetBlockerIDs(ctx, blockedID)
}

func (s *blockService) GetBlockedIDs(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error) {
	return s.blockRepo.GetBlockedIDs(ctx, blockerID)
}
