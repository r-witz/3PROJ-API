package ports

import (
	"context"
	"io"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type AttachmentInput struct {
	Reader      io.Reader
	FileName    string
	FileSize    int64
	ContentType string
}

type ConversationResponse struct {
	OtherUser   *domain.User
	LastMessage *domain.Message
	UnreadCount int
}

type MessageService interface {
	SendMessage(ctx context.Context, senderID, receiverID uuid.UUID, content *string, attachments []AttachmentInput) (*domain.Message, []*domain.MessageAttachment, error)
	GetConversation(ctx context.Context, userID, otherUserID uuid.UUID, offset, limit int) ([]*domain.Message, int, error)
	GetConversations(ctx context.Context, userID uuid.UUID, includeClosed bool, offset, limit int) ([]*ConversationResponse, int, error)
	MarkAsRead(ctx context.Context, userID, otherUserID uuid.UUID) error
	UpdateMessage(ctx context.Context, messageID, userID uuid.UUID, content string) (*domain.Message, error)
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*domain.MessageReaction, error)
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	CloseConversation(ctx context.Context, userID, otherUserID uuid.UUID) error
	ReopenConversation(ctx context.Context, userID, otherUserID uuid.UUID) error
	GetAttachmentsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageAttachment, error)
	GetReactionsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageReaction, error)
}
