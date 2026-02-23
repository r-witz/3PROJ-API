DROP TABLE IF EXISTS conversation_states;
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS message_attachments;
UPDATE messages SET content = '' WHERE content IS NULL;
ALTER TABLE messages ALTER COLUMN content SET NOT NULL;
ALTER TABLE messages DROP COLUMN IF EXISTS updated_at;
DROP TABLE IF EXISTS user_blocks;
