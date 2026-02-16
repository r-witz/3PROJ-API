package services

import (
	"context"
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
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
	userRepo          ports.UserRepository
	oauthRepo         ports.OAuthAccountRepository
	sessionRepo       ports.SessionRepository
	collectionService ports.CollectionService
	stateManager      *oauth.StateManager
	providers         map[oauth.OAuthProvider]oauth.Provider
	config            OAuthServiceConfig
}

func NewOAuthService(
	userRepo ports.UserRepository,
	oauthRepo ports.OAuthAccountRepository,
	sessionRepo ports.SessionRepository,
	collectionService ports.CollectionService,
	stateManager *oauth.StateManager,
	providers map[oauth.OAuthProvider]oauth.Provider,
	config OAuthServiceConfig,
) ports.OAuthService {
	return &oauthService{
		userRepo:          userRepo,
		oauthRepo:         oauthRepo,
		sessionRepo:       sessionRepo,
		collectionService: collectionService,
		stateManager:      stateManager,
		providers:         providers,
		config:            config,
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

func (s *oauthService) HandleCallback(ctx context.Context, input ports.OAuthCallbackInput) (*ports.OAuthAuthResult, error) {
	stateData, err := s.stateManager.Validate(input.State)
	if err != nil {
		return nil, domain.ErrOAuthStateMismatch
	}

	provider, ok := s.providers[input.Provider]
	if !ok {
		return nil, domain.ErrOAuthProviderNotSupported
	}

	accessToken, err := provider.ExchangeCode(ctx, input.Code, input.RedirectURI)
	if err != nil {
		return nil, domain.ErrInternal
	}

	oauthUserInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, domain.ErrInternal
	}

	oauthAccount, err := s.oauthRepo.GetByProviderAndProviderUserID(ctx, string(input.Provider), oauthUserInfo.ProviderUserID)
	if err != nil {
		return nil, domain.ErrInternal
	}

	var user *domain.User
	isNewUser := false

	if oauthAccount != nil {
		user, err = s.userRepo.GetByID(ctx, oauthAccount.UserID)
		if err != nil || user == nil {
			return nil, domain.ErrInternal
		}

		if user.BannedAt != nil {
			return nil, domain.ErrUserBanned
		}
	} else {
		existingUser, err := s.userRepo.GetByEmail(ctx, oauthUserInfo.Email)
		if err != nil {
			return nil, domain.ErrInternal
		}

		if existingUser != nil {
			user = existingUser
			if err := s.linkOAuthAccount(ctx, user.ID, input.Provider, oauthUserInfo); err != nil {
				return nil, err
			}
		} else {
			user, err = s.createOAuthUser(ctx, input.Provider, oauthUserInfo)
			if err != nil {
				return nil, err
			}
			isNewUser = true
		}
	}

	tokens, err := s.createSession(ctx, user)
	if err != nil {
		return nil, err
	}

	return &ports.OAuthAuthResult{
		User:                user,
		Tokens:              tokens,
		IsNewUser:           isNewUser,
		FrontendRedirectURI: stateData.RedirectURI,
	}, nil
}

func (s *oauthService) LinkAccount(ctx context.Context, input ports.OAuthLinkInput) error {
	if _, err := s.stateManager.Validate(input.State); err != nil {
		return domain.ErrOAuthStateMismatch
	}

	provider, ok := s.providers[input.Provider]
	if !ok {
		return domain.ErrOAuthProviderNotSupported
	}

	accessToken, err := provider.ExchangeCode(ctx, input.Code, input.RedirectURI)
	if err != nil {
		return domain.ErrInternal
	}

	oauthUserInfo, err := provider.GetUserInfo(ctx, accessToken)
	if err != nil {
		return domain.ErrInternal
	}

	existingOAuth, err := s.oauthRepo.GetByProviderAndProviderUserID(ctx, string(input.Provider), oauthUserInfo.ProviderUserID)
	if err != nil {
		return domain.ErrInternal
	}
	if existingOAuth != nil {
		if existingOAuth.UserID != input.UserID {
			return domain.ErrOAuthAccountAlreadyLinked
		}
		return nil
	}

	existingLink, err := s.oauthRepo.GetByUserIDAndProvider(ctx, input.UserID, string(input.Provider))
	if err != nil {
		return domain.ErrInternal
	}
	if existingLink != nil {
		if err := s.oauthRepo.Delete(ctx, existingLink.Provider, existingLink.ProviderUserID); err != nil {
			return domain.ErrInternal
		}
	}

	return s.linkOAuthAccount(ctx, input.UserID, input.Provider, oauthUserInfo)
}

func (s *oauthService) UnlinkAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	hasPassword := user.PasswordHash != nil

	githubOAuth, _ := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGitHub))
	googleOAuth, _ := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGoogle))

	oauthCount := 0
	if githubOAuth != nil {
		oauthCount++
	}
	if googleOAuth != nil {
		oauthCount++
	}

	if !hasPassword && oauthCount <= 1 {
		return domain.ErrCannotUnlinkOnlyAuth
	}

	oauthAccount, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(provider))
	if err != nil {
		return domain.ErrInternal
	}
	if oauthAccount == nil {
		return domain.ErrOAuthAccountNotFound
	}

	if err := s.oauthRepo.Delete(ctx, oauthAccount.Provider, oauthAccount.ProviderUserID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *oauthService) createOAuthUser(ctx context.Context, provider oauth.OAuthProvider, info *oauth.UserInfo) (*domain.User, error) {
	username, err := s.generateUniqueUsername(ctx, info.Username)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        info.Email,
		Username:     username,
		PasswordHash: nil,
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

	if s.collectionService != nil {
		if err := s.collectionService.CreateDefaultCollections(ctx, user.ID); err != nil {
			return nil, domain.ErrInternal
		}
	}

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
	existing, err := s.userRepo.GetByUsername(ctx, baseUsername)
	if err != nil {
		return "", domain.ErrInternal
	}
	if existing == nil {
		return baseUsername, nil
	}

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

	return fmt.Sprintf("%s_%s", baseUsername, uuid.New().String()[:8]), nil
}

func (s *oauthService) createSession(ctx context.Context, user *domain.User) (*ports.AuthTokens, error) {
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
