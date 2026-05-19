package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrGitHubTokenExchange = errors.New("failed to exchange GitHub authorization code")
	ErrGitHubUserInfo      = errors.New("failed to get GitHub user info")
)

type GitHubProvider struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewGitHubProvider(clientID, clientSecret string) *GitHubProvider {
	return &GitHubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{},
	}
}

func (p *GitHubProvider) Name() OAuthProvider {
	return ProviderGitHub
}

func (p *GitHubProvider) GetAuthorizationURL(state, redirectURI string) string {
	params := url.Values{
		"client_id":    {p.clientID},
		"redirect_uri": {redirectURI},
		"scope":        {"read:user user:email"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (p *GitHubProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubTokenExchange, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubTokenExchange, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d, body: %s", ErrGitHubTokenExchange, resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubTokenExchange, err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%w: %s - %s", ErrGitHubTokenExchange, tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

func (p *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	userResp, err := p.getUserProfile(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	email, err := p.getPrimaryEmail(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	var avatarURL *string
	if userResp.AvatarURL != "" {
		avatarURL = &userResp.AvatarURL
	}

	return &UserInfo{
		ProviderUserID: fmt.Sprintf("%d", userResp.ID),
		Email:          email,
		Username:       userResp.Login,
		AvatarURL:      avatarURL,
	}, nil
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func (p *GitHubProvider) getUserProfile(ctx context.Context, accessToken string) (*githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrGitHubUserInfo, resp.StatusCode, string(body))
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}

	return &user, nil
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (p *GitHubProvider) getPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d, body: %s", ErrGitHubUserInfo, resp.StatusCode, string(body))
	}

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("%w: %w", ErrGitHubUserInfo, err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("%w: no verified email found", ErrGitHubUserInfo)
}
