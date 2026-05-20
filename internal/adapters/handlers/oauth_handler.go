package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/oauth"

	"github.com/gin-gonic/gin"
)

type OAuthHandler struct {
	oauthService ports.OAuthService
	redirectBase string
}

func NewOAuthHandler(oauthService ports.OAuthService, redirectBase string) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		redirectBase: redirectBase,
	}
}

type OAuthRedirectResponse struct {
	AuthorizationURL string `json:"authorization_url" example:"https://github.com/login/oauth/authorize?..."`
	State            string `json:"state" example:"abc123..."`
}

type OAuthCallbackRequest struct {
	Code  string `form:"code" binding:"required"`
	State string `form:"state" binding:"required"`
}

func (h *OAuthHandler) redirectWithError(c *gin.Context, oauthError string) bool {
	state := c.Query("state")
	if state == "" {
		return false
	}

	frontendURI, err := h.oauthService.ExtractRedirectURI(state)
	if err != nil || frontendURI == "" {
		return false
	}

	fragment := url.Values{}
	fragment.Set("error", oauthError)
	redirectURL := fmt.Sprintf("%s#%s", frontendURI, fragment.Encode())
	c.Redirect(http.StatusFound, redirectURL)
	return true
}

type OAuthTokensResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
	IsNewUser    bool   `json:"is_new_user" example:"true"`
}

// @Summary      Get GitHub OAuth URL
// @Description  Get the GitHub authorization URL to redirect the user for OAuth authentication. The redirect_uri must match the origin of the request (Origin or Referer header).
// @Tags         oauth
// @Produce      json
// @Param        redirect_uri query string false "Frontend URL to redirect to after OAuth callback (must match request origin)"
// @Success      200 {object} response.Response{data=OAuthRedirectResponse}
// @Failure      400 {object} response.Response "Invalid redirect_uri"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github [get]
func (h *OAuthHandler) GitHubRedirect(c *gin.Context) {
	frontendRedirectURI := c.Query("redirect_uri")

	if err := validateRedirectURI(frontendRedirectURI, getRequestOrigin(c)); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/github/callback"
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGitHub, redirectURI, frontendRedirectURI, "", "")
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, OAuthRedirectResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// @Summary      GitHub OAuth callback
// @Description  Handle the GitHub OAuth callback. If a redirect_uri was provided during authorization, redirects to that URL with tokens in the fragment. Otherwise returns JSON. When the state contains mode=link, redirects with linked=true&provider=github instead of tokens. Banned users are redirected to the frontend with #error=USER_BANNED. The Accept-Language header is used to set the preferred locale for new accounts.
// @Tags         oauth
// @Produce      json
// @Param        Accept-Language header string false "Preferred language for new accounts (e.g. fr, es). Defaults to en" default(en)
// @Param        code  query string true "Authorization code from GitHub"
// @Param        state query string true "State parameter for CSRF protection"
// @Success      200 {object} response.Response{data=OAuthTokensResponse}
// @Success      302 "Redirects to frontend with tokens, link result, or error in URL fragment"
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response "User account is banned (or redirects with #error=USER_BANNED)"
// @Failure      409 {object} response.Response "OAuth account already linked to another user (link mode)"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github/callback [get]
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	if oauthError := c.Query("error"); oauthError != "" {
		if !h.redirectWithError(c, oauthError) {
			response.BadRequest(c, "OAuth authorization was denied", oauthError)
		}
		return
	}

	var req OAuthCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Missing code or state parameter", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/github/callback"
	locale := domain.LocaleFromAcceptLanguage(c.GetHeader("Accept-Language"))
	result, err := h.oauthService.HandleCallback(c.Request.Context(), ports.OAuthCallbackInput{
		Provider:    oauth.ProviderGitHub,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
		Locale:      locale,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUserBanned) || errors.Is(err, domain.ErrOAuthAccountAlreadyLinked) {
			if frontendURI, extractErr := h.oauthService.ExtractRedirectURI(req.State); extractErr == nil && frontendURI != "" {
				fragment := url.Values{}
				if errors.Is(err, domain.ErrUserBanned) {
					fragment.Set("error", "USER_BANNED")
				} else {
					fragment.Set("error", "OAUTH_ALREADY_LINKED")
				}
				redirectURL := fmt.Sprintf("%s#%s", frontendURI, fragment.Encode())
				c.Redirect(http.StatusFound, redirectURL)
				return
			}
		}
		response.HandleError(c, err)
		return
	}

	if result.LinkedProvider != "" {
		h.redirectWithLinkResult(c, result)
		return
	}

	if result.FrontendRedirectURI != "" {
		h.redirectWithTokens(c, result)
		return
	}

	response.Success(c, OAuthTokensResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
		IsNewUser:    result.IsNewUser,
	})
}

