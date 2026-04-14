package ports

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type VerificationCodeRepository interface {
	Store(ctx context.Context, code *domain.VerificationCode, ttl time.Duration) error
	Get(ctx context.Context, email string, purpose domain.VerificationCodePurpose) (*domain.VerificationCode, error)
	Delete(ctx context.Context, email string, purpose domain.VerificationCodePurpose) error
	CanRequest(ctx context.Context, email string, purpose domain.VerificationCodePurpose) (bool, error)
	RecordRequest(ctx context.Context, email string, purpose domain.VerificationCodePurpose, window time.Duration) error
	DeleteAllForEmail(ctx context.Context, email string) error
	StorePendingEmail(ctx context.Context, userID uuid.UUID, newEmail string, ttl time.Duration) error
	GetPendingEmail(ctx context.Context, userID uuid.UUID) (string, error)
	DeletePendingEmail(ctx context.Context, userID uuid.UUID) error
}
