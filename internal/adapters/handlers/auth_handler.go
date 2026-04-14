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

type SendVerificationCodeRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	Code  string `json:"code" binding:"required,len=6" example:"123456"`
}

type PasswordResetRequestBody struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email" example:"user@example.com"`
	Code        string `json:"code" binding:"required,len=6" example:"123456"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72" example:"NewSecure123!"`
}

type TokensResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

// @Summary      Register a new user
// @Description  Create a new user account, return authentication tokens, and send a verification code to the user's email. The user must verify their email before they can log in.
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
// @Description  Authenticate user with email and password. Returns 403 if the email has not been verified yet.
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

// @Summary      Send verification code
// @Description  Send a verification code to the specified email address. Returns success even if the email doesn't exist to prevent enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body SendVerificationCodeRequest true "Email address"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Failure      429 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/verify-email/send [post]
func (h *AuthHandler) SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authService.SendVerificationCode(c.Request.Context(), req.Email); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Verification code sent"})
}

// @Summary      Verify email
// @Description  Verify a user's email with their email address and a 6-digit code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body VerifyEmailRequest true "Email and verification code"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), req.Email, req.Code); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Email verified successfully"})
}

// @Summary      Request password reset
// @Description  Send a password reset code to the specified email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body PasswordResetRequestBody true "Email address"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      429 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/password-reset/request [post]
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req PasswordResetRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authService.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "If the email exists, a reset code has been sent"})
}

// @Summary      Reset password
// @Description  Reset the user's password with a verification code
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ResetPasswordRequest true "Reset details"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Router       /auth/password-reset [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), ports.ResetPasswordInput{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password reset successfully"})
}
