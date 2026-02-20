package handlers

import (
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageHandler struct {
	messageService ports.MessageService
}

func NewMessageHandler(messageService ports.MessageService) *MessageHandler {
	return &MessageHandler{messageService: messageService}
}

type SendMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000" example:"Hey, have you seen this movie?"`
}

type MessageResponse struct {
	ID         string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID   string  `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReceiverID string  `json:"receiver_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Content    string  `json:"content" example:"Hey, have you seen this movie?"`
	ReadAt     *string `json:"read_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CreatedAt  string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type LastMessagePreview struct {
	Content   string `json:"content" example:"Hey, have you seen this movie?"`
	IsOwn     bool   `json:"is_own" example:"true"`
	CreatedAt string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type ConversationListResponse struct {
	User        UserSummary        `json:"user"`
	LastMessage LastMessagePreview `json:"last_message"`
	UnreadCount int                `json:"unread_count" example:"3"`
}

// @Summary      Send a message
// @Description  Send a private message to another user. Both users must follow each other.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Receiver user ID" format(uuid)
// @Param        request body SendMessageRequest true "Message content"
// @Success      201 {object} response.Response{data=MessageResponse} "Message sent successfully"
// @Failure      400 {object} response.Response "Invalid request or cannot message self"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Not mutual follow"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId} [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	senderID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	receiverID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	message, err := h.messageService.SendMessage(c.Request.Context(), senderID, receiverID, req.Content)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, toMessageResponse(message))
}

// @Summary      Get conversation
// @Description  Get paginated messages between the authenticated user and another user. Returns the most recent messages first.
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Other user ID" format(uuid)
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]MessageResponse} "Conversation messages"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId} [get]
func (h *MessageHandler) GetConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	otherUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	offset, limit := parsePagination(c)

	messages, total, err := h.messageService.GetConversation(c.Request.Context(), userID, otherUserID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]MessageResponse, len(messages))
	for i, m := range messages {
		resp[i] = toMessageResponse(m)
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Get conversations
// @Description  Get list of all conversations for the authenticated user with last message preview and unread count
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]ConversationListResponse} "List of conversations"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages [get]
func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	offset, limit := parsePagination(c)

	conversations, total, err := h.messageService.GetConversations(c.Request.Context(), userID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]ConversationListResponse, len(conversations))
	for i, conv := range conversations {
		resp[i] = ConversationListResponse{
			LastMessage: LastMessagePreview{
				Content:   conv.LastMessage.Content,
				IsOwn:     conv.LastMessage.SenderID == userID,
				CreatedAt: conv.LastMessage.CreatedAt.Format(time.RFC3339),
			},
			UnreadCount: conv.UnreadCount,
		}
		if conv.OtherUser != nil {
			resp[i].User = UserSummary{
				ID:        conv.OtherUser.ID.String(),
				Username:  conv.OtherUser.Username,
				AvatarURL: conv.OtherUser.AvatarURL,
			}
		}
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Mark conversation as read
// @Description  Mark all messages from another user as read
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Other user ID" format(uuid)
// @Success      204 "Conversation marked as read"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId}/read [put]
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	otherUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.messageService.MarkAsRead(c.Request.Context(), userID, otherUserID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

func toMessageResponse(msg *domain.Message) MessageResponse {
	resp := MessageResponse{
		ID:         msg.ID.String(),
		SenderID:   msg.SenderID.String(),
		ReceiverID: msg.ReceiverID.String(),
		Content:    msg.Content,
		CreatedAt:  msg.CreatedAt.Format(time.RFC3339),
	}
	if msg.ReadAt != nil {
		readAt := msg.ReadAt.Format(time.RFC3339)
		resp.ReadAt = &readAt
	}
	return resp
}
