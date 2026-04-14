package services

import (
	"context"
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/auth"

	"github.com/google/uuid"
)

type AdminService struct {
	userRepo    ports.UserRepository
	reviewRepo  ports.ReviewRepository
	commentRepo ports.CommentRepository
	sessionRepo ports.SessionRepository
	banCache    ports.BanCache
}

func NewAdminService(
	userRepo ports.UserRepository,
	reviewRepo ports.ReviewRepository,
	commentRepo ports.CommentRepository,
	sessionRepo ports.SessionRepository,
	banCache ports.BanCache,
) *AdminService {
	return &AdminService{
		userRepo:    userRepo,
		reviewRepo:  reviewRepo,
		commentRepo: commentRepo,
		sessionRepo: sessionRepo,
		banCache:    banCache,
	}
}

func (s *AdminService) BanUser(ctx context.Context, adminID uuid.UUID, targetUserID uuid.UUID) error {
	if adminID == targetUserID {
		return domain.ErrCannotBanSelf
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return domain.ErrUserNotFound
	}

	if target.Role == domain.UserRoleAdmin || target.Role == domain.UserRoleSuperAdmin {
		return domain.ErrCannotBanAdmin
	}

	if target.BannedAt != nil {
		return domain.ErrUserAlreadyBanned
	}

	now := time.Now()
	target.BannedAt = &now
	target.UpdatedAt = now

	if err := s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	_ = s.banCache.SetBanned(ctx, targetUserID)

	return s.sessionRepo.DeleteByUserID(ctx, targetUserID)
}

func (s *AdminService) UnbanUser(ctx context.Context, adminID uuid.UUID, targetUserID uuid.UUID) error {
	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return domain.ErrUserNotFound
	}

	if target.BannedAt == nil {
		return domain.ErrUserNotBanned
	}

	target.BannedAt = nil
	target.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, target); err != nil {
		return err
	}

	_ = s.banCache.RemoveBanned(ctx, targetUserID)
	return nil
}

func (s *AdminService) DeleteReview(ctx context.Context, reviewID uuid.UUID) error {
	review, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}
	return s.reviewRepo.Delete(ctx, reviewID)
}

func (s *AdminService) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return domain.ErrCommentNotFound
	}
	return s.commentRepo.Delete(ctx, commentID)
}

func (s *AdminService) SetUserRole(ctx context.Context, superAdminID uuid.UUID, targetUserID uuid.UUID, newRole domain.UserRole) error {
	if superAdminID == targetUserID {
		return domain.ErrCannotChangeOwnRole
	}

	if newRole != domain.UserRoleUser && newRole != domain.UserRoleAdmin && newRole != domain.UserRoleSuperAdmin {
		return domain.ErrInvalidRole
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if target == nil {
		return domain.ErrUserNotFound
	}

	target.Role = newRole
	target.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, target)
}

func (s *AdminService) SeedSuperAdmin(ctx context.Context, input ports.SeedSuperAdminInput) error {
	exists, err := s.userRepo.ExistsByRole(ctx, domain.UserRoleSuperAdmin)
	if err != nil {
		return fmt.Errorf("failed to check for existing superadmin: %w", err)
	}
	if exists {
		return nil
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:            uuid.New(),
		Email:         input.Email,
		EmailVerified: true,
		Username:      input.Username,
		PasswordHash:  &passwordHash,
		Role:          domain.UserRoleSuperAdmin,
		Theme:         domain.UserThemeSystem,
		Locale:        domain.UserLocaleEN,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create superadmin: %w", err)
	}

	return nil
}
