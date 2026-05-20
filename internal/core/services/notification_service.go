package services

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type notificationService struct {
	notifRepo     ports.NotificationRepository
	notifPrefRepo ports.NotificationPreferencesRepository
}

func NewNotificationService(
	notifRepo ports.NotificationRepository,
	notifPrefRepo ports.NotificationPreferencesRepository,
) ports.NotificationService {
	return &notificationService{
		notifRepo:     notifRepo,
		notifPrefRepo: notifPrefRepo,
	}
}

func (s *notificationService) Notify(ctx context.Context, input ports.NotifyInput) (*domain.Notification, error) {
	peerTriggered := input.Type != domain.NotificationTypeSystem &&
		input.Type != domain.NotificationTypeAchievementUnlocked
	if peerTriggered && input.ActorID == input.UserID {
		return nil, nil
	}

	prefs, err := s.notifPrefRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if prefs == nil {
		prefs = domain.DefaultNotificationPreferences(input.UserID)
	}

	if !prefs.IsEnabled(input.Type) {
		return nil, nil
	}

	var actorID *uuid.UUID
	if peerTriggered {
		actorID = &input.ActorID
	}

	notification := &domain.Notification{
		ID:            uuid.New(),
		UserID:        input.UserID,
		ActorID:       actorID,
		Type:          input.Type,
		ReviewID:      input.ReviewID,
		CommentID:     input.CommentID,
		AchievementID: input.AchievementID,
		Message:       input.Message,
		CreatedAt:     time.Now(),
	}

	if err := s.notifRepo.Create(ctx, notification); err != nil {
		return nil, domain.ErrInternal
	}

	return notification, nil
}

func (s *notificationService) GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Notification, int, error) {
	notifications, err := s.notifRepo.GetByUserIDPaginated(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	total, err := s.notifRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, domain.ErrInternal
	}

	return notifications, total, nil
}

func (s *notificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.notifRepo.CountUnreadByUserID(ctx, userID)
	if err != nil {
		return 0, domain.ErrInternal
	}
	return count, nil
}

func (s *notificationService) MarkAsRead(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	notification, err := s.notifRepo.GetByID(ctx, notificationID)
	if err != nil {
		return domain.ErrInternal
	}
	if notification == nil {
		return domain.ErrNotificationNotFound
	}
	if notification.UserID != userID {
		return domain.ErrForbidden
	}

	return s.notifRepo.MarkAsRead(ctx, notificationID)
}

func (s *notificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.notifRepo.MarkAllAsRead(ctx, userID); err != nil {
		return domain.ErrInternal
	}
	return nil
}

func (s *notificationService) DeleteAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.notifRepo.DeleteAllByUserID(ctx, userID); err != nil {
		return domain.ErrInternal
	}
	return nil
}

func (s *notificationService) Delete(ctx context.Context, notificationID uuid.UUID, userID uuid.UUID) error {
	notification, err := s.notifRepo.GetByID(ctx, notificationID)
	if err != nil {
		return domain.ErrInternal
	}
	if notification == nil {
		return domain.ErrNotificationNotFound
	}
	if notification.UserID != userID {
		return domain.ErrForbidden
	}

	if err := s.notifRepo.Delete(ctx, notificationID); err != nil {
		return domain.ErrInternal
	}
	return nil
}

func (s *notificationService) GetPreferences(ctx context.Context, userID uuid.UUID) (*domain.NotificationPreferences, error) {
	prefs, err := s.notifPrefRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if prefs == nil {
		return domain.DefaultNotificationPreferences(userID), nil
	}
	return prefs, nil
}

func (s *notificationService) UpdatePreferences(ctx context.Context, userID uuid.UUID, input ports.UpdateNotificationPreferencesInput) (*domain.NotificationPreferences, error) {
	prefs, err := s.notifPrefRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if prefs == nil {
		prefs = domain.DefaultNotificationPreferences(userID)
	}

	if input.LikeReview != nil {
		prefs.LikeReview = *input.LikeReview
	}
	if input.LikeComment != nil {
		prefs.LikeComment = *input.LikeComment
	}
	if input.NewComment != nil {
		prefs.NewComment = *input.NewComment
	}
	if input.NewFollow != nil {
		prefs.NewFollow = *input.NewFollow
	}
	if input.System != nil {
		prefs.System = *input.System
	}
	if input.AchievementUnlocked != nil {
		prefs.AchievementUnlocked = *input.AchievementUnlocked
	}

	prefs.UpdatedAt = time.Now()

	if err := s.notifPrefRepo.Upsert(ctx, prefs); err != nil {
		return nil, domain.ErrInternal
	}

	return prefs, nil
}
