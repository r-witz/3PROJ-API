package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type NotificationPreferencesRepository struct {
	db *database.DB
}

func NewNotificationPreferencesRepository(db *database.DB) *NotificationPreferencesRepository {
	return &NotificationPreferencesRepository{db: db}
}

func (r *NotificationPreferencesRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.NotificationPreferences, error) {
	query := `
		SELECT user_id, like_review, like_comment, new_comment, new_follow, system, updated_at
		FROM notification_preferences WHERE user_id = $1
	`
	prefs := &domain.NotificationPreferences{}
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&prefs.UserID, &prefs.LikeReview, &prefs.LikeComment,
		&prefs.NewComment, &prefs.NewFollow, &prefs.System, &prefs.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return prefs, err
}

func (r *NotificationPreferencesRepository) Upsert(ctx context.Context, prefs *domain.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (user_id, like_review, like_comment, new_comment, new_follow, system, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO UPDATE SET
			like_review = EXCLUDED.like_review,
			like_comment = EXCLUDED.like_comment,
			new_comment = EXCLUDED.new_comment,
			new_follow = EXCLUDED.new_follow,
			system = EXCLUDED.system,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.Pool.Exec(ctx, query,
		prefs.UserID, prefs.LikeReview, prefs.LikeComment,
		prefs.NewComment, prefs.NewFollow, prefs.System, prefs.UpdatedAt,
	)
	return err
}
