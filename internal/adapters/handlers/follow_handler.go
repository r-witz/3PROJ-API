package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FollowHandler struct {
	followService ports.FollowService
	blockService  ports.BlockService
	hub           *ws.Hub
	banCache      ports.BanCache
	notifService  ports.NotificationService
}

func NewFollowHandler(followService ports.FollowService, blockService ports.BlockService, hub *ws.Hub, banCache ports.BanCache, notifService ports.NotificationService) *FollowHandler {
	return &FollowHandler{followService: followService, blockService: blockService, hub: hub, banCache: banCache, notifService: notifService}
}

type FollowUserResponse struct {
	ID         string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username   string  `json:"username" example:"johndoe"`
	AvatarURL  *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Bio        *string `json:"bio,omitempty" example:"Movie enthusiast"`
	FollowedAt string  `json:"followed_at" example:"2024-01-15T10:30:00Z"`
}

// @Summary      Follow a user
// @Description  Follow another user by their ID. Returns 403 if there is a block between the two users.
// @Tags         follows
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to follow" format(uuid)
// @Success      204 "Followed successfully"
// @Failure      400 {object} response.Response "Invalid user ID or cannot follow self"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "User not found"
// @Failure      409 {object} response.Response "Already following"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/follow [post]
func (h *FollowHandler) Follow(c *gin.Context) {
	followerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	followingID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), followerID, followingID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	if err := h.followService.Follow(c.Request.Context(), followerID, followingID); err != nil {
		response.HandleError(c, err)
		return
	}

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:       middleware.ActivityCreate,
		Type:         domain.ActivityTypeUserFollowed,
		UserID:       followerID,
		TargetUserID: &followingID,
	})

	notif, _ := h.notifService.Notify(c.Request.Context(), ports.NotifyInput{
		UserID:  followingID,
		ActorID: followerID,
		Type:    domain.NotificationTypeNewFollow,
	})
	if notif != nil {
		h.hub.SendToUser(followingID, ws.Event{
			Type: ws.EventNotificationNew,
			Data: notif,
		})
	}

	if mutual, _ := h.followService.IsFollowing(c.Request.Context(), followingID, followerID); mutual {
		event := ws.Event{
			Type: ws.EventMessagingUnblocked,
			Data: ws.MessagingUnblockedPayload{
				UserID: followerID.String(),
				Reason: "mutual_follow",
			},
		}
		h.hub.SendToUser(followingID, event)
		event.Data = ws.MessagingUnblockedPayload{
			UserID: followingID.String(),
			Reason: "mutual_follow",
		}
		h.hub.SendToUser(followerID, event)
	}

	c.Status(204)
}

// @Summary      Unfollow a user
// @Description  Unfollow a user by their ID
// @Tags         follows
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to unfollow" format(uuid)
// @Success      204 "Unfollowed successfully"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Not following this user"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/follow [delete]
func (h *FollowHandler) Unfollow(c *gin.Context) {
	followerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	followingID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.followService.Unfollow(c.Request.Context(), followerID, followingID); err != nil {
		response.HandleError(c, err)
		return
	}

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:       middleware.ActivityCreate,
		Type:         domain.ActivityTypeUserUnfollowed,
		UserID:       followerID,
		TargetUserID: &followingID,
	})

	h.hub.SendToUser(followingID, ws.Event{
		Type: ws.EventMessagingBlocked,
		Data: ws.MessagingBlockedPayload{
			UserID: followerID.String(),
			Reason: "unfollowed",
		},
	})

	c.Status(204)
}

// @Summary      Remove a follower
// @Description  Remove a user from your followers list
// @Tags         follows
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID of the follower to remove" format(uuid)
// @Success      204 "Follower removed successfully"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "User is not following you"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/followers [delete]
func (h *FollowHandler) RemoveFollower(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	followerID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.followService.RemoveFollower(c.Request.Context(), userID, followerID); err != nil {
		response.HandleError(c, err)
		return
	}

	h.hub.SendToUser(followerID, ws.Event{
		Type: ws.EventMessagingBlocked,
		Data: ws.MessagingBlockedPayload{
			UserID: userID.String(),
			Reason: "follower_removed",
		},
	})

	c.Status(204)
}

// @Summary      Get followers
// @Description  Get the paginated list of followers for a user. Optionally filter by username. If authenticated, users involved in a block relationship with the current user are excluded from results. Banned users are always excluded from the list and counts.
// @Tags         follows
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        q query string false "Search query to filter followers by username"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]FollowUserResponse} "List of followers"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/followers [get]
func (h *FollowHandler) GetFollowers(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	search := c.Query("q")
	offset, limit := parsePagination(c)

	result, err := h.followService.GetFollowers(c.Request.Context(), userID, search, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := GetHiddenUserIDs(c, h.blockService, h.banCache)

	users := make([]FollowUserResponse, 0, len(result.Users))
	hiddenCount := 0
	for _, u := range result.Users {
		if _, hidden := hiddenSet[u.User.ID]; hidden {
			hiddenCount++
			continue
		}
		if u.User.Role == domain.UserRoleSuperAdmin {
			hiddenCount++
			continue
		}
		users = append(users, toFollowUserResponse(u))
	}

	response.SuccessPaginated(c, users, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total - hiddenCount,
	})
}

// @Summary      Get following
// @Description  Get the paginated list of users that a user is following. Optionally filter by username. If authenticated, users involved in a block relationship with the current user are excluded from results. Banned users are always excluded from the list and counts.
// @Tags         follows
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        q query string false "Search query to filter following by username"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]FollowUserResponse} "List of following"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/following [get]
func (h *FollowHandler) GetFollowing(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	search := c.Query("q")
	offset, limit := parsePagination(c)

	result, err := h.followService.GetFollowing(c.Request.Context(), userID, search, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := GetHiddenUserIDs(c, h.blockService, h.banCache)

	users := make([]FollowUserResponse, 0, len(result.Users))
	hiddenCount := 0
	for _, u := range result.Users {
		if _, hidden := hiddenSet[u.User.ID]; hidden {
			hiddenCount++
			continue
		}
		if u.User.Role == domain.UserRoleSuperAdmin {
			hiddenCount++
			continue
		}
		users = append(users, toFollowUserResponse(u))
	}

	response.SuccessPaginated(c, users, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total - hiddenCount,
	})
}

func toFollowUserResponse(summary *ports.FollowUserSummary) FollowUserResponse {
	return FollowUserResponse{
		ID:         summary.User.ID.String(),
		Username:   summary.User.Username,
		AvatarURL:  summary.User.AvatarURL,
		Bio:        summary.User.Bio,
		FollowedAt: summary.FollowedAt,
	}
}
