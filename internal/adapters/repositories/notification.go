package repositories

import (
	"context"
	"errors"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationRepository struct {
	db *database.DB
}

func NewNotificationRepository(db *database.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		notification.ID, notification.UserID, notification.ActorID, notification.Type,
		notification.ReviewID, notification.CommentID, notification.Message,
		notification.ReadAt, notification.CreatedAt,
	)
	return err
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at
		FROM notifications WHERE id = $1
	`
	notification := &domain.Notification{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&notification.ID, &notification.UserID, &notification.ActorID, &notification.Type,
		&notification.ReviewID, &notification.CommentID, &notification.Message,
		&notification.ReadAt, &notification.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return notification, err
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at
		FROM notifications WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		notification := &domain.Notification{}
		if err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.ActorID, &notification.Type,
			&notification.ReviewID, &notification.CommentID, &notification.Message,
			&notification.ReadAt, &notification.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at
		FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		notification := &domain.Notification{}
		if err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.ActorID, &notification.Type,
			&notification.ReviewID, &notification.CommentID, &notification.Message,
			&notification.ReadAt, &notification.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) GetUnreadByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Notification, error) {
	query := `
		SELECT id, user_id, actor_id, type, review_id, comment_id, message, read_at, created_at
		FROM notifications WHERE user_id = $1 AND read_at IS NULL ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		notification := &domain.Notification{}
		if err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.ActorID, &notification.Type,
			&notification.ReviewID, &notification.CommentID, &notification.Message,
			&notification.ReadAt, &notification.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepository) Update(ctx context.Context, notification *domain.Notification) error {
	query := `
		UPDATE notifications
		SET read_at = $2
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query, notification.ID, notification.ReadAt)
	return err
}

func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications SET read_at = $2 WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id, time.Now())
	return err
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = $2 WHERE user_id = $1 AND read_at IS NULL`
	_, err := r.db.Pool.Exec(ctx, query, userID, time.Now())
	return err
}
