package repositories

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OAuthAccountRepository struct {
	db *database.DB
}

func NewOAuthAccountRepository(db *database.DB) *OAuthAccountRepository {
	return &OAuthAccountRepository{db: db}
}

func (r *OAuthAccountRepository) Create(ctx context.Context, account *domain.OAuthAccount) error {
	query := `
		INSERT INTO oauth_accounts (provider, provider_user_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Pool.Exec(ctx, query,
		account.Provider, account.ProviderUserID, account.UserID, account.CreatedAt,
	)
	return err
}

func (r *OAuthAccountRepository) GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error) {
	query := `
		SELECT provider, provider_user_id, user_id, created_at
		FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2
	`
	account := &domain.OAuthAccount{}
	err := r.db.Pool.QueryRow(ctx, query, provider, providerUserID).Scan(
		&account.Provider, &account.ProviderUserID, &account.UserID, &account.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return account, err
}

func (r *OAuthAccountRepository) GetByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.OAuthAccount, error) {
	query := `
		SELECT provider, provider_user_id, user_id, created_at
		FROM oauth_accounts WHERE user_id = $1 AND provider = $2
	`
	account := &domain.OAuthAccount{}
	err := r.db.Pool.QueryRow(ctx, query, userID, provider).Scan(
		&account.Provider, &account.ProviderUserID, &account.UserID, &account.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return account, err
}

func (r *OAuthAccountRepository) Delete(ctx context.Context, provider, providerUserID string) error {
	query := `DELETE FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`
	_, err := r.db.Pool.Exec(ctx, query, provider, providerUserID)
	return err
}