// @Summary      Get Google OAuth URL
// @Description  Get the Google authorization URL to redirect the user for OAuth authentication. The redirect_uri must match the origin of the request (Origin or Referer header).
// @Tags         oauth
// @Produce      json
// @Param        redirect_uri query string false "Frontend URL to redirect to after OAuth callback (must match request origin)"
// @Success      200 {object} response.Response{data=OAuthRedirectResponse}
// @Failure      400 {object} response.Response "Invalid redirect_uri"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google [get]
func (h *OAuthHandler) GoogleRedirect(c *gin.Context) {
	frontendRedirectURI := c.Query("redirect_uri")

	if err := validateRedirectURI(frontendRedirectURI, getRequestOrigin(c)); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/google/callback"
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGoogle, redirectURI, frontendRedirectURI, "", "")
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, OAuthRedirectResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// @Summary      Google OAuth callback
// @Description  Handle the Google OAuth callback. If a redirect_uri was provided during authorization, redirects to that URL with tokens in the fragment. Otherwise returns JSON. When the state contains mode=link, redirects with linked=true&provider=google instead of tokens. Banned users are redirected to the frontend with #error=USER_BANNED. The Accept-Language header is used to set the preferred locale for new accounts.
// @Tags         oauth
// @Produce      json
// @Param        Accept-Language header string false "Preferred language for new accounts (e.g. fr, es). Defaults to en" default(en)
// @Param        code  query string true "Authorization code from Google"
// @Param        state query string true "State parameter for CSRF protection"
// @Success      200 {object} response.Response{data=OAuthTokensResponse}
// @Success      302 "Redirects to frontend with tokens, link result, or error in URL fragment"
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response "User account is banned (or redirects with #error=USER_BANNED)"
// @Failure      409 {object} response.Response "OAuth account already linked to another user (link mode)"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	if oauthError := c.Query("error"); oauthError != "" {
		if !h.redirectWithError(c, oauthError) {
			response.BadRequest(c, "OAuth authorization was denied", oauthError)
		}
		return
	}

	var req OAuthCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Missing code or state parameter", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/google/callback"
	locale := domain.LocaleFromAcceptLanguage(c.GetHeader("Accept-Language"))
	result, err := h.oauthService.HandleCallback(c.Request.Context(), ports.OAuthCallbackInput{
		Provider:    oauth.ProviderGoogle,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
		Locale:      locale,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUserBanned) || errors.Is(err, domain.ErrOAuthAccountAlreadyLinked) {
			if frontendURI, extractErr := h.oauthService.ExtractRedirectURI(req.State); extractErr == nil && frontendURI != "" {
				fragment := url.Values{}
				if errors.Is(err, domain.ErrUserBanned) {
					fragment.Set("error", "USER_BANNED")
				} else {
					fragment.Set("error", "OAUTH_ALREADY_LINKED")
				}
				redirectURL := fmt.Sprintf("%s#%s", frontendURI, fragment.Encode())
				c.Redirect(http.StatusFound, redirectURL)
				return
			}
		}
		response.HandleError(c, err)
		return
	}

	if result.LinkedProvider != "" {
		h.redirectWithLinkResult(c, result)
		return
	}

	if result.FrontendRedirectURI != "" {
		h.redirectWithTokens(c, result)
		return
	}

	response.Success(c, OAuthTokensResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
		IsNewUser:    result.IsNewUser,
	})
}

