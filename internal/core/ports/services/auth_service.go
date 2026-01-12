package services

import (
	"context"

	"duskforge-api/internal/core/domain"
)

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*domain.User, *AuthTokens, error)
	Login(ctx context.Context, input LoginInput) (*domain.User, *AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
}
