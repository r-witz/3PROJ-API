package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type MessageRepository struct {
	db *database.DB
}

func NewMessageRepository(db *database.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (id, sender_id, receiver_id, content, read_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		message.ID, message.SenderID, message.ReceiverID, message.Content, message.ReadAt, message.CreatedAt,
	)
	return err
}

func (r *MessageRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
	query := `
		SELECT id, sender_id, receiver_id, content, read_at, created_at
		FROM messages WHERE id = $1
	`
	message := &domain.Message{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&message.ID, &message.SenderID, &message.ReceiverID, &message.Content, &message.ReadAt, &message.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return message, err
}

func (r *MessageRepository) GetConversation(ctx context.Context, userID1, userID2 uuid.UUID) ([]*domain.Message, error) {
	query := `
		SELECT id, sender_id, receiver_id, content, read_at, created_at
		FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
		ORDER BY created_at
	`
	rows, err := r.db.Pool.Query(ctx, query, userID1, userID2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		message := &domain.Message{}
		if err := rows.Scan(
			&message.ID, &message.SenderID, &message.ReceiverID, &message.Content, &message.ReadAt, &message.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (r *MessageRepository) GetConversationPaginated(ctx context.Context, userID1, userID2 uuid.UUID, offset, limit int) ([]*domain.Message, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
	`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, userID1, userID2).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, sender_id, receiver_id, content, read_at, created_at
		FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
		ORDER BY created_at DESC LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Pool.Query(ctx, query, userID1, userID2, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		message := &domain.Message{}
		if err := rows.Scan(
			&message.ID, &message.SenderID, &message.ReceiverID, &message.Content, &message.ReadAt, &message.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		messages = append(messages, message)
	}
	return messages, total, rows.Err()
}

func (r *MessageRepository) GetConversations(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ports.ConversationPreview, int, error) {
	countQuery := `
		SELECT COUNT(DISTINCT CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END)
		FROM messages
		WHERE sender_id = $1 OR receiver_id = $1
	`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		WITH conversation_partners AS (
			SELECT DISTINCT CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END AS other_user_id
			FROM messages
			WHERE sender_id = $1 OR receiver_id = $1
		),
		latest_messages AS (
			SELECT DISTINCT ON (cp.other_user_id)
				cp.other_user_id,
				m.id, m.sender_id, m.receiver_id, m.content, m.read_at, m.created_at
			FROM conversation_partners cp
			JOIN messages m ON
				(m.sender_id = $1 AND m.receiver_id = cp.other_user_id) OR
				(m.sender_id = cp.other_user_id AND m.receiver_id = $1)
			ORDER BY cp.other_user_id, m.created_at DESC
		),
		unread_counts AS (
			SELECT sender_id AS other_user_id, COUNT(*) AS unread_count
			FROM messages
			WHERE receiver_id = $1 AND read_at IS NULL
			GROUP BY sender_id
		)
		SELECT lm.other_user_id, lm.id, lm.sender_id, lm.receiver_id, lm.content, lm.read_at, lm.created_at,
			COALESCE(uc.unread_count, 0) AS unread_count
		FROM latest_messages lm
		LEFT JOIN unread_counts uc ON uc.other_user_id = lm.other_user_id
		ORDER BY lm.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var conversations []*ports.ConversationPreview
	for rows.Next() {
		msg := &domain.Message{}
		conv := &ports.ConversationPreview{LastMessage: msg}
		if err := rows.Scan(
			&conv.OtherUserID,
			&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.ReadAt, &msg.CreatedAt,
			&conv.UnreadCount,
		); err != nil {
			return nil, 0, err
		}
		conversations = append(conversations, conv)
	}
	return conversations, total, rows.Err()
}

func (r *MessageRepository) MarkConversationAsRead(ctx context.Context, userID, otherUserID uuid.UUID) error {
	query := `
		UPDATE messages SET read_at = NOW()
		WHERE sender_id = $2 AND receiver_id = $1 AND read_at IS NULL
	`
	_, err := r.db.Pool.Exec(ctx, query, userID, otherUserID)
	return err
}

func (r *MessageRepository) Update(ctx context.Context, message *domain.Message) error {
	query := `
		UPDATE messages
		SET content = $2, read_at = $3
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, message.ID, message.Content, message.ReadAt)
	return err
}

func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM messages WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
