CREATE TABLE user_blocks (
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id != blocked_id)
);
CREATE INDEX idx_user_blocks_blocked_id ON user_blocks (blocked_id);

ALTER TABLE messages ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE messages ALTER COLUMN content DROP NOT NULL;

CREATE TABLE message_attachments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size INTEGER NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    position SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_message_attachments_message_id ON message_attachments (message_id);

CREATE TABLE message_reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE TABLE conversation_states (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    other_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    closed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, other_user_id),
    CHECK (user_id != other_user_id)
);
