CREATE TYPE achievement_tier AS ENUM ('bronze', 'silver', 'gold', 'platinum');

ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'achievement_unlocked';

CREATE TABLE achievements (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code VARCHAR(80) UNIQUE NOT NULL,
    name VARCHAR(120) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(40) NOT NULL,
    tier achievement_tier NOT NULL DEFAULT 'bronze',
    icon_url TEXT,
    criterion JSONB NOT NULL,
    secret BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    system BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_achievements_category_active ON achievements(category) WHERE active;

CREATE TABLE user_achievements (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    achievement_id UUID NOT NULL REFERENCES achievements(id) ON DELETE CASCADE,
    unlocked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, achievement_id)
);

CREATE INDEX idx_user_achievements_user_unlocked ON user_achievements(user_id, unlocked_at DESC);

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS achievement_id UUID REFERENCES achievements(id) ON DELETE SET NULL;

ALTER TABLE notification_preferences
    ADD COLUMN IF NOT EXISTS achievement_unlocked BOOLEAN NOT NULL DEFAULT TRUE;

-- Seed catalog: four grind ladders, each with all four tiers so every goal
-- offers a clear progression path. system = TRUE marks these as built-in so
-- the API refuses to update or delete them.
INSERT INTO achievements (code, name, description, category, tier, criterion, system, sort_order) VALUES
    -- Reviews ladder
    ('first_review', 'First Impressions', 'Write your first review.', 'reviewing', 'bronze',
     '{"kind":"review_count","params":{"threshold":1}}', TRUE, 10),
    ('prolific_reviewer', 'Prolific Reviewer', 'Write 25 reviews.', 'reviewing', 'silver',
     '{"kind":"review_count","params":{"threshold":25}}', TRUE, 20),
    ('century_club_reviews', 'Century Club', 'Write 100 reviews.', 'reviewing', 'gold',
     '{"kind":"review_count","params":{"threshold":100}}', TRUE, 30),
    ('scribe_laureate', 'Scribe Laureate', 'Write 500 reviews.', 'reviewing', 'platinum',
     '{"kind":"review_count","params":{"threshold":500}}', TRUE, 40),

    -- Watching ladder
    ('first_watch', 'Lights, Camera', 'Add your first film to watched.', 'watching', 'bronze',
     '{"kind":"watched_count","params":{"threshold":1}}', TRUE, 110),
    ('cinephile', 'Cinephile', 'Watch 50 films.', 'watching', 'silver',
     '{"kind":"watched_count","params":{"threshold":50}}', TRUE, 120),
    ('movie_buff', 'Movie Buff', 'Watch 250 films.', 'watching', 'gold',
     '{"kind":"watched_count","params":{"threshold":250}}', TRUE, 130),
    ('silver_screener', 'Silver Screener', 'Watch 1000 films.', 'watching', 'platinum',
     '{"kind":"watched_count","params":{"threshold":1000}}', TRUE, 140),

    -- Likes received ladder
    ('first_fan', 'First Fan', 'Receive your first like.', 'social', 'bronze',
     '{"kind":"likes_received","params":{"threshold":1}}', TRUE, 210),
    ('crowd_pleaser', 'Crowd Pleaser', 'Receive 100 likes across your reviews.', 'social', 'silver',
     '{"kind":"likes_received","params":{"threshold":100}}', TRUE, 220),
    ('beloved', 'Beloved', 'Receive 500 likes across your reviews.', 'social', 'gold',
     '{"kind":"likes_received","params":{"threshold":500}}', TRUE, 230),
    ('community_legend', 'Community Legend', 'Receive 2000 likes across your reviews.', 'social', 'platinum',
     '{"kind":"likes_received","params":{"threshold":2000}}', TRUE, 240),

    -- Followers ladder
    ('first_follower', 'Making Friends', 'Reach 5 followers.', 'social', 'bronze',
     '{"kind":"followers_count","params":{"threshold":5}}', TRUE, 310),
    ('growing_voice', 'Growing Voice', 'Reach 50 followers.', 'social', 'silver',
     '{"kind":"followers_count","params":{"threshold":50}}', TRUE, 320),
    ('community_voice', 'Community Voice', 'Reach 250 followers.', 'social', 'gold',
     '{"kind":"followers_count","params":{"threshold":250}}', TRUE, 330),
    ('icon_status', 'Icon Status', 'Reach 1000 followers.', 'social', 'platinum',
     '{"kind":"followers_count","params":{"threshold":1000}}', TRUE, 340),

    -- Five-star ladder
    ('first_five_star', 'Love at First Sight', 'Give your first 5-star rating.', 'reviewing', 'bronze',
     '{"kind":"rating_given","params":{"rating":5.0,"threshold":1}}', TRUE, 410),
    ('masterpiece_hunter', 'Masterpiece Hunter', 'Give ten 5-star ratings.', 'reviewing', 'silver',
     '{"kind":"rating_given","params":{"rating":5.0,"threshold":10}}', TRUE, 420),
    ('taste_maker', 'Taste Maker', 'Give fifty 5-star ratings.', 'reviewing', 'gold',
     '{"kind":"rating_given","params":{"rating":5.0,"threshold":50}}', TRUE, 430),
    ('pantheon_keeper', 'Pantheon Keeper', 'Give 250 five-star ratings.', 'reviewing', 'platinum',
     '{"kind":"rating_given","params":{"rating":5.0,"threshold":250}}', TRUE, 440),

    -- Watch time ladder (runtime expressed in minutes)
    ('warm_up', 'Warm-Up', 'Watch 10 hours of films.', 'watching', 'bronze',
     '{"kind":"watched_runtime","params":{"minutes":600}}', TRUE, 510),
    ('marathon_runner', 'Marathon Runner', 'Watch 100 hours of films.', 'watching', 'silver',
     '{"kind":"watched_runtime","params":{"minutes":6000}}', TRUE, 520),
    ('binge_artisan', 'Binge Artisan', 'Watch 500 hours of films.', 'watching', 'gold',
     '{"kind":"watched_runtime","params":{"minutes":30000}}', TRUE, 530),
    ('screen_sage', 'Screen Sage', 'Watch 2000 hours of films.', 'watching', 'platinum',
     '{"kind":"watched_runtime","params":{"minutes":120000}}', TRUE, 540);
