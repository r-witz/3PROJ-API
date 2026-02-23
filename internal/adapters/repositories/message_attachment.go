package repositories

import (
	"context"
	"fmt"
	"strings"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
)

type MessageAttachmentRepository struct {
	db *database.DB
}

func NewMessageAttachmentRepository(db *database.DB) *MessageAttachmentRepository {
	return &MessageAttachmentRepository{db: db}
}

func (r *MessageAttachmentRepository) CreateBatch(ctx context.Context, attachments []*domain.MessageAttachment) error {
	if len(attachments) == 0 {
		return nil
	}

	values := make([]string, 0, len(attachments))
	args := make([]interface{}, 0, len(attachments)*7)
	for i, a := range attachments {
		base := i * 7
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)", base+1, base+2, base+3, base+4, base+5, base+6, base+7))
		args = append(args, a.ID, a.MessageID, a.FileURL, a.FileName, a.FileSize, a.ContentType, a.Position)
	}

	query := fmt.Sprintf(`
		INSERT INTO message_attachments (id, message_id, file_url, file_name, file_size, content_type, position)
		VALUES %s
	`, strings.Join(values, ", "))

	_, err := r.db.Pool.Exec(ctx, query, args...)
	return err
}

func (r *MessageAttachmentRepository) GetByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageAttachment, error) {
	query := `
		SELECT id, message_id, file_url, file_name, file_size, content_type, position, created_at
		FROM message_attachments WHERE message_id = $1
		ORDER BY position
	`
	rows, err := r.db.Pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*domain.MessageAttachment
	for rows.Next() {
		a := &domain.MessageAttachment{}
		if err := rows.Scan(&a.ID, &a.MessageID, &a.FileURL, &a.FileName, &a.FileSize, &a.ContentType, &a.Position, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

func (r *MessageAttachmentRepository) GetByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]*domain.MessageAttachment, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID][]*domain.MessageAttachment), nil
	}

	query := `
		SELECT id, message_id, file_url, file_name, file_size, content_type, position, created_at
		FROM message_attachments WHERE message_id = ANY($1)
		ORDER BY message_id, position
	`
	rows, err := r.db.Pool.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*domain.MessageAttachment)
	for rows.Next() {
		a := &domain.MessageAttachment{}
		if err := rows.Scan(&a.ID, &a.MessageID, &a.FileURL, &a.FileName, &a.FileSize, &a.ContentType, &a.Position, &a.CreatedAt); err != nil {
			return nil, err
		}
		result[a.MessageID] = append(result[a.MessageID], a)
	}
	return result, rows.Err()
}

func (r *MessageAttachmentRepository) DeleteByMessageID(ctx context.Context, messageID uuid.UUID) ([]*domain.MessageAttachment, error) {
	attachments, err := r.GetByMessageID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	query := `DELETE FROM message_attachments WHERE message_id = $1`
	_, err = r.db.Pool.Exec(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	return attachments, nil
}
