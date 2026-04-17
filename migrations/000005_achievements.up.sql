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

-- Seed catalog. system = TRUE marks these as built-in: the API refuses to
-- update or delete them so they remain part of the product identity.
INSERT INTO achievements (code, name, description, category, tier, criterion, system, sort_order) VALUES
    ('first_review', 'First Impressions', 'Write your first review.', 'reviewing', 'bronze',
     '{"kind":"review_count","params":{"threshold":1}}', TRUE, 10),
    ('prolific_reviewer', 'Prolific Reviewer', 'Write 25 reviews.', 'reviewing', 'silver',
     '{"kind":"review_count","params":{"threshold":25}}', TRUE, 20),
    ('century_club_reviews', 'Century Club', 'Write 100 reviews.', 'reviewing', 'gold',
     '{"kind":"review_count","params":{"threshold":100}}', TRUE, 30),

    ('first_watch', 'Lights, Camera, Action', 'Add your first film to watched.', 'watching', 'bronze',
     '{"kind":"watched_count","params":{"threshold":1}}', TRUE, 110),
    ('cinephile', 'Cinephile', 'Watch 50 films.', 'watching', 'silver',
     '{"kind":"watched_count","params":{"threshold":50}}', TRUE, 120),
    ('movie_buff', 'Movie Buff', 'Watch 250 films.', 'watching', 'gold',
     '{"kind":"watched_count","params":{"threshold":250}}', TRUE, 130),
    ('marathon_runner', 'Marathon Runner', 'Watch 100 hours of films.', 'watching', 'silver',
     '{"kind":"watched_runtime","params":{"minutes":6000}}', TRUE, 140),
    ('screen_sage', 'Screen Sage', 'Watch 500 hours of films.', 'watching', 'platinum',
     '{"kind":"watched_runtime","params":{"minutes":30000}}', TRUE, 150),

    ('first_fan', 'First Fan', 'Receive your first like.', 'social', 'bronze',
     '{"kind":"likes_received","params":{"threshold":1}}', TRUE, 210),
    ('crowd_pleaser', 'Crowd Pleaser', 'Receive 100 likes across your reviews.', 'social', 'silver',
     '{"kind":"likes_received","params":{"threshold":100}}', TRUE, 220),
    ('first_follower', 'Making Friends', 'Reach 10 followers.', 'social', 'bronze',
     '{"kind":"followers_count","params":{"threshold":10}}', TRUE, 230),
    ('community_voice', 'Community Voice', 'Reach 100 followers.', 'social', 'gold',
     '{"kind":"followers_count","params":{"threshold":100}}', TRUE, 240),
    ('conversationalist', 'Conversationalist', 'Post 50 comments.', 'social', 'silver',
     '{"kind":"comments_authored","params":{"threshold":50}}', TRUE, 250),

    ('masterpiece_hunter', 'Masterpiece Hunter', 'Give ten 5-star ratings.', 'reviewing', 'silver',
     '{"kind":"rating_given","params":{"rating":5.0,"threshold":10}}', TRUE, 40),
    ('curator', 'Curator', 'Create 5 custom collections.', 'collecting', 'silver',
     '{"kind":"custom_collections","params":{"threshold":5}}', TRUE, 310);
