package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/ports"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BlockHandler struct {
	blockService ports.BlockService
	hub          *ws.Hub
}

func NewBlockHandler(blockService ports.BlockService, hub *ws.Hub) *BlockHandler {
	return &BlockHandler{blockService: blockService, hub: hub}
}

type BlockedUserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	BlockedAt string  `json:"blocked_at" example:"2024-01-15T10:30:00Z"`
}

// @Summary      Block a user
// @Description  Block another user by their ID. This also removes any follow relationships between the two users.
// @Tags         blocks
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to block" format(uuid)
// @Success      204 "User blocked successfully"
// @Failure      400 {object} response.Response "Invalid user ID or cannot block self"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "User not found"
// @Failure      409 {object} response.Response "Already blocked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/block [post]
func (h *BlockHandler) BlockUser(c *gin.Context) {
	blockerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	blockedID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.blockService.BlockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		response.HandleError(c, err)
		return
	}

	h.hub.SendToUser(blockedID, ws.Event{
		Type: ws.EventMessagingBlocked,
		Data: ws.MessagingBlockedPayload{
			UserID: blockerID.String(),
			Reason: "blocked",
		},
	})

	c.Status(204)
}

// @Summary      Unblock a user
// @Description  Unblock a previously blocked user
// @Tags         blocks
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to unblock" format(uuid)
// @Success      204 "User unblocked successfully"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "User is not blocked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/block [delete]
func (h *BlockHandler) UnblockUser(c *gin.Context) {
	blockerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	blockedID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.blockService.UnblockUser(c.Request.Context(), blockerID, blockedID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Get blocked users
// @Description  Get the list of users blocked by the authenticated user
// @Tags         blocks
// @Produce      json
// @Security     BearerAuth
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]BlockedUserResponse} "List of blocked users"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/blocked [get]
func (h *BlockHandler) GetBlockedUsers(c *gin.Context) {
	blockerID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	offset, limit := parsePagination(c)

	result, err := h.blockService.GetBlockedUsers(c.Request.Context(), blockerID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	users := make([]BlockedUserResponse, len(result.Users))
	for i, u := range result.Users {
		resp := BlockedUserResponse{
			BlockedAt: u.BlockedAt,
		}
		if u.User != nil {
			resp.ID = u.User.ID.String()
			resp.Username = u.User.Username
			resp.AvatarURL = u.User.AvatarURL
		}
		users[i] = resp
	}

	response.SuccessPaginated(c, users, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total,
	})
}
