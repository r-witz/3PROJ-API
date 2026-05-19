package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	verificationPrefix         = "verification"
	verificationRatePrefix     = "verification:rate"
	verificationCooldownPrefix = "verification:cooldown"
	maxRequestsPerWindow       = 3
	cooldownDuration           = 60 * time.Second
)

type VerificationCodeRepo struct {
	client *redis.Client
}

func NewVerificationCodeRepo(client *redis.Client) *VerificationCodeRepo {
	return &VerificationCodeRepo{client: client}
}

func codeKey(purpose domain.VerificationCodePurpose, email string) string {
	return fmt.Sprintf("%s:%s:%s", verificationPrefix, purpose, email)
}

func rateKey(purpose domain.VerificationCodePurpose, email string) string {
	return fmt.Sprintf("%s:%s:%s", verificationRatePrefix, purpose, email)
}

func (r *VerificationCodeRepo) Store(ctx context.Context, code *domain.VerificationCode, ttl time.Duration) error {
	data, err := json.Marshal(code)
	if err != nil {
		return fmt.Errorf("failed to marshal verification code: %w", err)
	}
	return r.client.Set(ctx, codeKey(code.Purpose, code.Email), data, ttl).Err()
}

func (r *VerificationCodeRepo) Get(ctx context.Context, email string, purpose domain.VerificationCodePurpose) (*domain.VerificationCode, error) {
	data, err := r.client.Get(ctx, codeKey(purpose, email)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get verification code: %w", err)
	}

	var code domain.VerificationCode
	if err := json.Unmarshal(data, &code); err != nil {
		return nil, fmt.Errorf("failed to unmarshal verification code: %w", err)
	}
	return &code, nil
}

func (r *VerificationCodeRepo) Delete(ctx context.Context, email string, purpose domain.VerificationCodePurpose) error {
	return r.client.Del(ctx, codeKey(purpose, email)).Err()
}

func cooldownKey(purpose domain.VerificationCodePurpose, email string) string {
	return fmt.Sprintf("%s:%s:%s", verificationCooldownPrefix, purpose, email)
}

func (r *VerificationCodeRepo) CanRequest(ctx context.Context, email string, purpose domain.VerificationCodePurpose) (bool, error) {
	exists, err := r.client.Exists(ctx, cooldownKey(purpose, email)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cooldown: %w", err)
	}
	if exists > 0 {
		return false, nil
	}

	count, err := r.client.Get(ctx, rateKey(purpose, email)).Int()
	if errors.Is(err, redis.Nil) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}
	return count < maxRequestsPerWindow, nil
}

func (r *VerificationCodeRepo) DeleteAllForEmail(ctx context.Context, email string) error {
	purposes := []domain.VerificationCodePurpose{
		domain.VerificationCodePurposeEmailVerify,
		domain.VerificationCodePurposePasswordReset,
		domain.VerificationCodePurposeEmailChange,
	}
	pipe := r.client.Pipeline()
	for _, p := range purposes {
		pipe.Del(ctx, codeKey(p, email))
		pipe.Del(ctx, rateKey(p, email))
		pipe.Del(ctx, cooldownKey(p, email))
	}
	_, err := pipe.Exec(ctx)
	return err
}

func pendingEmailKey(userID uuid.UUID) string {
	return fmt.Sprintf("verification:pending_email:%s", userID.String())
}

func (r *VerificationCodeRepo) StorePendingEmail(ctx context.Context, userID uuid.UUID, newEmail string, ttl time.Duration) error {
	return r.client.Set(ctx, pendingEmailKey(userID), newEmail, ttl).Err()
}

func (r *VerificationCodeRepo) GetPendingEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	email, err := r.client.Get(ctx, pendingEmailKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get pending email: %w", err)
	}
	return email, nil
}

func (r *VerificationCodeRepo) DeletePendingEmail(ctx context.Context, userID uuid.UUID) error {
	return r.client.Del(ctx, pendingEmailKey(userID)).Err()
}

func (r *VerificationCodeRepo) RecordRequest(ctx context.Context, email string, purpose domain.VerificationCodePurpose, window time.Duration) error {
	pipe := r.client.Pipeline()
	key := rateKey(purpose, email)
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	pipe.Set(ctx, cooldownKey(purpose, email), 1, cooldownDuration)
	_, err := pipe.Exec(ctx)
	return err
}
