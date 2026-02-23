package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type MessageReactionRepository interface {
	Create(ctx context.Context, reaction *domain.MessageReaction) error
	Delete(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	GetByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageReaction, error)
	GetByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageReaction, error)
	Exists(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error)
}
