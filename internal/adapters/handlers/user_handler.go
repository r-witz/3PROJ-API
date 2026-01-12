package handlers

import (
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	portservices "duskforge-api/internal/core/ports/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService portservices.UserService
}

func NewUserHandler(userService portservices.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type UserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"email" example:"user@example.com"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Bio       *string `json:"bio,omitempty" example:"Movie enthusiast"`
	Website   *string `json:"website,omitempty" example:"https://example.com"`
	Role      string  `json:"role" example:"user"`
	Theme     string  `json:"theme" example:"system"`
	Locale    string  `json:"locale" example:"en"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type PublicUserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Bio       *string `json:"bio,omitempty" example:"Movie enthusiast"`
	Website   *string `json:"website,omitempty" example:"https://example.com"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type UpdateUserRequest struct {
	Username  *string `json:"username" binding:"omitempty,min=3,max=30" example:"newusername"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,url" example:"https://example.com/new-avatar.jpg"`
	Bio       *string `json:"bio" binding:"omitempty,max=500" example:"Updated bio"`
	Website   *string `json:"website" binding:"omitempty,url" example:"https://newwebsite.com"`
	Theme     *string `json:"theme" binding:"omitempty,oneof=light dark system" example:"dark"`
	Locale    *string `json:"locale" binding:"omitempty,oneof=en fr es" example:"fr"`
}

// @Summary      Get current user
// @Description  Get the profile of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=UserResponse} "User profile"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	user, err := h.userService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toUserResponse(user))
}

// @Summary      Update current user
// @Description  Update the profile of the currently authenticated user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body UpdateUserRequest true "Fields to update"
// @Success      200 {object} response.Response{data=UserResponse} "Updated user profile"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      409 {object} response.Response "Username already taken"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me [patch]
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := portservices.UpdateUserInput{
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		Bio:       req.Bio,
		Website:   req.Website,
	}

	if req.Theme != nil {
		theme := domain.UserTheme(*req.Theme)
		input.Theme = &theme
	}
	if req.Locale != nil {
		locale := domain.UserLocale(*req.Locale)
		input.Locale = &locale
	}

	user, err := h.userService.UpdateCurrentUser(c.Request.Context(), userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toUserResponse(user))
}

// @Summary      Get user by ID
// @Description  Get the public profile of a user by their ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID" format(uuid)
// @Success      200 {object} response.Response{data=PublicUserResponse} "User public profile"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toPublicUserResponse(user))
}

func toUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		Website:   user.Website,
		Role:      string(user.Role),
		Theme:     string(user.Theme),
		Locale:    string(user.Locale),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

func toPublicUserResponse(user *domain.User) PublicUserResponse {
	return PublicUserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		Website:   user.Website,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}
