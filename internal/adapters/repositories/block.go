package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BlockRepository struct {
	db *database.DB
}

func NewBlockRepository(db *database.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

func (r *BlockRepository) Create(ctx context.Context, block *domain.UserBlock) error {
	query := `
		INSERT INTO user_blocks (blocker_id, blocked_id, created_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Pool.Exec(ctx, query, block.BlockerID, block.BlockedID, block.CreatedAt)
	return err
}

func (r *BlockRepository) Delete(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	query := `DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, blockerID, blockedID)
	return err
}

func (r *BlockRepository) GetByBlockerAndBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (*domain.UserBlock, error) {
	query := `
		SELECT blocker_id, blocked_id, created_at
		FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2
	`
	block := &domain.UserBlock{}
	err := r.db.Pool.QueryRow(ctx, query, blockerID, blockedID).Scan(
		&block.BlockerID, &block.BlockedID, &block.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return block, err
}

func (r *BlockRepository) GetBlockedByUser(ctx context.Context, blockerID uuid.UUID, offset, limit int) ([]*domain.UserBlock, int, error) {
	countQuery := `SELECT COUNT(*) FROM user_blocks WHERE blocker_id = $1`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, blockerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT blocker_id, blocked_id, created_at
		FROM user_blocks WHERE blocker_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, blockerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blocks []*domain.UserBlock
	for rows.Next() {
		block := &domain.UserBlock{}
		if err := rows.Scan(&block.BlockerID, &block.BlockedID, &block.CreatedAt); err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, block)
	}
	return blocks, total, rows.Err()
}

func (r *BlockRepository) IsBlocked(ctx context.Context, userID1, userID2 uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)
	`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, userID1, userID2).Scan(&exists)
	return exists, err
}

func (r *BlockRepository) IsBlockedBy(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2)`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, blockerID, blockedID).Scan(&exists)
	return exists, err
}

func (r *BlockRepository) GetBlockerIDs(ctx context.Context, blockedID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT blocker_id FROM user_blocks WHERE blocked_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, blockedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *BlockRepository) GetBlockedIDs(ctx context.Context, blockerID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT blocked_id FROM user_blocks WHERE blocker_id = $1`
	rows, err := r.db.Pool.Query(ctx, query, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
