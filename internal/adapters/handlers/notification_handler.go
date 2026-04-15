package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notifService ports.NotificationService
}

func NewNotificationHandler(notifService ports.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

type NotificationResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ActorID   *string `json:"actor_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	Type      string  `json:"type" example:"new_follow"`
	ReviewID  *string `json:"review_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	CommentID *string `json:"comment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	Message   *string `json:"message,omitempty" example:"Welcome to Duskforge!"`
	ReadAt    *string `json:"read_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type NotificationPreferencesResponse struct {
	LikeReview  bool   `json:"like_review" example:"true"`
	LikeComment bool   `json:"like_comment" example:"true"`
	NewComment  bool   `json:"new_comment" example:"true"`
	NewFollow   bool   `json:"new_follow" example:"true"`
	System      bool   `json:"system" example:"true"`
	UpdatedAt   string `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

type UpdateNotificationPreferencesRequest struct {
	LikeReview  *bool `json:"like_review" example:"true"`
	LikeComment *bool `json:"like_comment" example:"true"`
	NewComment  *bool `json:"new_comment" example:"true"`
	NewFollow   *bool `json:"new_follow" example:"false"`
	System      *bool `json:"system" example:"true"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count" example:"5"`
}

// @Summary      Get notifications
// @Description  Get the authenticated user's notifications with pagination
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]NotificationResponse} "List of notifications"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	offset, limit := parsePagination(c)

	notifications, total, err := h.notifService.GetByUserID(c.Request.Context(), userID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		resp = append(resp, toNotificationResponse(n))
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Get unread notification count
// @Description  Get the number of unread notifications for the authenticated user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=UnreadCountResponse} "Unread count"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/unread/count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	count, err := h.notifService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, UnreadCountResponse{UnreadCount: count})
}

// @Summary      Mark all notifications as read
// @Description  Mark all unread notifications as read for the authenticated user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      204 "All notifications marked as read"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/read [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.notifService.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Mark a notification as read
// @Description  Mark a specific notification as read
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        notificationId path string true "Notification ID" format(uuid)
// @Success      204 "Notification marked as read"
// @Failure      400 {object} response.Response "Invalid notification ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Notification not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/{notificationId}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("notificationId"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID", nil)
		return
	}

	if err := h.notifService.MarkAsRead(c.Request.Context(), notificationID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Delete all notifications
// @Description  Delete all notifications for the authenticated user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      204 "All notifications deleted"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications [delete]
func (h *NotificationHandler) DeleteAll(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	if err := h.notifService.DeleteAll(c.Request.Context(), userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Delete a notification
// @Description  Delete a specific notification
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        notificationId path string true "Notification ID" format(uuid)
// @Success      204 "Notification deleted"
// @Failure      400 {object} response.Response "Invalid notification ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Notification not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/{notificationId} [delete]
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	notificationID, err := uuid.Parse(c.Param("notificationId"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID", nil)
		return
	}

	if err := h.notifService.Delete(c.Request.Context(), notificationID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Get notification preferences
// @Description  Get the authenticated user's notification preferences
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=NotificationPreferencesResponse} "Notification preferences"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/preferences [get]
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	prefs, err := h.notifService.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toNotificationPreferencesResponse(prefs))
}

// @Summary      Update notification preferences
// @Description  Update the authenticated user's notification preferences. Only provided fields are updated.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body UpdateNotificationPreferencesRequest true "Preferences to update"
// @Success      200 {object} response.Response{data=NotificationPreferencesResponse} "Updated preferences"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /notifications/preferences [patch]
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.HandleValidationError(c, err)
		return
	}

	prefs, err := h.notifService.UpdatePreferences(c.Request.Context(), userID, ports.UpdateNotificationPreferencesInput{
		LikeReview:  req.LikeReview,
		LikeComment: req.LikeComment,
		NewComment:  req.NewComment,
		NewFollow:   req.NewFollow,
		System:      req.System,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toNotificationPreferencesResponse(prefs))
}

func toNotificationResponse(n *domain.Notification) NotificationResponse {
	resp := NotificationResponse{
		ID:        n.ID.String(),
		UserID:    n.UserID.String(),
		Type:      string(n.Type),
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if n.ActorID != nil {
		s := n.ActorID.String()
		resp.ActorID = &s
	}
	if n.ReviewID != nil {
		s := n.ReviewID.String()
		resp.ReviewID = &s
	}
	if n.CommentID != nil {
		s := n.CommentID.String()
		resp.CommentID = &s
	}
	if n.Message != nil {
		resp.Message = n.Message
	}
	if n.ReadAt != nil {
		s := n.ReadAt.Format("2006-01-02T15:04:05Z")
		resp.ReadAt = &s
	}
	return resp
}

func toNotificationPreferencesResponse(p *domain.NotificationPreferences) NotificationPreferencesResponse {
	return NotificationPreferencesResponse{
		LikeReview:  p.LikeReview,
		LikeComment: p.LikeComment,
		NewComment:  p.NewComment,
		NewFollow:   p.NewFollow,
		System:      p.System,
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
