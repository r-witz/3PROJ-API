-- ENUMS
CREATE TYPE user_role AS ENUM ('user', 'admin');
CREATE TYPE user_theme AS ENUM ('light', 'dark', 'system');
CREATE TYPE user_locale AS ENUM ('en', 'fr', 'es');
CREATE TYPE collection_type AS ENUM ('system', 'custom');
CREATE TYPE collection_visibility AS ENUM ('public', 'private');
CREATE TYPE notification_type AS ENUM ('like_review', 'like_comment', 'new_comment', 'new_follow', 'system');
CREATE TYPE report_reason AS ENUM ('spam', 'harassment', 'spoiler', 'inappropriate', 'other');
CREATE TYPE report_status_type AS ENUM ('pending', 'resolved', 'dismissed');

-- USERS TABLE
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    username VARCHAR(50) UNIQUE NOT NULL,
    avatar_url TEXT,
    bio TEXT,
    website TEXT,
    role user_role NOT NULL DEFAULT 'user',
    theme user_theme NOT NULL DEFAULT 'system',
    locale user_locale NOT NULL DEFAULT 'en',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    banned_at TIMESTAMP WITH TIME ZONE
);

-- OAUTH ACCOUNTS
CREATE TABLE oauth_accounts (
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (provider, provider_user_id),
    UNIQUE(user_id, provider)
);

-- SESSIONS / REFRESH TOKENS
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- COLLECTIONS
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    type collection_type NOT NULL DEFAULT 'custom',
    visibility collection_visibility NOT NULL DEFAULT 'public',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- COLLECTION ITEMS
CREATE TABLE collection_items (
    collection_id UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tmdb_id INT NOT NULL,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    runtime SMALLINT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (collection_id, tmdb_id)
);

-- REVIEWS
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tmdb_id INT NOT NULL,
    rating NUMERIC(2,1) NOT NULL CHECK (rating >= 0.5 AND rating <= 5.0 AND (rating * 10) % 5 = 0),
    content TEXT,
    contains_spoilers BOOLEAN NOT NULL DEFAULT FALSE,
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, tmdb_id)
);

-- REVIEW LIKES
CREATE TABLE review_likes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, review_id)
);

-- COMMENTS
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    contains_spoilers BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- COMMENT LIKES
CREATE TABLE comment_likes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, comment_id)
);

-- FOLLOWS
CREATE TABLE follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id),
    CHECK (follower_id != following_id)
);

-- MESSAGES
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (sender_id != receiver_id)
);

-- REPORTS
CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason report_reason NOT NULL,
    details TEXT,
    status report_status_type NOT NULL DEFAULT 'pending',
    target_user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    target_review_id UUID REFERENCES reviews(id) ON DELETE CASCADE,
    target_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolver_id UUID REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT report_target_integrity CHECK (
        (target_user_id IS NOT NULL AND target_review_id IS NULL AND target_comment_id IS NULL) OR
        (target_user_id IS NULL AND target_review_id IS NOT NULL AND target_comment_id IS NULL) OR
        (target_user_id IS NULL AND target_review_id IS NULL AND target_comment_id IS NOT NULL)
    )
);

-- NOTIFICATIONS
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type notification_type NOT NULL,
    review_id UUID REFERENCES reviews(id) ON DELETE CASCADE,
    comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    message TEXT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_data_integrity CHECK (
        (type = 'like_review'   AND actor_id IS NOT NULL AND review_id IS NOT NULL AND comment_id IS NULL AND message IS NULL) OR
        (type = 'like_comment'  AND actor_id IS NOT NULL AND comment_id IS NOT NULL AND review_id IS NULL AND message IS NULL) OR
        (type = 'new_comment'   AND actor_id IS NOT NULL AND comment_id IS NOT NULL AND review_id IS NULL AND message IS NULL) OR
        (type = 'new_follow'    AND actor_id IS NOT NULL AND review_id IS NULL     AND comment_id IS NULL AND message IS NULL) OR
        (type = 'system'        AND actor_id IS NULL     AND review_id IS NULL     AND comment_id IS NULL AND message IS NOT NULL)
    )
);

-- INDEXES
CREATE INDEX idx_reviews_tmdb_id ON reviews(tmdb_id);
CREATE INDEX idx_collections_user_id ON collections(user_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_messages_sender_receiver ON messages(sender_id, receiver_id);

