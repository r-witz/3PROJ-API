package repositories

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
)

type MessageReactionRepository struct {
	db *database.DB
}

func NewMessageReactionRepository(db *database.DB) *MessageReactionRepository {
	return &MessageReactionRepository{db: db}
}

func (r *MessageReactionRepository) Create(ctx context.Context, reaction *domain.MessageReaction) error {
	query := `
		INSERT INTO message_reactions (message_id, user_id, emoji, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Pool.Exec(ctx, query, reaction.MessageID, reaction.UserID, reaction.Emoji, reaction.CreatedAt)
	return err
}

func (r *MessageReactionRepository) Delete(ctx context.Context, messageID, userID uuid.UUID, emoji string) error {
	query := `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`
	_, err := r.db.Pool.Exec(ctx, query, messageID, userID, emoji)
	return err
}

func (r *MessageReactionRepository) GetByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageReaction, error) {
	query := `
		SELECT message_id, user_id, emoji, created_at
		FROM message_reactions WHERE message_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []*domain.MessageReaction
	for rows.Next() {
		reaction := &domain.MessageReaction{}
		if err := rows.Scan(&reaction.MessageID, &reaction.UserID, &reaction.Emoji, &reaction.CreatedAt); err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}
	return reactions, rows.Err()
}

func (r *MessageReactionRepository) GetByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageReaction, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID][]*domain.MessageReaction), nil
	}

	query := `
		SELECT message_id, user_id, emoji, created_at
		FROM message_reactions WHERE message_id = ANY($1)
		ORDER BY message_id, created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*domain.MessageReaction)
	for rows.Next() {
		reaction := &domain.MessageReaction{}
		if err := rows.Scan(&reaction.MessageID, &reaction.UserID, &reaction.Emoji, &reaction.CreatedAt); err != nil {
			return nil, err
		}
		result[reaction.MessageID] = append(result[reaction.MessageID], reaction)
	}
	return result, rows.Err()
}

func (r *MessageReactionRepository) Exists(ctx context.Context, messageID, userID uuid.UUID, emoji string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3)`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, messageID, userID, emoji).Scan(&exists)
	return exists, err
}
