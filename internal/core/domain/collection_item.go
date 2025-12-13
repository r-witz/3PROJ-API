package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CollectionItem struct {
	CollectionID uuid.UUID       `json:"collection_id" db:"collection_id"`
	TMDBID       int             `json:"tmdb_id" db:"tmdb_id"`
	AddedAt      time.Time       `json:"added_at" db:"added_at"`
	Runtime      int16           `json:"runtime" db:"runtime"`
	Metadata     json.RawMessage `json:"metadata" db:"metadata"`
}
