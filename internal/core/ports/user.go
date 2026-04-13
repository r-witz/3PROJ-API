package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type UserSearchParams struct {
	Query        string
	Limit        int
	Offset       int
	SortField    string
	SortOrder    string
	ExcludeRoles []domain.UserRole
	HideBanned   bool
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	SearchByUsername(ctx context.Context, params UserSearchParams) ([]*domain.User, int, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByRole(ctx context.Context, role domain.UserRole) (bool, error)
	GetBannedUserIDs(ctx context.Context) ([]uuid.UUID, error)
}
