package domain

import (
	"time"

	"github.com/google/uuid"
)

type MessageAttachment struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MessageID   uuid.UUID `json:"message_id" db:"message_id"`
	FileURL     string    `json:"file_url" db:"file_url"`
	FileName    string    `json:"file_name" db:"file_name"`
	FileSize    int       `json:"file_size" db:"file_size"`
	ContentType string    `json:"content_type" db:"content_type"`
	Position    int16     `json:"position" db:"position"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
