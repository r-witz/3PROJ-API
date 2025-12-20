package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ActivityRepository struct {
	db *database.DB
}

func NewActivityRepository(db *database.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

func (r *ActivityRepository) Create(ctx context.Context, activity *domain.Activity) error {
	query := `
		INSERT INTO activities (id, user_id, type, review_id, collection_id, comment_id, tmdb_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		activity.ID, activity.UserID, activity.Type, activity.ReviewID,
		activity.CollectionID, activity.CommentID, activity.TMDBID, activity.CreatedAt,
	)
	return err
}

func (r *ActivityRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	query := `
		SELECT id, user_id, type, review_id, collection_id, comment_id, tmdb_id, created_at
		FROM activities WHERE id = $1
	`
	activity := &domain.Activity{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&activity.ID, &activity.UserID, &activity.Type, &activity.ReviewID,
		&activity.CollectionID, &activity.CommentID, &activity.TMDBID, &activity.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return activity, err
}

func (r *ActivityRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Activity, error) {
	query := `
		SELECT id, user_id, type, review_id, collection_id, comment_id, tmdb_id, created_at
		FROM activities WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*domain.Activity
	for rows.Next() {
		activity := &domain.Activity{}
		if err := rows.Scan(
			&activity.ID, &activity.UserID, &activity.Type, &activity.ReviewID,
			&activity.CollectionID, &activity.CommentID, &activity.TMDBID, &activity.CreatedAt,
		); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	return activities, rows.Err()
}

func (r *ActivityRepository) GetFeedForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Activity, error) {
	query := `
		SELECT a.id, a.user_id, a.type, a.review_id, a.collection_id, a.comment_id, a.tmdb_id, a.created_at
		FROM activities a
		INNER JOIN follows f ON a.user_id = f.following_id
		WHERE f.follower_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*domain.Activity
	for rows.Next() {
		activity := &domain.Activity{}
		if err := rows.Scan(
			&activity.ID, &activity.UserID, &activity.Type, &activity.ReviewID,
			&activity.CollectionID, &activity.CommentID, &activity.TMDBID, &activity.CreatedAt,
		); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	return activities, rows.Err()
}

func (r *ActivityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM activities WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
