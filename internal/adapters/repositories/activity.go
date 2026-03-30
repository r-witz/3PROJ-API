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

func (r *ActivityRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Activity, error) {
	query := `
		SELECT id, user_id, type, review_id, collection_id, comment_id, tmdb_id, created_at
		FROM activities WHERE user_id = $1 ORDER BY created_at DESC
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

func (r *ActivityRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM activities WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *ActivityRepository) CountFeedForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM activities a
		INNER JOIN follows f ON a.user_id = f.following_id
		WHERE f.follower_id = $1
	`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *ActivityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM activities WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *ActivityRepository) DeleteByTypeAndReference(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error {
	query := `
		DELETE FROM activities
		WHERE user_id = $1 AND type = $2
		  AND (review_id = $3 OR ($3::uuid IS NULL AND review_id IS NULL))
		  AND (collection_id = $4 OR ($4::uuid IS NULL AND collection_id IS NULL))
		  AND (comment_id = $5 OR ($5::uuid IS NULL AND comment_id IS NULL))
		  AND (tmdb_id = $6 OR ($6::int IS NULL AND tmdb_id IS NULL))
	`
	_, err := r.db.Pool.Exec(ctx, query, userID, actType, reviewID, collectionID, commentID, tmdbID)
	return err
}
