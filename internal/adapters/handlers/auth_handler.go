package handlers

import (
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService ports.AuthService
}

func NewAuthHandler(authService ports.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Username string `json:"username" binding:"required,min=3,max=50" example:"johndoe"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"securepassword123"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"securepassword123"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type TokensResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

// @Summary      Register a new user
// @Description  Create a new user account and return authentication tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration details"
// @Success      201 {object} response.Response{data=TokensResponse}
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	_, tokens, err := h.authService.Register(c.Request.Context(), ports.RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, TokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// @Summary      Login user
// @Description  Authenticate user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} response.Response{data=TokensResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      403 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	_, tokens, err := h.authService.Login(c.Request.Context(), ports.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, TokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// @Summary      Refresh tokens
// @Description  Get new access and refresh tokens using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshRequest true "Refresh token"
// @Success      200 {object} response.Response{data=TokensResponse}
// @Failure      400 {object} response.Response
// @Failure      401 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	tokens, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, TokensResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// @Summary      Logout user
// @Description  Invalidate the refresh token and end the session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LogoutRequest true "Refresh token to invalidate"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Logged out successfully"})
}
