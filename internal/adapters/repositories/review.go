package repositories

import (
	"context"
	"errors"
	"fmt"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReviewRepository struct {
	db *database.DB
}

func NewReviewRepository(db *database.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	query := `
		INSERT INTO reviews (id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		review.ID, review.UserID, review.TMDBID, review.Rating, review.Content,
		review.ContainsSpoilers, review.FeaturedAt, review.CreatedAt, review.UpdatedAt,
	)
	return err
}

func (r *ReviewRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Review, error) {
	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE id = $1
	`
	review := &domain.Review{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&review.ID, &review.UserID, &review.TMDBID, &review.Rating, &review.Content,
		&review.ContainsSpoilers, &review.FeaturedAt, &review.CreatedAt, &review.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return review, err
}

func (r *ReviewRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Review, error) {
	if len(ids) == 0 {
		return []*domain.Review{}, nil
	}

	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE id = ANY($1)
	`
	rows, err := r.db.Pool.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		review := &domain.Review{}
		if err := rows.Scan(
			&review.ID, &review.UserID, &review.TMDBID, &review.Rating, &review.Content,
			&review.ContainsSpoilers, &review.FeaturedAt, &review.CreatedAt, &review.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (r *ReviewRepository) GetByUserID(ctx context.Context, userID uuid.UUID, tmdbID *int, offset, limit int, sort ports.ReviewSort) ([]*domain.Review, error) {
	orderClause := buildReviewOrderClause(sort)

	where := "WHERE r.user_id = $1"
	args := []any{userID}
	paramIdx := 2

	if tmdbID != nil {
		where += fmt.Sprintf(" AND r.tmdb_id = $%d", paramIdx)
		args = append(args, *tmdbID)
		paramIdx++
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.tmdb_id, r.rating, r.content, r.contains_spoilers, r.featured_at, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN (SELECT review_id, COUNT(*) AS like_count FROM review_likes GROUP BY review_id) rl ON rl.review_id = r.id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderClause, paramIdx, paramIdx+1)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		review := &domain.Review{}
		if err := rows.Scan(
			&review.ID, &review.UserID, &review.TMDBID, &review.Rating, &review.Content,
			&review.ContainsSpoilers, &review.FeaturedAt, &review.CreatedAt, &review.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (r *ReviewRepository) GetByTMDBID(ctx context.Context, tmdbID int, excludeUserID *uuid.UUID, offset, limit int, sort ports.ReviewSort) ([]*domain.Review, error) {
	orderClause := buildReviewOrderClause(sort)

	where := "WHERE r.tmdb_id = $1 AND r.content IS NOT NULL AND r.content != ''"
	args := []any{tmdbID}
	paramIdx := 2

	if excludeUserID != nil {
		where += fmt.Sprintf(" AND r.user_id != $%d", paramIdx)
		args = append(args, *excludeUserID)
		paramIdx++
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT r.id, r.user_id, r.tmdb_id, r.rating, r.content, r.contains_spoilers, r.featured_at, r.created_at, r.updated_at
		FROM reviews r
		LEFT JOIN (SELECT review_id, COUNT(*) AS like_count FROM review_likes GROUP BY review_id) rl ON rl.review_id = r.id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, orderClause, paramIdx, paramIdx+1)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*domain.Review
	for rows.Next() {
		review := &domain.Review{}
		if err := rows.Scan(
			&review.ID, &review.UserID, &review.TMDBID, &review.Rating, &review.Content,
			&review.ContainsSpoilers, &review.FeaturedAt, &review.CreatedAt, &review.UpdatedAt,
		); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (r *ReviewRepository) GetByUserIDAndTMDBID(ctx context.Context, userID uuid.UUID, tmdbID int) (*domain.Review, error) {
	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE user_id = $1 AND tmdb_id = $2
	`
	review := &domain.Review{}
	err := r.db.Pool.QueryRow(ctx, query, userID, tmdbID).Scan(
		&review.ID, &review.UserID, &review.TMDBID, &review.Rating, &review.Content,
		&review.ContainsSpoilers, &review.FeaturedAt, &review.CreatedAt, &review.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return review, err
}

func (r *ReviewRepository) CountByTMDBID(ctx context.Context, tmdbID int, excludeUserID *uuid.UUID) (int, error) {
	var count int
	if excludeUserID != nil {
		err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE tmdb_id = $1 AND user_id != $2 AND content IS NOT NULL AND content != ''`, tmdbID, *excludeUserID).Scan(&count)
		return count, err
	}
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE tmdb_id = $1 AND content IS NOT NULL AND content != ''`, tmdbID).Scan(&count)
	return count, err
}

func (r *ReviewRepository) CountByUserID(ctx context.Context, userID uuid.UUID, tmdbID *int) (int, error) {
	var count int
	if tmdbID != nil {
		err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE user_id = $1 AND tmdb_id = $2`, userID, *tmdbID).Scan(&count)
		return count, err
	}
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *ReviewRepository) Update(ctx context.Context, review *domain.Review) error {
	query := `
		UPDATE reviews
		SET rating = $2, content = $3, contains_spoilers = $4, featured_at = $5, updated_at = $6
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		review.ID, review.Rating, review.Content, review.ContainsSpoilers, review.FeaturedAt, review.UpdatedAt,
	)
	return err
}

func (r *ReviewRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM reviews WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}

func (r *ReviewRepository) GetAverageRatingsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]float64, error) {
	if len(tmdbIDs) == 0 {
		return make(map[int]float64), nil
	}

	query := `
		SELECT tmdb_id, AVG(rating)
		FROM reviews
		WHERE tmdb_id = ANY($1)
		GROUP BY tmdb_id
	`
	rows, err := r.db.Pool.Query(ctx, query, tmdbIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]float64)
	for rows.Next() {
		var tmdbID int
		var avgRating float64
		if err := rows.Scan(&tmdbID, &avgRating); err != nil {
			return nil, err
		}
		result[tmdbID] = avgRating
	}
	return result, rows.Err()
}

