package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type BlockedUserSummary struct {
	User      *domain.User
	BlockedAt string
}

type BlockListResult struct {
	Users  []*BlockedUserSummary
	Total  int
	Offset int
	Limit  int
}

type BlockService interface {
	BlockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID, blockedID uuid.UUID) error
	GetBlockedUsers(ctx context.Context, blockerID uuid.UUID, offset, limit int) (*BlockListResult, error)
	IsBlocked(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error)
	IsBlockedBy(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	GetBlockerIDs(ctx context.Context, blockedID uuid.UUID) ([]uuid.UUID, error)
	GetBlockedIDs(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error)
}
