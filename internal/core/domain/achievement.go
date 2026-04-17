package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AchievementTier string

const (
	AchievementTierBronze   AchievementTier = "bronze"
	AchievementTierSilver   AchievementTier = "silver"
	AchievementTierGold     AchievementTier = "gold"
	AchievementTierPlatinum AchievementTier = "platinum"
)

var ValidAchievementTiers = map[AchievementTier]bool{
	AchievementTierBronze:   true,
	AchievementTierSilver:   true,
	AchievementTierGold:     true,
	AchievementTierPlatinum: true,
}

type AchievementCategory string

const (
	AchievementCategoryReviewing  AchievementCategory = "reviewing"
	AchievementCategoryWatching   AchievementCategory = "watching"
	AchievementCategorySocial     AchievementCategory = "social"
	AchievementCategoryCollecting AchievementCategory = "collecting"
	AchievementCategoryDiscovery  AchievementCategory = "discovery"
)

var ValidAchievementCategories = map[AchievementCategory]bool{
	AchievementCategoryReviewing:  true,
	AchievementCategoryWatching:   true,
	AchievementCategorySocial:     true,
	AchievementCategoryCollecting: true,
	AchievementCategoryDiscovery:  true,
}

type Achievement struct {
	ID          uuid.UUID           `json:"id" db:"id"`
	Code        string              `json:"code" db:"code"`
	Name        string              `json:"name" db:"name"`
	Description string              `json:"description" db:"description"`
	Category    AchievementCategory `json:"category" db:"category"`
	Tier        AchievementTier     `json:"tier" db:"tier"`
	IconURL     *string             `json:"icon_url,omitempty" db:"icon_url"`
	Criterion   json.RawMessage     `json:"criterion" db:"criterion"`
	Secret      bool                `json:"secret" db:"secret"`
	Active      bool                `json:"active" db:"active"`
	System      bool                `json:"system" db:"system"`
	SortOrder   int                 `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" db:"updated_at"`
}

type UserAchievement struct {
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	AchievementID uuid.UUID `json:"achievement_id" db:"achievement_id"`
	UnlockedAt    time.Time `json:"unlocked_at" db:"unlocked_at"`
}
