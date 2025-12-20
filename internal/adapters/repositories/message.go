package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
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
