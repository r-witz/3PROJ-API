package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/auth"

	"github.com/google/uuid"
)

type userService struct {
	userRepo ports.UserRepository
}

func NewUserService(userRepo ports.UserRepository) ports.UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *userService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.GetByID(ctx, userID)
}

func (s *userService) UpdateCurrentUser(ctx context.Context, userID uuid.UUID, input ports.UpdateUserInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	if input.Email != nil {
		existing, err := s.userRepo.GetByEmail(ctx, *input.Email)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if existing != nil && existing.ID != userID {
			return nil, domain.ErrEmailAlreadyExists
		}
		user.Email = *input.Email
	}

	if input.Username != nil {
		existing, err := s.userRepo.GetByUsername(ctx, *input.Username)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if existing != nil && existing.ID != userID {
			return nil, domain.ErrUsernameAlreadyExists
		}
		user.Username = *input.Username
	}

	if input.Bio != nil {
		user.Bio = input.Bio
	}
	if input.Website != nil {
		if *input.Website == "" {
			user.Website = nil
		} else {
			user.Website = input.Website
		}
	}
	if input.Theme != nil {
		user.Theme = *input.Theme
	}
	if input.Locale != nil {
		user.Locale = *input.Locale
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, domain.ErrInternal
	}

	return user, nil
}

func (s *userService) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	user.AvatarURL = &avatarURL
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, domain.ErrInternal
	}

	return user, nil
}

func (s *userService) DeleteAvatar(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	user.AvatarURL = nil
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, domain.ErrInternal
	}

	return user, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, input ports.ChangePasswordInput) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return domain.ErrUserNotFound
	}

	if user.PasswordHash != nil {
		match, err := auth.ComparePassword(*user.PasswordHash, input.CurrentPassword)
		if err != nil {
			return domain.ErrInternal
		}
		if !match {
			return domain.ErrIncorrectPassword
		}
	}

	newHash, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		return mapPasswordError(err)
	}

	user.PasswordHash = &newHash
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *userService) DeleteCurrentUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return domain.ErrInternal
	}
	if user == nil {
		return domain.ErrUserNotFound
	}

	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *userService) SearchUsers(ctx context.Context, input ports.SearchUsersInput) (*ports.SearchUsersResult, error) {
	if input.Offset < 0 {
		input.Offset = 0
	}
	if input.Limit < 1 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	searchParams := ports.UserSearchParams{
		Query:     input.Query,
		Limit:     input.Limit,
		Offset:    input.Offset,
		SortField: input.SortField,
		SortOrder: input.SortOrder,
	}

	isAdmin := input.CallerRole == string(domain.UserRoleAdmin) || input.CallerRole == string(domain.UserRoleSuperAdmin)
	if !isAdmin {
		searchParams.ExcludeRoles = []domain.UserRole{domain.UserRoleAdmin, domain.UserRoleSuperAdmin}
		searchParams.HideBanned = true
	}

	users, total, err := s.userRepo.SearchByUsername(ctx, searchParams)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return &ports.SearchUsersResult{
		Users:  users,
		Total:  total,
		Offset: input.Offset,
		Limit:  len(users),
	}, nil
}
