package handlers

import (
	"fmt"
	"net/url"
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/query"
	"duskforge-api/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService   ports.UserService
	followService ports.FollowService
	blockService  ports.BlockService
	storage       *storage.MinioStorage
}

func NewUserHandler(userService ports.UserService, followService ports.FollowService, blockService ports.BlockService, storage *storage.MinioStorage) *UserHandler {
	return &UserHandler{userService: userService, followService: followService, blockService: blockService, storage: storage}
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
	BannedAt     *string   `json:"banned_at,omitempty" example:"2024-01-15T10:30:00Z"`
}

type SearchUserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	Bio       *string `json:"bio" example:"Movie enthusiast"`
}

type AdminSearchUserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"email" example:"user@example.com"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url" example:"https://example.com/avatar.jpg"`
	Bio       *string `json:"bio" example:"Movie enthusiast"`
	Role      string  `json:"role" example:"user"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	BannedAt  *string `json:"banned_at,omitempty" example:"2024-02-01T10:30:00Z"`
}

type UpdatePreferencesRequest struct {
	Theme  *string `json:"theme" binding:"omitempty,oneof=light dark system" example:"dark"`
	Locale *string `json:"locale" binding:"omitempty,oneof=en fr es" example:"fr"`
}

type UpdateUserRequest struct {
	Email       *string                   `json:"email" binding:"omitempty,email" example:"newemail@example.com"`
	Username    *string                   `json:"username" binding:"omitempty,min=3,max=50" example:"newusername"`
	Bio         *string                   `json:"bio" binding:"omitempty,max=500" example:"Updated bio"`
	Website     *string                   `json:"website" example:"https://newwebsite.com"`
	Preferences *UpdatePreferencesRequest `json:"preferences" binding:"omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"omitempty" example:"oldpassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72" example:"newpassword456"`
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
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if req.Website != nil && *req.Website != "" {
		u, err := url.ParseRequestURI(*req.Website)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			response.BadRequest(c, "Invalid website URL", nil)
			return
		}
	}

	input := ports.UpdateUserInput{
		Email:    req.Email,
		Username: req.Username,
		Bio:      req.Bio,
		Website:  req.Website,
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

// @Summary      Upload or update avatar
// @Description  Upload a new avatar image for the current user. Replaces existing avatar if one exists. Accepts JPEG, PNG, GIF, and WebP (max 5MB).
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        avatar formData file true "Avatar image file"
// @Success      200 {object} response.Response{data=UserResponse} "Updated user profile with new avatar"
// @Failure      400 {object} response.Response "Invalid file or file too large"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/avatar [put]
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "Avatar file is required", nil)
		return
	}
	defer file.Close()

	const maxSize = 5 << 20 // 5MB
	if header.Size > maxSize {
		response.BadRequest(c, "File too large, maximum size is 5MB", nil)
		return
	}

	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
	}

	ext, allowed := allowedTypes[contentType]
	if !allowed {
		response.BadRequest(c, "Invalid file type, allowed: JPEG, PNG, GIF, WebP", nil)
		return
	}

	ctx := c.Request.Context()

	// Delete old avatar from storage if exists
	user, err := h.userService.GetCurrentUser(ctx, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if user.AvatarURL != nil {
		h.storage.DeleteByURL(ctx, *user.AvatarURL)
	}

	objectName := fmt.Sprintf("%s_%d%s", userID.String(), time.Now().UnixNano(), ext)
	avatarURL, err := h.storage.Upload(ctx, objectName, file, header.Size, contentType)
	if err != nil {
		response.InternalError(c)
		return
	}

	updatedUser, err := h.userService.UpdateAvatar(ctx, userID, avatarURL)
	if err != nil {
		h.storage.Delete(ctx, objectName)
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

	response.Success(c, toUserResponse(updatedUser, stats))
}

// @Summary      Delete avatar
// @Description  Remove the avatar image of the current user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=UserResponse} "Updated user profile without avatar"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/avatar [delete]
func (h *UserHandler) DeleteAvatar(c *gin.Context) {
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

	if user.AvatarURL != nil {
		h.storage.DeleteByURL(ctx, *user.AvatarURL)
	}

	updatedUser, err := h.userService.DeleteAvatar(ctx, userID)
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

	response.Success(c, toUserResponse(updatedUser, stats))
}


// @Summary      Change or set password
// @Description  Change the password of the currently authenticated user. OAuth-only users can omit current_password to set a password for the first time.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ChangePasswordRequest true "New password (current_password optional for OAuth-only accounts)"
// @Success      200 {object} response.Response "Password changed successfully"
// @Failure      400 {object} response.Response "Invalid request body or password too short/long"
// @Failure      401 {object} response.Response "Unauthorized or incorrect current password"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.HandleValidationError(c, err) {
			return
		}
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.ChangePasswordInput{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	if err := h.userService.ChangePassword(c.Request.Context(), userID, input); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// @Summary      Delete current user
// @Description  Permanently delete the currently authenticated user's account. Super-admin accounts cannot be deleted.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      204 "Account deleted successfully"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Super-admin account cannot be deleted"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me [delete]
func (h *UserHandler) DeleteCurrentUser(c *gin.Context) {
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

	if err := h.userService.DeleteCurrentUser(ctx, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	if user.AvatarURL != nil {
		h.storage.DeleteByURL(ctx, *user.AvatarURL)
	}

	c.Status(204)
}

// @Summary      Get user by ID
// @Description  Get the public profile of a user by their ID. If authenticated, includes follow relationship info. Returns 403 if there is a block between the authenticated user and the target user.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Success      200 {object} response.Response{data=PublicUserResponse} "User public profile"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	idStr := c.Param("userId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	ctx := c.Request.Context()

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != id {
		if blocked, err := h.blockService.IsBlocked(ctx, currentUserID, id); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

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

func (h *UserHandler) getHiddenUserIDs(c *gin.Context) map[uuid.UUID]struct{} {
	hiddenSet := make(map[uuid.UUID]struct{})
	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		return hiddenSet
	}
	ctx := c.Request.Context()
	if blockerIDs, err := h.blockService.GetBlockerIDs(ctx, currentUserID); err == nil {
		for _, id := range blockerIDs {
			hiddenSet[id] = struct{}{}
		}
	}
	if blockedIDs, err := h.blockService.GetBlockedIDs(ctx, currentUserID); err == nil {
		for _, id := range blockedIDs {
			hiddenSet[id] = struct{}{}
		}
	}
	return hiddenSet
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
	resp := PublicUserResponse{
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
	if user.BannedAt != nil {
		bannedAt := user.BannedAt.Format(time.RFC3339)
		resp.BannedAt = &bannedAt
	}
	return resp
}

func toSearchUserResponse(user *domain.User) SearchUserResponse {
	return SearchUserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
	}
}

func toAdminSearchUserResponse(user *domain.User) AdminSearchUserResponse {
	resp := AdminSearchUserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Bio:       user.Bio,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
	if user.BannedAt != nil {
		bannedAt := user.BannedAt.Format(time.RFC3339)
		resp.BannedAt = &bannedAt
	}
	return resp
}

// @Summary      Search users
// @Description  Search and browse users by username with sorting and pagination. Regular users only see non-admin, non-banned accounts. Admins and super-admins see all accounts with additional details. If query is omitted, returns all visible users.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        query query string false "Search query (username)"
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 100)" default(20)
// @Param        sort query string false "Sort field with direction prefix (+asc, -desc)" Enums(+username, -username, +created_at, -created_at)
// @Success      200 {object} response.PaginatedResponse{data=[]SearchUserResponse} "Search results"
// @Success      200 {object} response.PaginatedResponse{data=[]AdminSearchUserResponse} "Search results (admin view)"
// @Failure      400 {object} response.Response "Invalid query parameters"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/search [get]
func (h *UserHandler) Search(c *gin.Context) {
	searchQuery := c.Query("query")

	params, err := query.Parse(c, query.Config{
		DefaultLimit: 20,
		MaxLimit:     100,
		AllowedSorts: []string{"username", "created_at"},
	})
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}

	callerRole, _ := middleware.GetRole(c)

	input := ports.SearchUsersInput{
		Query:      searchQuery,
		Offset:     params.Offset,
		Limit:      params.Limit,
		SortField:  params.SortField,
		SortOrder:  params.SortOrder,
		CallerRole: callerRole,
	}

	result, err := h.userService.SearchUsers(c.Request.Context(), input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := h.getHiddenUserIDs(c)

	isAdmin := callerRole == string(domain.UserRoleAdmin) || callerRole == string(domain.UserRoleSuperAdmin)

	if isAdmin {
		users := make([]AdminSearchUserResponse, 0, len(result.Users))
		hiddenCount := 0
		for _, user := range result.Users {
			if _, hidden := hiddenSet[user.ID]; hidden {
				hiddenCount++
				continue
			}
			users = append(users, toAdminSearchUserResponse(user))
		}
		response.SuccessPaginated(c, users, &response.Pagination{
			Offset: result.Offset,
			Limit:  result.Limit,
			Total:  result.Total - hiddenCount,
		})
		return
	}

	users := make([]SearchUserResponse, 0, len(result.Users))
	hiddenCount := 0
	for _, user := range result.Users {
		if _, hidden := hiddenSet[user.ID]; hidden {
			hiddenCount++
			continue
		}
		users = append(users, toSearchUserResponse(user))
	}

	response.SuccessPaginated(c, users, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total - hiddenCount,
	})
}