// @Summary      Link GitHub account
// @Description  Initiate OAuth flow to link a GitHub account to the authenticated user. Returns an authorization URL to redirect the user to GitHub. After authorization, the callback will link the account and redirect to the frontend with linked=true&provider=github in the URL fragment.
// @Tags         oauth
// @Produce      json
// @Security     BearerAuth
// @Param        redirect_uri query string false "Frontend URL to redirect to after OAuth callback (must match request origin)"
// @Success      200 {object} response.Response{data=OAuthRedirectResponse}
// @Failure      400 {object} response.Response "Invalid redirect_uri"
// @Failure      401 {object} response.Response "User not authenticated"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github/link [get]
func (h *OAuthHandler) LinkGitHub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	frontendRedirectURI := c.Query("redirect_uri")
	if err := validateRedirectURI(frontendRedirectURI, getRequestOrigin(c)); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/github/callback"
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGitHub, redirectURI, frontendRedirectURI, "link", userID.String())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, OAuthRedirectResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// @Summary      Unlink GitHub account
// @Description  Remove the linked GitHub account from the current user
// @Tags         oauth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github/unlink [delete]
func (h *OAuthHandler) UnlinkGitHub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := h.oauthService.UnlinkAccount(c.Request.Context(), userID, oauth.ProviderGitHub)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "GitHub account unlinked successfully"})
}

// @Summary      Link Google account
// @Description  Initiate OAuth flow to link a Google account to the authenticated user. Returns an authorization URL to redirect the user to Google. After authorization, the callback will link the account and redirect to the frontend with linked=true&provider=google in the URL fragment.
// @Tags         oauth
// @Produce      json
// @Security     BearerAuth
// @Param        redirect_uri query string false "Frontend URL to redirect to after OAuth callback (must match request origin)"
// @Success      200 {object} response.Response{data=OAuthRedirectResponse}
// @Failure      400 {object} response.Response "Invalid redirect_uri"
// @Failure      401 {object} response.Response "User not authenticated"
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google/link [get]
func (h *OAuthHandler) LinkGoogle(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	frontendRedirectURI := c.Query("redirect_uri")
	if err := validateRedirectURI(frontendRedirectURI, getRequestOrigin(c)); err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/google/callback"
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGoogle, redirectURI, frontendRedirectURI, "link", userID.String())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, OAuthRedirectResponse{
		AuthorizationURL: authURL,
		State:            state,
	})
}

// @Summary      Unlink Google account
// @Description  Remove the linked Google account from the current user
// @Tags         oauth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google/unlink [delete]
func (h *OAuthHandler) UnlinkGoogle(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	err := h.oauthService.UnlinkAccount(c.Request.Context(), userID, oauth.ProviderGoogle)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Google account unlinked successfully"})
}

// @Summary      Get linked OAuth providers
// @Description  Get the linked OAuth providers for the current authenticated user
// @Tags         oauth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=ports.LinkedProvidersResult} "Linked providers status"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /auth/oauth/providers [get]
func (h *OAuthHandler) GetLinkedProviders(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.oauthService.GetLinkedProviders(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, result)
}

func (h *OAuthHandler) redirectWithTokens(c *gin.Context, result *ports.OAuthAuthResult) {
	fragment := url.Values{}
	fragment.Set("access_token", result.Tokens.AccessToken)
	fragment.Set("refresh_token", result.Tokens.RefreshToken)
	fragment.Set("token_type", "Bearer")
	fragment.Set("expires_in", fmt.Sprintf("%d", result.Tokens.ExpiresIn))
	fragment.Set("is_new_user", fmt.Sprintf("%t", result.IsNewUser))

	redirectURL := fmt.Sprintf("%s#%s", result.FrontendRedirectURI, fragment.Encode())
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *OAuthHandler) redirectWithLinkResult(c *gin.Context, result *ports.OAuthAuthResult) {
	fragment := url.Values{}
	fragment.Set("linked", "true")
	fragment.Set("provider", result.LinkedProvider)

	redirectURL := fmt.Sprintf("%s#%s", result.FrontendRedirectURI, fragment.Encode())
	c.Redirect(http.StatusFound, redirectURL)
}

func getRequestOrigin(c *gin.Context) string {
	if origin := c.GetHeader("Origin"); origin != "" {
		return origin
	}

	if referer := c.GetHeader("Referer"); referer != "" {
		if parsed, err := url.Parse(referer); err == nil {
			return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
		}
	}

	return ""
}

func validateRedirectURI(redirectURI, requestOrigin string) error {
	if redirectURI == "" {
		return nil
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect_uri format")
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("redirect_uri must be an absolute URL")
	}

	redirectOrigin := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	if requestOrigin == "" {
		return fmt.Errorf("could not determine request origin (missing Origin or Referer header)")
	}

	if redirectOrigin != requestOrigin {
		return fmt.Errorf("redirect_uri origin does not match request origin")
	}

	return nil
}
