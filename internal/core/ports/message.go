package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ConversationPreview struct {
	OtherUserID uuid.UUID
	LastMessage *domain.Message
	UnreadCount int
}

type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
	GetConversation(ctx context.Context, userID1, userID2 uuid.UUID) ([]*domain.Message, error)
	GetConversationPaginated(ctx context.Context, userID1, userID2 uuid.UUID, offset, limit int) ([]*domain.Message, int, error)
	GetConversations(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ConversationPreview, int, error)
	GetConversationsFiltered(ctx context.Context, userID uuid.UUID, excludeUserIDs []uuid.UUID, offset, limit int) ([]*ConversationPreview, int, error)
	MarkConversationAsRead(ctx context.Context, userID, otherUserID uuid.UUID) error
	Update(ctx context.Context, message *domain.Message) error
	Delete(ctx context.Context, id uuid.UUID) error
}
