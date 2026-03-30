package ports

import (
	"context"

	"github.com/google/uuid"
)

type StatsService interface {
	GetUserStats(ctx context.Context, userID uuid.UUID) (*UserStats, error)
}
