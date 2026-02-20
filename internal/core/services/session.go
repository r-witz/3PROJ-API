package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/auth"

	"github.com/google/uuid"
)

type TokenConfig struct {
	AccessTokenSecret  string
	AccessTokenExpiry  time.Duration
	RefreshTokenSecret string
	RefreshTokenExpiry time.Duration
}

func createSession(ctx context.Context, sessionRepo ports.SessionRepository, user *domain.User, config TokenConfig) (*ports.AuthTokens, error) {
	sessionID := uuid.New()

	accessToken, err := auth.GenerateAccessToken(
		user.ID, string(user.Role), config.AccessTokenSecret, config.AccessTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	refreshToken, err := auth.GenerateRefreshToken(
		sessionID, config.RefreshTokenSecret, config.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	now := time.Now()
	session := &domain.Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: auth.HashToken(refreshToken),
		ExpiresAt:        now.Add(config.RefreshTokenExpiry),
		CreatedAt:        now,
	}

	if err := sessionRepo.Create(ctx, session); err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(config.AccessTokenExpiry.Seconds()),
	}, nil
}
