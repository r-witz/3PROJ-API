package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
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

func (r *ReviewRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Review, error) {
	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE user_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
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

func (r *ReviewRepository) GetByTMDBID(ctx context.Context, tmdbID int) ([]*domain.Review, error) {
	query := `
		SELECT id, user_id, tmdb_id, rating, content, contains_spoilers, featured_at, created_at, updated_at
		FROM reviews WHERE tmdb_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, tmdbID)
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
