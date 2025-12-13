package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActivityType string

const (
	ActivityTypeReviewCreated       ActivityType = "review_created"
	ActivityTypeCollectionCreated   ActivityType = "collection_created"
	ActivityTypeCollectionItemAdded ActivityType = "collection_item_added"
)

type Activity struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	UserID       uuid.UUID    `json:"user_id" db:"user_id"`
	Type         ActivityType `json:"type" db:"type"`
	ReviewID     *uuid.UUID   `json:"review_id,omitempty" db:"review_id"`
	CollectionID *uuid.UUID   `json:"collection_id,omitempty" db:"collection_id"`
	TMDBID       *int         `json:"tmdb_id,omitempty" db:"tmdb_id"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}
