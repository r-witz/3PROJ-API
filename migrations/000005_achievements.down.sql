ALTER TABLE notification_preferences DROP COLUMN IF EXISTS achievement_unlocked;

ALTER TABLE notifications DROP COLUMN IF EXISTS achievement_id;

DROP INDEX IF EXISTS idx_user_achievements_user_unlocked;
DROP TABLE IF EXISTS user_achievements;

DROP INDEX IF EXISTS idx_achievements_category_active;
DROP TABLE IF EXISTS achievements;

DROP TYPE IF EXISTS achievement_tier;

-- Note: ENUM values cannot be dropped in PostgreSQL without recreating the type.
-- The 'achievement_unlocked' value remains on notification_type; this is harmless.
