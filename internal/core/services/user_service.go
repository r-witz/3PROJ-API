package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	portservices "duskforge-api/internal/core/ports/services"

	"github.com/google/uuid"
)

type userService struct {
	userRepo ports.UserRepository
}

func NewUserService(userRepo ports.UserRepository) portservices.UserService {
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

func (s *userService) UpdateCurrentUser(ctx context.Context, userID uuid.UUID, input portservices.UpdateUserInput) (*domain.User, error) {
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

	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
	}
	if input.Bio != nil {
		user.Bio = input.Bio
	}
	if input.Website != nil {
		user.Website = input.Website
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
