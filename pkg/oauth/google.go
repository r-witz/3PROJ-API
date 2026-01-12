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
	ErrGoogleTokenExchange = errors.New("failed to exchange Google authorization code")
	ErrGoogleUserInfo      = errors.New("failed to get Google user info")
)

// GoogleProvider implements OAuth authentication with Google
type GoogleProvider struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewGoogleProvider creates a new Google OAuth provider
func NewGoogleProvider(clientID, clientSecret string) *GoogleProvider {
	return &GoogleProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{},
	}
}

func (p *GoogleProvider) Name() OAuthProvider {
	return ProviderGoogle
}

func (p *GoogleProvider) GetAuthorizationURL(state, redirectURI string) string {
	params := url.Values{
		"client_id":     {p.clientID},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"offline"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleTokenExchange, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleTokenExchange, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d, body: %s", ErrGoogleTokenExchange, resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleTokenExchange, err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%w: %s - %s", ErrGoogleTokenExchange, tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

func (p *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoogleUserInfo, err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoogleUserInfo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrGoogleUserInfo, resp.StatusCode, string(body))
	}

	var userResp struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGoogleUserInfo, err)
	}

	if !userResp.VerifiedEmail {
		return nil, fmt.Errorf("%w: email not verified", ErrGoogleUserInfo)
	}

	// Generate username from name or email
	username := generateUsernameFromName(userResp.Name, userResp.Email)

	var avatarURL *string
	if userResp.Picture != "" {
		avatarURL = &userResp.Picture
	}

	return &UserInfo{
		ProviderUserID: userResp.ID,
		Email:          userResp.Email,
		Username:       username,
		AvatarURL:      avatarURL,
	}, nil
}

// generateUsernameFromName creates a username from a name or email
func generateUsernameFromName(name, email string) string {
	// Try to use the name
	if name != "" {
		// Replace spaces with underscores and convert to lowercase
		username := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		// Remove any non-alphanumeric characters except underscores
		var cleaned strings.Builder
		for _, r := range username {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
				cleaned.WriteRune(r)
			}
		}
		if cleaned.Len() >= 3 {
			return cleaned.String()
		}
	}

	// Fallback to email prefix
	if idx := strings.Index(email, "@"); idx > 0 {
		return strings.ToLower(email[:idx])
	}

	return "user"
}
