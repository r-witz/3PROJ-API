package domain

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	TMDBID           int        `json:"tmdb_id" db:"tmdb_id"`
	Rating           float64    `json:"rating" db:"rating"`
	Content          *string    `json:"content,omitempty" db:"content"`
	ContainsSpoilers bool       `json:"contains_spoilers" db:"contains_spoilers"`
	FeaturedAt       *time.Time `json:"featured_at,omitempty" db:"featured_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type ReviewLike struct {
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ReviewID  uuid.UUID `json:"review_id" db:"review_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
