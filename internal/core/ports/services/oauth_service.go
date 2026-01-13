package services

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/oauth"

	"github.com/google/uuid"
)

// OAuthCallbackInput contains the data received from an OAuth callback
type OAuthCallbackInput struct {
	Provider    oauth.OAuthProvider
	Code        string
	State       string
	RedirectURI string
}

// OAuthLinkInput contains the data needed to link an OAuth account
type OAuthLinkInput struct {
	UserID      uuid.UUID
	Provider    oauth.OAuthProvider
	Code        string
	State       string
	RedirectURI string
}

// OAuthAuthResult contains the result of an OAuth authentication
type OAuthAuthResult struct {
	User                *domain.User
	Tokens              *AuthTokens
	IsNewUser           bool
	FrontendRedirectURI string
}

// OAuthService defines the interface for OAuth authentication operations
type OAuthService interface {
	// GetAuthorizationURL generates the OAuth provider redirect URL with state
	// frontendRedirectURI is the URL to redirect the user to after OAuth callback
	GetAuthorizationURL(provider oauth.OAuthProvider, redirectURI string, frontendRedirectURI string) (authURL string, state string, err error)

	// HandleCallback processes the OAuth callback and returns auth tokens
	// This handles both new user registration and existing OAuth login
	HandleCallback(ctx context.Context, input OAuthCallbackInput) (*OAuthAuthResult, error)

	// LinkAccount links an OAuth provider to an existing authenticated user
	LinkAccount(ctx context.Context, input OAuthLinkInput) error

	// UnlinkAccount removes an OAuth provider from a user's account
	UnlinkAccount(ctx context.Context, userID uuid.UUID, provider oauth.OAuthProvider) error
}
