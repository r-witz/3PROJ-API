package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	SenderID   uuid.UUID  `json:"sender_id" db:"sender_id"`
	ReceiverID uuid.UUID  `json:"receiver_id" db:"receiver_id"`
	Content    *string    `json:"content" db:"content"`
	ReadAt     *time.Time `json:"read_at,omitempty" db:"read_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}
