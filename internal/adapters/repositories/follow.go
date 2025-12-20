package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FollowRepository struct {
	db *database.DB
}

func NewFollowRepository(db *database.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Create(ctx context.Context, follow *domain.Follow) error {
	query := `
		INSERT INTO follows (follower_id, following_id, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Pool.Exec(ctx, query, follow.FollowerID, follow.FollowingID, follow.CreatedAt)
	return err
}

func (r *FollowRepository) GetFollowers(ctx context.Context, userID uuid.UUID) ([]*domain.Follow, error) {
	query := `
		SELECT follower_id, following_id, created_at
		FROM follows WHERE following_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var follows []*domain.Follow
	for rows.Next() {
		follow := &domain.Follow{}
		if err := rows.Scan(&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt); err != nil {
			return nil, err
		}
		follows = append(follows, follow)
	}
	return follows, rows.Err()
}

func (r *FollowRepository) GetFollowing(ctx context.Context, userID uuid.UUID) ([]*domain.Follow, error) {
	query := `
		SELECT follower_id, following_id, created_at
		FROM follows WHERE follower_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var follows []*domain.Follow
	for rows.Next() {
		follow := &domain.Follow{}
		if err := rows.Scan(&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt); err != nil {
			return nil, err
		}
		follows = append(follows, follow)
	}
	return follows, rows.Err()
}

func (r *FollowRepository) GetByFollowerIDAndFollowingID(ctx context.Context, followerID, followingID uuid.UUID) (*domain.Follow, error) {
	query := `
		SELECT follower_id, following_id, created_at
		FROM follows WHERE follower_id = $1 AND following_id = $2
	`
	follow := &domain.Follow{}
	err := r.db.Pool.QueryRow(ctx, query, followerID, followingID).Scan(
		&follow.FollowerID, &follow.FollowingID, &follow.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return follow, err
}

func (r *FollowRepository) Delete(ctx context.Context, followerID, followingID uuid.UUID) error {
	query := `DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, followerID, followingID)
	return err
}

func (r *FollowRepository) CountFollowers(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE following_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (r *FollowRepository) CountFollowing(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE follower_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
