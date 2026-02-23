package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type MessageAttachmentRepository interface {
	CreateBatch(ctx context.Context, attachments []*domain.MessageAttachment) error
	GetByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageAttachment, error)
	GetByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageAttachment, error)
	DeleteByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageAttachment, error)
}
