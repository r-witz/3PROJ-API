package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CommentLikeRepository struct {
	db *database.DB
}

func NewCommentLikeRepository(db *database.DB) *CommentLikeRepository {
	return &CommentLikeRepository{db: db}
}

func (r *CommentLikeRepository) Create(ctx context.Context, like *domain.CommentLike) error {
	query := `
		INSERT INTO comment_likes (user_id, comment_id, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Pool.Exec(ctx, query, like.UserID, like.CommentID, like.CreatedAt)
	return err
}

func (r *CommentLikeRepository) GetByCommentID(ctx context.Context, commentID uuid.UUID) ([]*domain.CommentLike, error) {
	query := `
		SELECT user_id, comment_id, created_at
		FROM comment_likes WHERE comment_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []*domain.CommentLike
	for rows.Next() {
		like := &domain.CommentLike{}
		if err := rows.Scan(&like.UserID, &like.CommentID, &like.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, like)
	}
	return likes, rows.Err()
}

func (r *CommentLikeRepository) GetByUserIDAndCommentID(ctx context.Context, userID, commentID uuid.UUID) (*domain.CommentLike, error) {
	query := `
		SELECT user_id, comment_id, created_at
		FROM comment_likes WHERE user_id = $1 AND comment_id = $2
	`
	like := &domain.CommentLike{}
	err := r.db.Pool.QueryRow(ctx, query, userID, commentID).Scan(
		&like.UserID, &like.CommentID, &like.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return like, err
}

func (r *CommentLikeRepository) Delete(ctx context.Context, userID, commentID uuid.UUID) error {
	query := `DELETE FROM comment_likes WHERE user_id = $1 AND comment_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, userID, commentID)
	return err
}

func (r *CommentLikeRepository) CountByCommentID(ctx context.Context, commentID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM comment_likes WHERE comment_id = $1`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, commentID).Scan(&count)
	return count, err
}

func (r *CommentLikeRepository) CountByCommentIDs(ctx context.Context, commentIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	query := `SELECT comment_id, COUNT(*) FROM comment_likes WHERE comment_id = ANY($1) GROUP BY comment_id`
	rows, err := r.db.Pool.Query(ctx, query, commentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		result[id] = count
	}
	return result, rows.Err()
}

func (r *CommentLikeRepository) GetLikedByUser(ctx context.Context, userID uuid.UUID, commentIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}

	query := `SELECT comment_id FROM comment_likes WHERE user_id = $1 AND comment_id = ANY($2)`
	rows, err := r.db.Pool.Query(ctx, query, userID, commentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}
