package domain

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	ReviewID         uuid.UUID `json:"review_id" db:"review_id"`
	Content          string    `json:"content" db:"content"`
	ContainsSpoilers bool      `json:"contains_spoilers" db:"contains_spoilers"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type CommentLike struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CommentID uuid.UUID `json:"comment_id" db:"comment_id"`
}
