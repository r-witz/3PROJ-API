package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CommentRepository struct {
	db *database.DB
}

func NewCommentRepository(db *database.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	query := `
		INSERT INTO comments (id, user_id, review_id, content, contains_spoilers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		comment.ID, comment.UserID, comment.ReviewID, comment.Content,
		comment.ContainsSpoilers, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *CommentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	query := `
		SELECT id, user_id, review_id, content, contains_spoilers, created_at, updated_at
		FROM comments WHERE id = $1
	`
	comment := &domain.Comment{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&comment.ID, &comment.UserID, &comment.ReviewID, &comment.Content,
		&comment.ContainsSpoilers, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return comment, err
}

func (r *CommentRepository) GetByReviewID(ctx context.Context, reviewID uuid.UUID, offset, limit int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, review_id, content, contains_spoilers, created_at, updated_at
		FROM comments WHERE review_id = $1 ORDER BY created_at LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, reviewID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*domain.Comment
	for rows.Next() {
		comment := &domain.Comment{}
		if err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ReviewID, &comment.Content,
			&comment.ContainsSpoilers, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (r *CommentRepository) CountByReviewID(ctx context.Context, reviewID uuid.UUID) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM comments WHERE review_id = $1`, reviewID).Scan(&count)
	return count, err
}

func (r *CommentRepository) CountByReviewIDs(ctx context.Context, reviewIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(reviewIDs))
	if len(reviewIDs) == 0 {
		return result, nil
	}

	query := `SELECT review_id, COUNT(*) FROM comments WHERE review_id = ANY($1) GROUP BY review_id`
	rows, err := r.db.Pool.Query(ctx, query, reviewIDs)
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

func (r *CommentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	query := `
		UPDATE comments
		SET content = $2, contains_spoilers = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.db.Pool.Exec(ctx, query,
		comment.ID, comment.Content, comment.ContainsSpoilers, comment.UpdatedAt,
	)
	return err
}

func (r *CommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, query, id)
	return err
}
