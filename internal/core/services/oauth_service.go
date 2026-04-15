package services

import (
	"context"
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/oauth"

	"github.com/google/uuid"
)

type oauthService struct {
	userRepo          ports.UserRepository
	oauthRepo         ports.OAuthAccountRepository
	sessionRepo       ports.SessionRepository
	collectionService ports.CollectionService
	stateManager      *oauth.StateManager
	providers         map[oauth.OAuthProvider]oauth.Provider
	config            TokenConfig
}

func NewOAuthService(
	userRepo ports.UserRepository,
	oauthRepo ports.OAuthAccountRepository,
	sessionRepo ports.SessionRepository,
	collectionService ports.CollectionService,
	stateManager *oauth.StateManager,
	providers map[oauth.OAuthProvider]oauth.Provider,
	config TokenConfig,
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

func (s *oauthService) GetAuthorizationURL(provider oauth.OAuthProvider, redirectURI string, frontendRedirectURI string, mode string, userID string) (string, string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", "", domain.ErrOAuthProviderNotSupported
	}

	state, err := s.stateManager.Generate(frontendRedirectURI, mode, userID)
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

	if stateData.Mode == "link" {
		userID, err := uuid.Parse(stateData.UserID)
		if err != nil {
			return nil, domain.ErrInternal
		}

		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil || user == nil {
			return nil, domain.ErrInternal
		}

		existingOAuth, err := s.oauthRepo.GetByProviderAndProviderUserID(ctx, string(input.Provider), oauthUserInfo.ProviderUserID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if existingOAuth != nil {
			if existingOAuth.UserID != userID {
				return nil, domain.ErrOAuthAccountAlreadyLinked
			}
			return &ports.OAuthAuthResult{
				FrontendRedirectURI: stateData.RedirectURI,
				LinkedProvider:      string(input.Provider),
			}, nil
		}

		existingLink, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(input.Provider))
		if err != nil {
			return nil, domain.ErrInternal
		}
		if existingLink != nil {
			if err := s.oauthRepo.Delete(ctx, existingLink.Provider, existingLink.ProviderUserID); err != nil {
				return nil, domain.ErrInternal
			}
		}

		if err := s.linkOAuthAccount(ctx, userID, input.Provider, oauthUserInfo); err != nil {
			return nil, err
		}

		return &ports.OAuthAuthResult{
			FrontendRedirectURI: stateData.RedirectURI,
			LinkedProvider:      string(input.Provider),
		}, nil
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
			if !user.EmailVerified {
				if err := s.userRepo.SetEmailVerified(ctx, user.ID, true); err != nil {
					return nil, domain.ErrInternal
				}
				user.EmailVerified = true
			}
			if err := s.linkOAuthAccount(ctx, user.ID, input.Provider, oauthUserInfo); err != nil {
				return nil, err
			}
		} else {
			user, err = s.createOAuthUser(ctx, input.Provider, oauthUserInfo, input.Locale)
			if err != nil {
				return nil, err
			}
			isNewUser = true
		}
	}

	tokens, err := createSession(ctx, s.sessionRepo, user, s.config)
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

func (s *oauthService) ExtractRedirectURI(state string) (string, error) {
	stateData, err := s.stateManager.Validate(state)
	if err != nil {
		return "", err
	}
	return stateData.RedirectURI, nil
}

func (s *oauthService) UnlinkAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return domain.ErrInternal
	}

	oauthAccount, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(provider))
	if err != nil {
		return domain.ErrInternal
	}
	if oauthAccount == nil {
		return domain.ErrOAuthAccountNotFound
	}

	hasPassword := user.PasswordHash != nil

	otherProviders := []oauth.OAuthProvider{oauth.ProviderGitHub, oauth.ProviderGoogle}
	otherLinked := 0
	for _, p := range otherProviders {
		if p == provider {
			continue
		}
		link, _ := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(p))
		if link != nil {
			otherLinked++
		}
	}

	if !hasPassword && otherLinked == 0 {
		return domain.ErrCannotUnlinkOnlyAuth
	}

	if err := s.oauthRepo.Delete(ctx, oauthAccount.Provider, oauthAccount.ProviderUserID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *oauthService) GetLinkedProviders(ctx context.Context, userID uuid.UUID) (*ports.LinkedProvidersResult, error) {
	githubAccount, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGitHub))
	if err != nil {
		return nil, domain.ErrInternal
	}

	googleAccount, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, string(oauth.ProviderGoogle))
	if err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.LinkedProvidersResult{
		GitHub: githubAccount != nil,
		Google: googleAccount != nil,
	}, nil
}

func (s *oauthService) createOAuthUser(ctx context.Context, provider oauth.OAuthProvider, info *oauth.UserInfo, locale domain.UserLocale) (*domain.User, error) {
	username, err := s.generateUniqueUsername(ctx, info.Username)
	if err != nil {
		return nil, err
	}

	if locale == "" {
		locale = domain.UserLocaleEN
	}

	now := time.Now()
	user := &domain.User{
		ID:            uuid.New(),
		Email:         info.Email,
		EmailVerified: true,
		Username:      username,
		PasswordHash:  nil,
		AvatarURL:     info.AvatarURL,
		Role:          domain.UserRoleUser,
		Theme:         domain.UserThemeSystem,
		Locale:        locale,
		CreatedAt:     now,
		UpdatedAt:     now,
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
