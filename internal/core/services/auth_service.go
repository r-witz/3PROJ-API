package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/auth"

	"github.com/google/uuid"
)

type AuthServiceConfig struct {
	AccessTokenSecret  string
	AccessTokenExpiry  time.Duration
	RefreshTokenSecret string
	RefreshTokenExpiry time.Duration
}

type authService struct {
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	config      AuthServiceConfig
}

func NewAuthService(
	userRepo ports.UserRepository,
	sessionRepo ports.SessionRepository,
	config AuthServiceConfig,
) ports.AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		config:      config,
	}
}

func (s *authService) Register(ctx context.Context, input ports.RegisterInput) (*domain.User, *ports.AuthTokens, error) {
	existing, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, nil, domain.ErrEmailAlreadyExists
	}

	existing, err = s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, nil, domain.ErrUsernameAlreadyExists
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		Username:     input.Username,
		PasswordHash: &passwordHash,
		Role:         domain.UserRoleUser,
		Theme:        domain.UserThemeSystem,
		Locale:       domain.UserLocaleEN,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, nil, domain.ErrInternal
	}

	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *authService) Login(ctx context.Context, input ports.LoginInput) (*domain.User, *ports.AuthTokens, error) {
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, domain.ErrInternal
	}
	if user == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if user.BannedAt != nil {
		return nil, nil, domain.ErrUserBanned
	}

	if user.PasswordHash == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	match, err := auth.ComparePassword(*user.PasswordHash, input.Password)
	if err != nil || !match {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (*ports.AuthTokens, error) {
	claims, err := auth.ValidateRefreshToken(refreshToken, s.config.RefreshTokenSecret)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	tokenHash := auth.HashToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if session == nil || session.ID != claims.SessionID {
		return nil, domain.ErrInvalidToken
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessionRepo.Delete(ctx, session.ID)
		return nil, domain.ErrSessionExpired
	}

	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, domain.ErrInternal
	}

	accessToken, err := auth.GenerateAccessToken(
		user.ID, string(user.Role), s.config.AccessTokenSecret, s.config.AccessTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	newRefreshToken, err := auth.GenerateRefreshToken(
		session.ID, s.config.RefreshTokenSecret, s.config.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	session.RefreshTokenHash = auth.HashToken(newRefreshToken)
	session.ExpiresAt = time.Now().Add(s.config.RefreshTokenExpiry)
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := auth.HashToken(refreshToken)
	session, err := s.sessionRepo.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		return domain.ErrInternal
	}
	if session == nil {
		return nil
	}

	return s.sessionRepo.Delete(ctx, session.ID)
}

func (s *authService) createSession(ctx context.Context, user *domain.User) (*ports.AuthTokens, error) {
	sessionID := uuid.New()

	accessToken, err := auth.GenerateAccessToken(
		user.ID, string(user.Role), s.config.AccessTokenSecret, s.config.AccessTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	refreshToken, err := auth.GenerateRefreshToken(
		sessionID, s.config.RefreshTokenSecret, s.config.RefreshTokenExpiry,
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	session := &domain.Session{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: auth.HashToken(refreshToken),
		ExpiresAt:        time.Now().Add(s.config.RefreshTokenExpiry),
		CreatedAt:        time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}
