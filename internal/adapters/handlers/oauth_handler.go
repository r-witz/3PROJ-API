package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
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

type OAuthTokensResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
	IsNewUser    bool   `json:"is_new_user" example:"true"`
}

type OAuthLinkRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
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
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGitHub, redirectURI, frontendRedirectURI)
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
// @Description  Handle the GitHub OAuth callback. If a redirect_uri was provided during authorization, redirects to that URL with tokens in the fragment. Otherwise returns JSON.
// @Tags         oauth
// @Produce      json
// @Param        code  query string true "Authorization code from GitHub"
// @Param        state query string true "State parameter for CSRF protection"
// @Success      200 {object} response.Response{data=OAuthTokensResponse}
// @Success      302 "Redirects to frontend with tokens in URL fragment"
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github/callback [get]
func (h *OAuthHandler) GitHubCallback(c *gin.Context) {
	var req OAuthCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Missing code or state parameter", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/github/callback"
	result, err := h.oauthService.HandleCallback(c.Request.Context(), ports.OAuthCallbackInput{
		Provider:    oauth.ProviderGitHub,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// If frontend redirect URI was provided, redirect with tokens in fragment
	if result.FrontendRedirectURI != "" {
		h.redirectWithTokens(c, result)
		return
	}

	// Otherwise return JSON response
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
	authURL, state, err := h.oauthService.GetAuthorizationURL(oauth.ProviderGoogle, redirectURI, frontendRedirectURI)
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
// @Description  Handle the Google OAuth callback. If a redirect_uri was provided during authorization, redirects to that URL with tokens in the fragment. Otherwise returns JSON.
// @Tags         oauth
// @Produce      json
// @Param        code  query string true "Authorization code from Google"
// @Param        state query string true "State parameter for CSRF protection"
// @Success      200 {object} response.Response{data=OAuthTokensResponse}
// @Success      302 "Redirects to frontend with tokens in URL fragment"
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	var req OAuthCallbackRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Missing code or state parameter", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/google/callback"
	result, err := h.oauthService.HandleCallback(c.Request.Context(), ports.OAuthCallbackInput{
		Provider:    oauth.ProviderGoogle,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// If frontend redirect URI was provided, redirect with tokens in fragment
	if result.FrontendRedirectURI != "" {
		h.redirectWithTokens(c, result)
		return
	}

	// Otherwise return JSON response
	response.Success(c, OAuthTokensResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.Tokens.ExpiresIn,
		IsNewUser:    result.IsNewUser,
	})
}

// @Summary      Link GitHub account
// @Description  Link a GitHub account to the current authenticated user
// @Tags         oauth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body OAuthLinkRequest true "OAuth code and state"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      409 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/github/link [post]
func (h *OAuthHandler) LinkGitHub(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req OAuthLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/github/callback"
	err := h.oauthService.LinkAccount(c.Request.Context(), ports.OAuthLinkInput{
		UserID:      userID,
		Provider:    oauth.ProviderGitHub,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "GitHub account linked successfully"})
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
// @Description  Link a Google account to the current authenticated user
// @Tags         oauth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body OAuthLinkRequest true "OAuth code and state"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      409 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/oauth/google/link [post]
func (h *OAuthHandler) LinkGoogle(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req OAuthLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	redirectURI := h.redirectBase + "/api/v1/auth/oauth/google/callback"
	err := h.oauthService.LinkAccount(c.Request.Context(), ports.OAuthLinkInput{
		UserID:      userID,
		Provider:    oauth.ProviderGoogle,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: redirectURI,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Google account linked successfully"})
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

// redirectWithTokens redirects to the frontend URL with tokens in the URL fragment
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

// getRequestOrigin extracts the origin from the request's Origin or Referer header
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

// validateRedirectURI checks that the redirect URI matches the request origin
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
