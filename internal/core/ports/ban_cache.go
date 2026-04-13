package ports

import (
	"context"

	"github.com/google/uuid"
)

type BanCache interface {
	IsBanned(ctx context.Context, userID uuid.UUID) (bool, error)
	SetBanned(ctx context.Context, userID uuid.UUID) error
	RemoveBanned(ctx context.Context, userID uuid.UUID) error
}
