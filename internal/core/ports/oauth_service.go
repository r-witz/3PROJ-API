package ports

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/oauth"

	"github.com/google/uuid"
)

type OAuthCallbackInput struct {
	Provider    oauth.OAuthProvider
	Code        string
	State       string
	RedirectURI string
}

type OAuthAuthResult struct {
	User                *domain.User
	Tokens              *AuthTokens
	IsNewUser           bool
	FrontendRedirectURI string
	LinkedProvider      string
}

type LinkedProvidersResult struct {
	GitHub bool `json:"github"`
	Google bool `json:"google"`
}

type OAuthService interface {
	GetAuthorizationURL(provider oauth.OAuthProvider, redirectURI string, frontendRedirectURI string, mode string, userID string) (authURL string, state string, err error)
	HandleCallback(ctx context.Context, input OAuthCallbackInput) (*OAuthAuthResult, error)
	UnlinkAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider) error
	GetLinkedProviders(ctx context.Context, userID uuid.UUID) (*LinkedProvidersResult, error)
	ExtractRedirectURI(state string) (string, error)
}
