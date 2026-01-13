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
	userService   portservices.UserService
	followService portservices.FollowService
}

func NewUserHandler(userService portservices.UserService, followService portservices.FollowService) *UserHandler {
	return &UserHandler{userService: userService, followService: followService}
}

type UserPreferences struct {
	Theme  string `json:"theme" example:"system"`
	Locale string `json:"locale" example:"en"`
}

type UserStats struct {
	FollowersCount int `json:"followers_count" example:"150"`
	FollowingCount int `json:"following_count" example:"75"`
}

type UserResponse struct {
	ID          string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email       string          `json:"email" example:"user@example.com"`
	Username    string          `json:"username" example:"johndoe"`
	AvatarURL   *string         `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	Bio         *string         `json:"bio" example:"Movie enthusiast"`
	Website     *string         `json:"website" example:"https://example.com"`
	Role        string          `json:"role" example:"user"`
	Preferences UserPreferences `json:"preferences"`
	Stats       UserStats       `json:"stats"`
	CreatedAt   string          `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string          `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	BannedAt    *string         `json:"banned_at,omitempty" example:"2024-01-15T10:30:00Z"`
}

type PublicUserResponse struct {
	ID           string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username     string    `json:"username" example:"johndoe"`
	AvatarURL    *string   `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	Bio          *string   `json:"bio" example:"Movie enthusiast"`
	Website      *string   `json:"website" example:"https://example.com"`
	Stats        UserStats `json:"stats"`
	IsFollowing  bool      `json:"is_following" example:"true"`
	IsFollowedBy bool      `json:"is_followed_by" example:"false"`
	CreatedAt    string    `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type UpdatePreferencesRequest struct {
	Theme  *string `json:"theme" binding:"omitempty,oneof=light dark system" example:"dark"`
	Locale *string `json:"locale" binding:"omitempty,oneof=en fr es" example:"fr"`
}

type UpdateUserRequest struct {
	Email       *string                   `json:"email" binding:"omitempty,email" example:"newemail@example.com"`
	Username    *string                   `json:"username" binding:"omitempty,min=3,max=30" example:"newusername"`
	AvatarURL   *string                   `json:"avatar_url" binding:"omitempty,url" example:"https://example.com/new-avatar.jpg"`
	Bio         *string                   `json:"bio" binding:"omitempty,max=500" example:"Updated bio"`
	Website     *string                   `json:"website" binding:"omitempty,url" example:"https://newwebsite.com"`
	Preferences *UpdatePreferencesRequest `json:"preferences" binding:"omitempty"`
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

	ctx := c.Request.Context()

	user, err := h.userService.GetCurrentUser(ctx, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	followStats, err := h.followService.GetStats(ctx, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	stats := UserStats{
		FollowersCount: followStats.FollowersCount,
		FollowingCount: followStats.FollowingCount,
	}

	response.Success(c, toUserResponse(user, stats))
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
		Email:     req.Email,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		Bio:       req.Bio,
		Website:   req.Website,
	}

	if req.Preferences != nil {
		if req.Preferences.Theme != nil {
			theme := domain.UserTheme(*req.Preferences.Theme)
			input.Theme = &theme
		}
		if req.Preferences.Locale != nil {
			locale := domain.UserLocale(*req.Preferences.Locale)
			input.Locale = &locale
		}
	}

	ctx := c.Request.Context()

	user, err := h.userService.UpdateCurrentUser(ctx, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	followStats, err := h.followService.GetStats(ctx, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	stats := UserStats{
		FollowersCount: followStats.FollowersCount,
		FollowingCount: followStats.FollowingCount,
	}

	response.Success(c, toUserResponse(user, stats))
}

// @Summary      Delete current user
// @Description  Permanently delete the currently authenticated user's account
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      204 "Account deleted successfully"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me [delete]
func (h *UserHandler) DeleteCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.userService.DeleteCurrentUser(c.Request.Context(), userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Get user by ID
// @Description  Get the public profile of a user by their ID. If authenticated, includes follow relationship info.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
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

	ctx := c.Request.Context()

	user, err := h.userService.GetByID(ctx, id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	followStats, err := h.followService.GetStats(ctx, id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	stats := UserStats{
		FollowersCount: followStats.FollowersCount,
		FollowingCount: followStats.FollowingCount,
	}

	var isFollowing, isFollowedBy bool

	if currentUserID, ok := middleware.GetUserID(c); ok {
		isFollowing, _ = h.followService.IsFollowing(ctx, currentUserID, id)
		isFollowedBy, _ = h.followService.IsFollowing(ctx, id, currentUserID)
	}

	response.Success(c, toPublicUserResponse(user, stats, isFollowing, isFollowedBy))
}

func toUserResponse(user *domain.User, stats UserStats) UserResponse {
	resp := UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		Website:   user.Website,
		Role:      string(user.Role),
		Preferences: UserPreferences{
			Theme:  string(user.Theme),
			Locale: string(user.Locale),
		},
		Stats:     stats,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
	if user.BannedAt != nil {
		bannedAt := user.BannedAt.Format(time.RFC3339)
		resp.BannedAt = &bannedAt
	}
	return resp
}

func toPublicUserResponse(user *domain.User, stats UserStats, isFollowing, isFollowedBy bool) PublicUserResponse {
	return PublicUserResponse{
		ID:           user.ID.String(),
		Username:    user.Username,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		Website:     user.Website,
		Stats:       stats,
		IsFollowing:  isFollowing,
		IsFollowedBy: isFollowedBy,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
}