func (r *ReviewRepository) GetRatingsByUserIDAndTMDBIDs(ctx context.Context, userID uuid.UUID, tmdbIDs []int) (map[int]float64, error) {
	if len(tmdbIDs) == 0 {
		return make(map[int]float64), nil
	}

	query := `
		SELECT tmdb_id, rating
		FROM reviews
		WHERE user_id = $1 AND tmdb_id = ANY($2)
	`
	rows, err := r.db.Pool.Query(ctx, query, userID, tmdbIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]float64)
	for rows.Next() {
		var tmdbID int
		var rating float64
		if err := rows.Scan(&tmdbID, &rating); err != nil {
			return nil, err
		}
		result[tmdbID] = rating
	}
	return result, rows.Err()
}

func (r *ReviewRepository) GetRatingStatsByTMDBIDs(ctx context.Context, tmdbIDs []int) (map[int]ports.RatingStats, error) {
	if len(tmdbIDs) == 0 {
		return make(map[int]ports.RatingStats), nil
	}

	query := `
		SELECT tmdb_id, AVG(rating), COUNT(*)
		FROM reviews
		WHERE tmdb_id = ANY($1)
		GROUP BY tmdb_id
	`
	rows, err := r.db.Pool.Query(ctx, query, tmdbIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]ports.RatingStats)
	for rows.Next() {
		var tmdbID int
		var avgRating float64
		var count int
		if err := rows.Scan(&tmdbID, &avgRating, &count); err != nil {
			return nil, err
		}
		result[tmdbID] = ports.RatingStats{
			Rating: avgRating,
			Count:  count,
		}
	}
	return result, rows.Err()
}

func buildReviewOrderClause(sort ports.ReviewSort) string {
	dir := "DESC"
	if sort.Asc {
		dir = "ASC"
	}

	switch sort.Field {
	case ports.ReviewSortByLikes:
		return fmt.Sprintf("COALESCE(rl.like_count, 0) %s, r.created_at DESC", dir)
	case ports.ReviewSortByRating:
		return fmt.Sprintf("r.rating %s, r.created_at DESC", dir)
	default:
		return fmt.Sprintf("r.created_at %s", dir)
	}
}
