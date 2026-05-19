package domain

import (
	"time"

	"github.com/google/uuid"
)

type CollectionType string

const (
	CollectionTypeSystem CollectionType = "system"
	CollectionTypeCustom CollectionType = "custom"
)

type CollectionVisibility string

const (
	CollectionVisibilityPublic  CollectionVisibility = "public"
	CollectionVisibilityPrivate CollectionVisibility = "private"
)

const SystemCollectionWatched = "watched"

type Collection struct {
	ID          uuid.UUID            `json:"id" db:"id"`
	UserID      uuid.UUID            `json:"user_id" db:"user_id"`
	Name        string               `json:"name" db:"name"`
	Slug        string               `json:"slug" db:"slug"`
	Type        CollectionType       `json:"type" db:"type"`
	Visibility  CollectionVisibility `json:"visibility" db:"visibility"`
	Description *string              `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" db:"updated_at"`
}
