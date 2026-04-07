package ports

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type SeedSuperAdminInput struct {
	Email    string
	Username string
	Password string
}

type AdminService interface {
	BanUser(ctx context.Context, adminID uuid.UUID, targetUserID uuid.UUID) error
	UnbanUser(ctx context.Context, adminID uuid.UUID, targetUserID uuid.UUID) error
	DeleteReview(ctx context.Context, reviewID uuid.UUID) error
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
	SetUserRole(ctx context.Context, superAdminID uuid.UUID, targetUserID uuid.UUID, newRole domain.UserRole) error
	GetUsers(ctx context.Context, offset, limit int, bannedOnly bool) ([]*domain.User, int, error)
	SeedSuperAdmin(ctx context.Context, input SeedSuperAdminInput) error
}
