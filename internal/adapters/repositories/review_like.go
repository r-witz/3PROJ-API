package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ReviewLikeRepository struct {
	db *database.DB
}

func NewReviewLikeRepository(db *database.DB) *ReviewLikeRepository {
	return &ReviewLikeRepository{db: db}
}

func (r *ReviewLikeRepository) Create(ctx context.Context, like *domain.ReviewLike) error {
	query := `
		INSERT INTO review_likes (user_id, review_id, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Pool.Exec(ctx, query, like.UserID, like.ReviewID, like.CreatedAt)
	return err
}

func (r *ReviewLikeRepository) GetByReviewID(ctx context.Context, reviewID uuid.UUID) ([]*domain.ReviewLike, error) {
	query := `
		SELECT user_id, review_id, created_at
		FROM review_likes WHERE review_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, reviewID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []*domain.ReviewLike
	for rows.Next() {
		like := &domain.ReviewLike{}
		if err := rows.Scan(&like.UserID, &like.ReviewID, &like.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, like)
	}
	return likes, rows.Err()
}

func (r *ReviewLikeRepository) GetByUserIDAndReviewID(ctx context.Context, userID, reviewID uuid.UUID) (*domain.ReviewLike, error) {
	query := `
		SELECT user_id, review_id, created_at
		FROM review_likes WHERE user_id = $1 AND review_id = $2
	`
	like := &domain.ReviewLike{}
	err := r.db.Pool.QueryRow(ctx, query, userID, reviewID).Scan(
		&like.UserID, &like.ReviewID, &like.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return like, err
}

func (r *ReviewLikeRepository) Delete(ctx context.Context, userID, reviewID uuid.UUID) error {
	query := `DELETE FROM review_likes WHERE user_id = $1 AND review_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, userID, reviewID)
	return err
}

func (r *ReviewLikeRepository) CountByReviewID(ctx context.Context, reviewID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM review_likes WHERE review_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, reviewID).Scan(&count)
	return count, err
}
