package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActivityType string

const (
	ActivityTypeReviewCreated       ActivityType = "review_created"
	ActivityTypeReviewUpdated       ActivityType = "review_updated"
	ActivityTypeCollectionCreated   ActivityType = "collection_created"
	ActivityTypeCollectionItemAdded ActivityType = "collection_item_added"
	ActivityTypeReviewLiked         ActivityType = "review_liked"
	ActivityTypeCommentLiked        ActivityType = "comment_liked"
	ActivityTypeUserFollowed        ActivityType = "user_followed"
	ActivityTypeUserUnfollowed      ActivityType = "user_unfollowed"
	ActivityTypeWatchlistItemAdded  ActivityType = "watchlist_item_added"
	ActivityTypeCommentCreated      ActivityType = "comment_created"
)

var ValidActivityTypes = map[ActivityType]bool{
	ActivityTypeReviewCreated:       true,
	ActivityTypeReviewUpdated:       true,
	ActivityTypeCollectionCreated:   true,
	ActivityTypeCollectionItemAdded: true,
	ActivityTypeReviewLiked:         true,
	ActivityTypeCommentLiked:        true,
	ActivityTypeUserFollowed:        true,
	ActivityTypeUserUnfollowed:      true,
	ActivityTypeWatchlistItemAdded:  true,
	ActivityTypeCommentCreated:      true,
}

type Activity struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	UserID       uuid.UUID    `json:"user_id" db:"user_id"`
	Type         ActivityType `json:"type" db:"type"`
	ReviewID     *uuid.UUID   `json:"review_id,omitempty" db:"review_id"`
	CollectionID *uuid.UUID   `json:"collection_id,omitempty" db:"collection_id"`
	CommentID    *uuid.UUID   `json:"comment_id,omitempty" db:"comment_id"`
	TMDBID       *int         `json:"tmdb_id,omitempty" db:"tmdb_id"`
	TargetUserID *uuid.UUID   `json:"target_user_id,omitempty" db:"target_user_id"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}
