package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ConversationResponse struct {
	OtherUser   *domain.User
	LastMessage *domain.Message
	UnreadCount int
}

type MessageService interface {
	SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, content string) (*domain.Message, error)
	GetConversation(ctx context.Context, userID, otherUserID uuid.UUID, offset, limit int) ([]*domain.Message, int, error)
	GetConversations(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ConversationResponse, int, error)
	MarkAsRead(ctx context.Context, userID, otherUserID uuid.UUID) error
}
