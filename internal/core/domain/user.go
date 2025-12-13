package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type UserTheme string

const (
	UserThemeLight  UserTheme = "light"
	UserThemeDark   UserTheme = "dark"
	UserThemeSystem UserTheme = "system"
)

type UserLocale string

const (
	UserLocaleEN UserLocale = "en"
	UserLocaleFR UserLocale = "fr"
	UserLocaleES UserLocale = "es"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	PasswordHash *string    `json:"-" db:"password_hash"`
	Username     string     `json:"username" db:"username"`
	AvatarURL    *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	Bio          *string    `json:"bio,omitempty" db:"bio"`
	Website      *string    `json:"website,omitempty" db:"website"`
	Role         UserRole   `json:"role" db:"role"`
	Theme        UserTheme  `json:"theme" db:"theme"`
	Locale       UserLocale `json:"locale" db:"locale"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	BannedAt     *time.Time `json:"banned_at,omitempty" db:"banned_at"`
}
