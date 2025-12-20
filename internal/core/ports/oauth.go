package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type OAuthAccountRepository interface {
	Create(ctx context.Context, account *domain.OAuthAccount) error
	GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.OAuthAccount, error)
	GetByUserIDAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*domain.OAuthAccount, error)
	Delete(ctx context.Context, provider, providerUserID string) error
}
