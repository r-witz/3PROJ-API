package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageReaction struct {
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
