package handlers

import (
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageHandler struct {
	messageService ports.MessageService
	hub            *ws.Hub
}

func NewMessageHandler(messageService ports.MessageService, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{messageService: messageService, hub: hub}
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000" example:"Updated message content"`
}

type AddReactionRequest struct {
	Emoji string `json:"emoji" binding:"required,min=1,max=32" example:"👍"`
}

type RemoveReactionRequest struct {
	Emoji string `json:"emoji" binding:"required,min=1,max=32" example:"👍"`
}

type AttachmentResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FileURL     string `json:"file_url" example:"https://minio.example.com/bucket/messages/abc.jpg"`
	FileName    string `json:"file_name" example:"photo.jpg"`
	FileSize    int    `json:"file_size" example:"102400"`
	ContentType string `json:"content_type" example:"image/jpeg"`
	Position    int16  `json:"position" example:"0"`
}

type ReactionGroupResponse struct {
	Emoji string   `json:"emoji" example:"👍"`
	Count int      `json:"count" example:"2"`
	Users []string `json:"users"`
}

type MessageResponse struct {
	ID          string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID    string                  `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReceiverID  string                  `json:"receiver_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Content     *string                 `json:"content" example:"Hey, have you seen this movie?"`
	ReadAt      *string                 `json:"read_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CreatedAt   string                  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   *string                 `json:"updated_at,omitempty" example:"2024-01-15T11:00:00Z"`
	Attachments []AttachmentResponse    `json:"attachments,omitempty"`
	Reactions   []ReactionGroupResponse `json:"reactions,omitempty"`
}

type LastMessagePreview struct {
	Content   *string `json:"content" example:"Hey, have you seen this movie?"`
	IsOwn     bool    `json:"is_own" example:"true"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

type ConversationListResponse struct {
	User        UserSummary        `json:"user"`
	LastMessage LastMessagePreview `json:"last_message"`
	UnreadCount int                `json:"unread_count" example:"3"`
}

// @Summary      Send a message
// @Description  Send a private message to another user. Both users must follow each other. Supports multipart form with optional file attachments.
// @Tags         messages
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Receiver user ID" format(uuid)
// @Param        content formData string false "Message content"
// @Param        attachments formData file false "File attachments (max 10)"
// @Success      201 {object} response.Response{data=MessageResponse} "Message sent successfully"
// @Failure      400 {object} response.Response "Invalid request or cannot message self"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Not mutual follow or user blocked"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId} [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	senderID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	receiverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	var content *string
	if c.ContentType() == "application/json" {
		var req struct {
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.Content != "" {
			content = &req.Content
		}
	} else {
		if val := c.PostForm("content"); val != "" {
			content = &val
		}
	}

	var attachments []ports.AttachmentInput
	if c.Request.MultipartForm != nil && c.Request.MultipartForm.File != nil {
		files := c.Request.MultipartForm.File["attachments"]
		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				response.BadRequest(c, "Failed to read attachment", nil)
				return
			}
			defer file.Close()

			attachments = append(attachments, ports.AttachmentInput{
				Reader:      file,
				FileName:    fileHeader.Filename,
				FileSize:    fileHeader.Size,
				ContentType: fileHeader.Header.Get("Content-Type"),
			})
		}
	}

	message, createdAttachments, err := h.messageService.SendMessage(c.Request.Context(), senderID, receiverID, content, attachments)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := toMessageResponse(message)
	if len(createdAttachments) > 0 {
		resp.Attachments = make([]AttachmentResponse, len(createdAttachments))
		for i, a := range createdAttachments {
			resp.Attachments[i] = AttachmentResponse{
				ID:          a.ID.String(),
				FileURL:     a.FileURL,
				FileName:    a.FileName,
				FileSize:    a.FileSize,
				ContentType: a.ContentType,
				Position:    a.Position,
			}
		}
	}

	event := ws.Event{Type: ws.EventMessageNew, Data: resp}
	h.hub.SendToUser(receiverID, event)

	response.Created(c, resp)
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

	otherUserID, err := uuid.Parse(c.Param("id"))
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

	messageIDs := make([]uuid.UUID, len(messages))
	for i, m := range messages {
		messageIDs[i] = m.ID
	}

	attachmentsMap, _ := h.messageService.GetAttachmentsByMessageIDs(c.Request.Context(), messageIDs)
	reactionsMap, _ := h.messageService.GetReactionsByMessageIDs(c.Request.Context(), messageIDs)

	resp := make([]MessageResponse, len(messages))
	for i, m := range messages {
		resp[i] = toMessageResponse(m)
		if atts, ok := attachmentsMap[m.ID]; ok {
			resp[i].Attachments = toAttachmentResponses(atts)
		}
		if reactions, ok := reactionsMap[m.ID]; ok {
			resp[i].Reactions = toReactionGroupResponses(reactions)
		}
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
// @Param        include_closed query bool false "Include closed/archived conversations" default(false)
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
	includeClosed := c.Query("include_closed") == "true"

	conversations, total, err := h.messageService.GetConversations(c.Request.Context(), userID, includeClosed, offset, limit)
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

	otherUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.messageService.MarkAsRead(c.Request.Context(), userID, otherUserID); err != nil {
		response.HandleError(c, err)
		return
	}

	event := ws.Event{Type: ws.EventConversationRead, Data: ws.ConversationReadPayload{
		ReaderID: userID.String(),
		OtherID:  otherUserID.String(),
	}}
	h.hub.SendToUser(otherUserID, event)

	c.Status(204)
}

