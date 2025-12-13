package domain

import (
	"time"

	"github.com/google/uuid"
)

type OAuthAccount struct {
	Provider       string    `json:"provider" db:"provider"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
