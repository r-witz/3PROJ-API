package services

import (
	"context"
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	portservices "duskforge-api/internal/core/ports/services"
	"duskforge-api/pkg/auth"
	"duskforge-api/pkg/oauth"

	"github.com/google/uuid"
)

type OAuthServiceConfig struct {
	AccessTokenSecret  string
	AccessTokenExpiry  time.Duration
	RefreshTokenSecret string
	RefreshTokenExpiry time.Duration
}

type oauthService struct {
	userRepo     ports.UserRepository
	oauthRepo    ports.OAuthAccountRepository
	sessionRepo  ports.SessionRepository
	stateManager *oauth.StateManager
	providers    map[oauth.OAuthProvider]oauth.Provider
	config       OAuthServiceConfig
}

func NewOAuthService(
	userRepo ports.UserRepository,
	oauthRepo ports.OAuthAccountRepository,
	sessionRepo ports.SessionRepository,
	stateManager *oauth.StateManager,
	providers map[oauth.OAuthProvider]oauth.Provider,
	config OAuthServiceConfig,
) portservices.OAuthService {
	return &oauthService{
		userRepo:     userRepo,
		oauthRepo:    oauthRepo,
		sessionRepo:  sessionRepo,
		stateManager: stateManager,
		providers:    providers,
		config:       config,
	}
}

func (s *oauthService) GetAuthorizationURL(provider oauth.OAuthProvider, redirectURI string, frontendRedirectURI string) (string, string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", "", domain.ErrOAuthProviderNotSupported
	}

	state, err := s.stateManager.Generate(frontendRedirectURI)
	if err != nil {
		return "", "", domain.ErrInternal
	}

	authURL := p.GetAuthorizationURL(state, redirectURI)
	return authURL, state, nil
}

func (s *oauthService) HandleCallback(ctx context.Context, input portservices.OAuthCallbackInput) (*portservices.OAuthAuthResult, error) {
	// Validate state and extract data
	stateData, err := s.stateManager.Validate(input.State)
	if err != nil {
		return nil, domain.ErrOAuthStateMismatch
	}

	// Get provider
	provider, ok := s.providers[input.Provider]
	if !ok {
		return nil, domain.ErrOAuthProviderNotSupported
	}

	// Exchange code for token
	accessToken, err := provider.ExchangeCode(ctx, input.Code, input.RedirectURI)
	if err != nil {
		return nil, domain.ErrInternal
	}

	// Get user info from provider
	oauthUserInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, domain.ErrInternal
	}

	// Check if OAuth account exists
	oauthAccount, err := s.oauthRepo.GetByProviderAndProviderUserID(ctx, string(input.Provider), oauthUserInfo.ProviderUserID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	var user *domain.User
	isNewUser := false

	if oauthAccount != nil {
		// Existing OAuth user - get user and login
		user, err = s.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil || user == nil {
			return nil, domain.ErrInternal
		}

		if user.BannedAt != nil {
			return nil, domain.ErrUserBanned
		}
	} else {
		// Check if email already exists
		existingUser, err := s.userRepo.GetByEmail(ctx, oauthUserInfo.Email)
		if err != nil {
			return nil, domain.ErrInternal
		}

		if existingUser != nil {
			// Email exists - link OAuth to existing account
			user = existingUser
			if err := s.linkOAuthAccount(ctx, user.ID, input.Provider, oauthUserInfo); err != nil {
				return nil, err
			}
		} else {
			// New user - register via OAuth
			user, err = s.createOAuthUser(ctx, input.Provider, oauthUserInfo)
			if err != nil {
				return nil, err
			}
			isNewUser = true
		}
	}

	// Create session
	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, err
	}

	return &portservices.OAuthAuthResult{
		User:                user,
		Tokens:              tokens,
		IsNewUser:           isNewUser,
		FrontendRedirectURI: stateData.RedirectURI,
	}, nil
}

