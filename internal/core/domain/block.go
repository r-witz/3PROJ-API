package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserBlock struct {
	BlockerID uuid.UUID `json:"blocker_id" db:"blocker_id"`
	BlockedID uuid.UUID `json:"blocked_id" db:"blocked_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
