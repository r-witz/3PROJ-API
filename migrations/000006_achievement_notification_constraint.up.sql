ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notification_data_integrity;

ALTER TABLE notifications ADD CONSTRAINT notification_data_integrity CHECK (
    (actor_id IS NOT NULL) = (type IN ('like_review', 'like_comment', 'new_comment', 'new_follow'))
    AND (review_id IS NOT NULL) = (type = 'like_review')
    AND (comment_id IS NOT NULL) = (type IN ('like_comment', 'new_comment'))
    AND (achievement_id IS NOT NULL) = (type = 'achievement_unlocked')
    AND (message IS NOT NULL) = (type IN ('system', 'achievement_unlocked'))
);