func (s *oauthService) LinkAccount(ctx context.Context, input portservices.OAuthLinkInput) error {
	// Validate state
	if _, err := s.stateManager.Validate(input.State); err != nil {
		return domain.ErrOAuthStateMismatch
	}

	// Get provider
	provider, ok := s.providers[input.Provider]
	if !ok {
		return domain.ErrOAuthProviderNotSupported
	}

	// Exchange code for token
	accessToken, err := provider.ExchangeCode(ctx, input.Code, input.RedirectURI)
	if err != nil {
		return domain.ErrInternal
	}

	// Get user info from provider
	oauthUserInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return domain.ErrInternal
	}

	// Check if OAuth account is already linked to another user
	existingOAuth, err := s.oauthRepo.GetByProviderAndProviderUserID(ctx, string(input.Provider), oauthUserInfo.ProviderUserID)
	if err != nil {
		return domain.ErrInternal
	}
	if existingOAuth != nil {
		if existingOAuth.UserID != input.UserID {
			return domain.ErrOAuthAccountAlreadyLinked
		}
		// Already linked to this user
		return nil
	}

	// Check if user already has this provider linked
	existingLink, err := s.oauthRepo.GetByUserIDAndProvider(ctx, input.UserID, string(input.Provider))
	if err != nil {
		return domain.ErrInternal
	}
	if existingLink != nil {
		// Already has this provider linked, update it
		if err := s.oauthRepo.Delete(ctx, existingLink.Provider, existingLink.ProviderUserID); err != nil {
			return domain.ErrInternal
		}
	}

	// Link OAuth account
	return s.linkOAuthAccount(ctx, input.UserID, input.Provider, oauthUserInfo)
}

func (s *oauthService) UnlinkAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider) error {
	// Get user to check authentication methods
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	// Check if this is the only auth method
	hasPassword := user.PasswordHash != nil

	// Count OAuth accounts
	githubOAuth, _ := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGitHub))
	googleOAuth, _ := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGoogle))

	oauthCount := 0
	if githubOAuth != nil {
		oauthCount++
	}
	if googleOAuth != nil {
		oauthCount++
	}

	// Cannot unlink if this is the only auth method
	if !hasPassword && oauthCount <= 1 {
		return domain.ErrCannotUnlinkOnlyAuth
	}

	// Get the OAuth account to delete
	oauthAccount, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(provider))
	if err != nil {
		return domain.ErrInternal
	}
	if oauthAccount == nil {
		return domain.ErrOAuthAccountNotFound
	}

	// Delete OAuth account
	if err := s.oauthRepo.Delete(ctx, oauthAccount.Provider, oauthAccount.ProviderUserID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *oauthService) createOAuthUser(ctx context.Context, provider oauth.OAuthProvider, info *oauth.UserInfo) (*domain.User, error) {
	// Generate unique username
	username, err := s.generateUniqueUsername(ctx, info.Username)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        info.Email,
		Username:     username,
		PasswordHash: nil, // OAuth-only users have no password
		AvatarURL:    info.AvatarURL,
		Role:         domain.UserRoleUser,
		Theme:        domain.UserThemeSystem,
		Locale:       domain.UserLocaleEN,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, domain.ErrInternal
	}

	// Link OAuth account
	if err := s.linkOAuthAccount(ctx, user.ID, provider, info); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *oauthService) linkOAuthAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider, info *oauth.UserInfo) error {
	oauthAccount := &domain.OAuthAccount{
		Provider:       string(provider),
		ProviderUserID: info.ProviderUserID,
		UserID:         userID,
		CreatedAt:      time.Now(),
	}

	if err := s.oauthRepo.Create(ctx, oauthAccount); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *oauthService) generateUniqueUsername(ctx context.Context, baseUsername string) (string, error) {
	// Try base username first
	existing, err := s.userRepo.GetByUsername(ctx, baseUsername)
	if err != nil {
		return "", domain.ErrInternal
	}
	if existing == nil {
		return baseUsername, nil
	}

	// Append numbers until we find a unique username
	for i := 1; i < 1000; i++ {
		username := fmt.Sprintf("%s%d", baseUsername, i)
		existing, err := s.userRepo.GetByUsername(ctx, username)
		if err != nil {
			return "", domain.ErrInternal
		}
		if existing == nil {
			return username, nil
		}
	}

	// Fallback to UUID suffix if all else fails
	return fmt.Sprintf("%s_%s", baseUsername, uuid.New().String()[:8]), nil
}

func (s *oauthService) createSession(ctx context.Context, user *domain.User) (*portservices.AuthTokens, error) {
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

	return &portservices.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}
