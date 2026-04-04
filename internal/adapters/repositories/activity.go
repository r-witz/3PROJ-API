package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

const activityColumns = `id, user_id, type, review_id, collection_id, comment_id, tmdb_id, target_user_id, created_at`

func scanActivity(row pgx.Row) (*domain.Activity, error) {
	activity := &domain.Activity{}
	err := row.Scan(
		&activity.ID, &activity.UserID, &activity.Type, &activity.ReviewID,
		&activity.CollectionID, &activity.CommentID, &activity.TMDBID,
		&activity.TargetUserID, &activity.CreatedAt,
	)
	return activity, err
}

func scanActivities(rows pgx.Rows) ([]*domain.Activity, error) {
	var activities []*domain.Activity
	for rows.Next() {
		activity := &domain.Activity{}
		if err := rows.Scan(
			&activity.ID, &activity.UserID, &activity.Type, &activity.ReviewID,
			&activity.CollectionID, &activity.CommentID, &activity.TMDBID,
			&activity.TargetUserID, &activity.CreatedAt,
		); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	return activities, rows.Err()
}

func buildTypeFilter(types []domain.ActivityType, paramOffset int) (string, []interface{}) {
	if len(types) == 0 {
		return "", nil
	}
	placeholders := make([]string, len(types))
	args := make([]interface{}, len(types))
	for i, t := range types {
		placeholders[i] = fmt.Sprintf("$%d", paramOffset+i)
		args[i] = t
	}
	return " AND a.type IN (" + strings.Join(placeholders, ", ") + ")", args
}

func (r *ActivityRepository) Create(ctx context.Context, activity *domain.Activity) error {
	query := `
		INSERT INTO activities (id, user_id, type, review_id, collection_id, comment_id, tmdb_id, target_user_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		activity.ID, activity.UserID, activity.Type, activity.ReviewID,
		activity.CollectionID, activity.CommentID, activity.TMDBID,
		activity.TargetUserID, activity.CreatedAt,
	)
	return err
}

func (r *ActivityRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Activity, error) {
	query := `SELECT ` + activityColumns + ` FROM activities WHERE id = $1`
	activity, err := scanActivity(r.db.Pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return activity, err
}

func (r *ActivityRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Activity, error) {
	query := `SELECT ` + activityColumns + ` FROM activities WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *ActivityRepository) GetFeedForUser(ctx context.Context, userID uuid.UUID, limit, offset int, types []domain.ActivityType) ([]*domain.Activity, error) {
	baseQuery := `
		SELECT a.` + strings.ReplaceAll(activityColumns, ", ", ", a.") + `
		FROM activities a
		INNER JOIN follows f ON a.user_id = f.following_id
		WHERE f.follower_id = $1
	`
	args := []interface{}{userID}
	typeFilter, typeArgs := buildTypeFilter(types, 2)
	baseQuery += typeFilter
	args = append(args, typeArgs...)

	nextParam := 2 + len(typeArgs)
	baseQuery += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", nextParam, nextParam+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *ActivityRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, limit, offset int, types []domain.ActivityType) ([]*domain.Activity, error) {
	baseQuery := `SELECT a.` + strings.ReplaceAll(activityColumns, ", ", ", a.") + ` FROM activities a WHERE a.user_id = $1`
	args := []interface{}{userID}
	typeFilter, typeArgs := buildTypeFilter(types, 2)
	baseQuery += typeFilter
	args = append(args, typeArgs...)

	nextParam := 2 + len(typeArgs)
	baseQuery += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d", nextParam, nextParam+1)
	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *ActivityRepository) CountByUserID(ctx context.Context, userID uuid.UUID, types []domain.ActivityType) (int, error) {
	baseQuery := `SELECT COUNT(*) FROM activities a WHERE a.user_id = $1`
	args := []interface{}{userID}
	typeFilter, typeArgs := buildTypeFilter(types, 2)
	baseQuery += typeFilter
	args = append(args, typeArgs...)

	var count int
	err := r.db.Pool.QueryRow(ctx, baseQuery, args...).Scan(&count)
	return count, err
}

func (r *ActivityRepository) CountFeedForUser(ctx context.Context, userID uuid.UUID, types []domain.ActivityType) (int, error) {
	baseQuery := `
		SELECT COUNT(*)
		FROM activities a
		INNER JOIN follows f ON a.user_id = f.following_id
		WHERE f.follower_id = $1
	`
	args := []interface{}{userID}
	typeFilter, typeArgs := buildTypeFilter(types, 2)
	baseQuery += typeFilter
	args = append(args, typeArgs...)

	var count int
	err := r.db.Pool.QueryRow(ctx, baseQuery, args...).Scan(&count)
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

func (r *ActivityRepository) DeleteByFields(ctx context.Context, userID uuid.UUID, actType domain.ActivityType, reviewID *uuid.UUID, collectionID *uuid.UUID, commentID *uuid.UUID, tmdbID *int) error {
	query := `DELETE FROM activities WHERE user_id = $1 AND type = $2`
	args := []interface{}{userID, actType}
	paramIdx := 3

	if reviewID != nil {
		query += fmt.Sprintf(" AND review_id = $%d", paramIdx)
		args = append(args, *reviewID)
		paramIdx++
	}
	if collectionID != nil {
		query += fmt.Sprintf(" AND collection_id = $%d", paramIdx)
		args = append(args, *collectionID)
		paramIdx++
	}
	if commentID != nil {
		query += fmt.Sprintf(" AND comment_id = $%d", paramIdx)
		args = append(args, *commentID)
		paramIdx++
	}
	if tmdbID != nil {
		query += fmt.Sprintf(" AND tmdb_id = $%d", paramIdx)
		args = append(args, *tmdbID)
		paramIdx++
	}

	_, err := r.db.Pool.Exec(ctx, query, args...)
	return err
}
