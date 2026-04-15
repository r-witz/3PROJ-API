package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserDataExport struct {
	ExportedAt    time.Time            `json:"exported_at"`
	User          UserProfileExport    `json:"user"`
	Reviews       []Review             `json:"reviews"`
	Comments      []Comment            `json:"comments"`
	Collections   []CollectionExport   `json:"collections"`
	Messages      []Message            `json:"messages"`
	Followers     []uuid.UUID          `json:"followers"`
	Following     []uuid.UUID          `json:"following"`
	BlockedUsers  []uuid.UUID          `json:"blocked_users"`
	ReviewLikes   []ReviewLike         `json:"review_likes"`
	CommentLikes  []CommentLike        `json:"comment_likes"`
	Activities    []Activity           `json:"activities"`
	Notifications []Notification       `json:"notifications"`
	OAuthAccounts []OAuthAccountExport `json:"oauth_accounts"`
}

type UserProfileExport struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	Username      string     `json:"username"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Bio           *string    `json:"bio,omitempty"`
	Website       *string    `json:"website,omitempty"`
	Role          UserRole   `json:"role"`
	Theme         UserTheme  `json:"theme"`
	Locale        UserLocale `json:"locale"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CollectionExport struct {
	Collection Collection       `json:"collection"`
	Items      []CollectionItem `json:"items"`
}

type OAuthAccountExport struct {
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}
