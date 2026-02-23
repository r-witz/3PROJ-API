package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConversationState struct {
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	OtherUserID uuid.UUID  `json:"other_user_id" db:"other_user_id"`
	ClosedAt    *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}
