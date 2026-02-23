package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type BlockRepository interface {
	Create(ctx context.Context, block *domain.UserBlock) error
	Delete(ctx context.Context, blockerID, blockedID uuid.UUID) error
	GetByBlockerAndBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (*domain.UserBlock, error)
	GetBlockedByUser(ctx context.Context, blockerID uuid.UUID, offset, limit int) ([]*domain.UserBlock, int, error)
	IsBlocked(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error)
}
