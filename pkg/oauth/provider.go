package oauth

import (
	"context"
)

// OAuthProvider represents an OAuth provider type
type OAuthProvider string

const (
	ProviderGitHub OAuthProvider = "github"
	ProviderGoogle OAuthProvider = "google"
)

// UserInfo contains user information retrieved from an OAuth provider
type UserInfo struct {
	ProviderUserID string
	Email          string
	Username       string
	AvatarURL      *string
}

// Provider defines the interface for OAuth providers
type Provider interface {
	// Name returns the provider identifier
	Name() OAuthProvider

	// GetAuthorizationURL builds the OAuth authorization URL
	GetAuthorizationURL(state, redirectURI string) string

	// ExchangeCode exchanges an authorization code for an access token
	ExchangeCode(ctx context.Context, code, redirectURI string) (accessToken string, err error)

	// GetUserInfo retrieves user information using the access token
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}
