package oauth

import (
	"context"
)

type OAuthProvider string

const (
	ProviderGitHub OAuthProvider = "github"
	ProviderGoogle OAuthProvider = "google"
)

type UserInfo struct {
	ProviderUserID string
	Email          string
	Username       string
	AvatarURL      *string
}

type Provider interface {
	Name() OAuthProvider
	GetAuthorizationURL(state, redirectURI string) string
	ExchangeCode(ctx context.Context, code, redirectURI string) (accessToken string, err error)
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}
