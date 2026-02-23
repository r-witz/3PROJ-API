package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ConversationStateRepository struct {
	db *database.DB
}

func NewConversationStateRepository(db *database.DB) *ConversationStateRepository {
	return &ConversationStateRepository{db: db}
}

func (r *ConversationStateRepository) Upsert(ctx context.Context, state *domain.ConversationState) error {
	query := `
		INSERT INTO conversation_states (user_id, other_user_id, closed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, other_user_id) DO UPDATE SET closed_at = $3, updated_at = $5
	`
	_, err := r.db.Pool.Exec(ctx, query, state.UserID, state.OtherUserID, state.ClosedAt, state.CreatedAt, state.UpdatedAt)
	return err
}

func (r *ConversationStateRepository) GetByUserAndOther(ctx context.Context, userID, otherUserID uuid.UUID) (*domain.ConversationState, error) {
	query := `
		SELECT user_id, other_user_id, closed_at, created_at, updated_at
		FROM conversation_states WHERE user_id = $1 AND other_user_id = $2
	`
	state := &domain.ConversationState{}
	err := r.db.Pool.QueryRow(ctx, query, userID, otherUserID).Scan(
		&state.UserID, &state.OtherUserID, &state.ClosedAt, &state.CreatedAt, &state.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return state, err
}

func (r *ConversationStateRepository) GetClosedConversationPartnerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT other_user_id FROM conversation_states
		WHERE user_id = $1 AND closed_at IS NOT NULL
	`
	rows, err := r.db.Pool.Query(ctx, query, userID)
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

func (r *ConversationStateRepository) ClearClosedAt(ctx context.Context, userID, otherUserID uuid.UUID) error {
	query := `
		UPDATE conversation_states SET closed_at = NULL, updated_at = NOW()
		WHERE user_id = $1 AND other_user_id = $2
	`
	_, err := r.db.Pool.Exec(ctx, query, userID, otherUserID)
	return err
}