// @Summary      Update a message
// @Description  Update your own message content
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId path string true "Message ID" format(uuid)
// @Param        request body UpdateMessageRequest true "Updated message content"
// @Success      200 {object} response.Response{data=MessageResponse} "Updated message"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden - not the sender"
// @Failure      404 {object} response.Response "Message not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{messageId} [patch]
func (h *MessageHandler) UpdateMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid message ID", nil)
		return
	}

	var req UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	message, err := h.messageService.UpdateMessage(c.Request.Context(), messageID, userID, req.Content)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := toMessageResponse(message)
	event := ws.Event{Type: ws.EventMessageUpdated, Data: resp}
	h.hub.SendToUser(message.ReceiverID, event)

	response.Success(c, resp)
}

// @Summary      Delete a message
// @Description  Delete your own message
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        messageId path string true "Message ID" format(uuid)
// @Success      204 "Message deleted"
// @Failure      400 {object} response.Response "Invalid message ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden - not the sender"
// @Failure      404 {object} response.Response "Message not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{messageId} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid message ID", nil)
		return
	}

	message, err := h.messageService.DeleteMessage(c.Request.Context(), messageID, userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	event := ws.Event{Type: ws.EventMessageDeleted, Data: ws.MessageDeletedPayload{
		MessageID:  message.ID.String(),
		SenderID:   message.SenderID.String(),
		ReceiverID: message.ReceiverID.String(),
	}}
	h.hub.SendToUser(message.ReceiverID, event)

	c.Status(204)
}

// @Summary      Add a reaction to a message
// @Description  Add an emoji reaction to a message. Only participants of the conversation can react.
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId path string true "Message ID" format(uuid)
// @Param        request body AddReactionRequest true "Reaction emoji"
// @Success      201 {object} response.Response "Reaction added"
// @Failure      400 {object} response.Response "Invalid request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Not a participant or user blocked"
// @Failure      404 {object} response.Response "Message not found"
// @Failure      409 {object} response.Response "Reaction already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{messageId}/reactions [post]
func (h *MessageHandler) AddReaction(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid message ID", nil)
		return
	}

	var req AddReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	_, err = h.messageService.AddReaction(c.Request.Context(), messageID, userID, req.Emoji)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if msg, err := h.messageService.GetMessageByID(c.Request.Context(), messageID, userID); err == nil {
		event := ws.Event{Type: ws.EventReactionAdded, Data: ws.ReactionPayload{
			MessageID:  messageID.String(),
			SenderID:   msg.SenderID.String(),
			ReceiverID: msg.ReceiverID.String(),
			UserID:     userID.String(),
			Emoji:      req.Emoji,
		}}
		otherID := msg.SenderID
		if msg.SenderID == userID {
			otherID = msg.ReceiverID
		}
		h.hub.SendToUser(otherID, event)
	}

	c.Status(201)
}

