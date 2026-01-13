package services

import (
	"context"

	"duskforge-api/internal/core/domain"

	"github.com/google/uuid"
)

type UpdateUserInput struct {
	Email     *string
	Username  *string
	AvatarURL *string
	Bio       *string
	Website   *string
	Theme     *domain.UserTheme
	Locale    *domain.UserLocale
}

type SearchUsersInput struct {
	Query     string
	Page      int
	PerPage   int
	SortField string
	SortOrder string
}

type SearchUsersResult struct {
	Users      []*domain.User
	Total      int
	Page       int
	PerPage    int
	TotalPages int
}

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateCurrentUser(ctx context.Context, userID uuid.UUID, input UpdateUserInput) (*domain.User, error)
	DeleteCurrentUser(ctx context.Context, userID uuid.UUID) error
	SearchUsers(ctx context.Context, input SearchUsersInput) (*SearchUsersResult, error)
}
