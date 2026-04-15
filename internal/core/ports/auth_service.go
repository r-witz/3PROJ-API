package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type RegisterInput struct {
	Email    string
	Username string
	Password string
	Locale   domain.UserLocale
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

type ResetPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*domain.User, *AuthTokens, error)
	Login(ctx context.Context, input LoginInput) (*domain.User, *AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	SendVerificationCode(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, email string, code string) error
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, input ResetPasswordInput) error
	RequestEmailChange(ctx context.Context, userID uuid.UUID, newEmail string) error
	ConfirmEmailChange(ctx context.Context, userID uuid.UUID, code string) error
}
