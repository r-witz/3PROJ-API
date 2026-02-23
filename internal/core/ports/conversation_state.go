package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type ConversationStateRepository interface {
	Upsert(ctx context.Context, state *domain.ConversationState) error
	GetByUserAndOther(ctx context.Context, userID, otherUserID uuid.UUID) (*domain.ConversationState, error)
	GetClosedConversationPartnerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ClearClosedAt(ctx context.Context, userID, otherUserID uuid.UUID) error
}