// @Summary      Remove a reaction from a message
// @Description  Remove your emoji reaction from a message
// @Tags         messages
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        messageId path string true "Message ID" format(uuid)
// @Param        request body RemoveReactionRequest true "Reaction emoji to remove"
// @Success      204 "Reaction removed"
// @Failure      400 {object} response.Response "Invalid request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Reaction not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{messageId}/reactions [delete]
func (h *MessageHandler) RemoveReaction(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	messageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid message ID", nil)
		return
	}

	var req RemoveReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.messageService.RemoveReaction(c.Request.Context(), messageID, userID, req.Emoji); err != nil {
		response.HandleError(c, err)
		return
	}

	if msg, err := h.messageService.GetMessageByID(c.Request.Context(), messageID, userID); err == nil {
		event := ws.Event{Type: ws.EventReactionRemoved, Data: ws.ReactionPayload{
			MessageID:  messageID.String(),
			SenderID:   msg.SenderID.String(),
			ReceiverID: msg.ReceiverID.String(),
			UserID:     userID.String(),
			Emoji:      req.Emoji,
		}}
		otherID := msg.SenderID
		if msg.SenderID == userID {
			otherID = msg.ReceiverID
		}
		h.hub.SendToUser(otherID, event)
	}

	c.Status(204)
}

// @Summary      Close a conversation
// @Description  Close/archive a conversation with another user. The conversation will be hidden from the default list.
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Other user ID" format(uuid)
// @Success      204 "Conversation closed"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      409 {object} response.Response "Conversation already closed"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId}/close [post]
func (h *MessageHandler) CloseConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	otherUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.messageService.CloseConversation(c.Request.Context(), userID, otherUserID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Reopen a conversation
// @Description  Reopen a previously closed/archived conversation
// @Tags         messages
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "Other user ID" format(uuid)
// @Success      204 "Conversation reopened"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      409 {object} response.Response "Conversation is not closed"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /messages/{userId}/close [delete]
func (h *MessageHandler) ReopenConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	otherUserID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.messageService.ReopenConversation(c.Request.Context(), userID, otherUserID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

func toAttachmentResponses(attachments []*domain.MessageAttachment) []AttachmentResponse {
	resp := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		resp[i] = AttachmentResponse{
			ID:          a.ID.String(),
			FileURL:     a.FileURL,
			FileName:    a.FileName,
			FileSize:    a.FileSize,
			ContentType: a.ContentType,
			Position:    a.Position,
		}
	}
	return resp
}

func toReactionGroupResponses(reactions []*domain.MessageReaction) []ReactionGroupResponse {
	groups := make(map[string]*ReactionGroupResponse)
	order := make([]string, 0)
	for _, r := range reactions {
		if g, ok := groups[r.Emoji]; ok {
			g.Count++
			g.Users = append(g.Users, r.UserID.String())
		} else {
			groups[r.Emoji] = &ReactionGroupResponse{
				Emoji: r.Emoji,
				Count: 1,
				Users: []string{r.UserID.String()},
			}
			order = append(order, r.Emoji)
		}
	}
	resp := make([]ReactionGroupResponse, len(order))
	for i, emoji := range order {
		resp[i] = *groups[emoji]
	}
	return resp
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
	if msg.UpdatedAt != nil {
		updatedAt := msg.UpdatedAt.Format(time.RFC3339)
		resp.UpdatedAt = &updatedAt
	}
	return resp
}
