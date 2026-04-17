package services

import (
	"context"
	"encoding/json"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/logger"
	ws "duskforge-api/pkg/websocket"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type achievementService struct {
	achievementRepo ports.AchievementRepository
	statsRepo       ports.StatsRepository
	notifService    ports.NotificationService
	hub             *ws.Hub
}

func NewAchievementService(
	achievementRepo ports.AchievementRepository,
	statsRepo ports.StatsRepository,
	notifService ports.NotificationService,
	hub *ws.Hub,
) ports.AchievementService {
	return &achievementService{
		achievementRepo: achievementRepo,
		statsRepo:       statsRepo,
		notifService:    notifService,
		hub:             hub,
	}
}

// --- Catalog CRUD ---

func (s *achievementService) Create(ctx context.Context, input ports.CreateAchievementInput) (*domain.Achievement, error) {
	if err := validateCatalogInput(input.Code, input.Name, input.Description, input.Category, input.Tier, input.Criterion); err != nil {
		return nil, err
	}

	existing, err := s.achievementRepo.GetByCode(ctx, input.Code)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if existing != nil {
		return nil, domain.ErrAchievementCodeExists
	}

	now := time.Now()
	a := &domain.Achievement{
		ID:          uuid.New(),
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		Tier:        input.Tier,
		IconURL:     input.IconURL,
		Criterion:   input.Criterion,
		Secret:      input.Secret,
		Active:      input.Active,
		System:      false, // system flag is reserved for seeded rows.
		SortOrder:   input.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.achievementRepo.Create(ctx, a); err != nil {
		return nil, domain.ErrInternal
	}
	return a, nil
}

func (s *achievementService) Update(ctx context.Context, id uuid.UUID, input ports.UpdateAchievementInput) (*domain.Achievement, error) {
	a, err := s.achievementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if a == nil {
		return nil, domain.ErrAchievementNotFound
	}
	if a.System {
		return nil, domain.ErrAchievementSystemLocked
	}

	if input.Name != nil {
		a.Name = *input.Name
	}
	if input.Description != nil {
		a.Description = *input.Description
	}
	if input.Category != nil {
		if !domain.ValidAchievementCategories[*input.Category] {
			return nil, domain.ErrAchievementInvalidCategory
		}
		a.Category = *input.Category
	}
	if input.Tier != nil {
		if !domain.ValidAchievementTiers[*input.Tier] {
			return nil, domain.ErrAchievementInvalidTier
		}
		a.Tier = *input.Tier
	}
	if input.IconURL != nil {
		a.IconURL = input.IconURL
	}
	if input.Criterion != nil {
		if err := validateCriterion(input.Criterion); err != nil {
			return nil, domain.ErrAchievementInvalidCriterion
		}
		a.Criterion = input.Criterion
	}
	if input.Secret != nil {
		a.Secret = *input.Secret
	}
	if input.Active != nil {
		a.Active = *input.Active
	}
	if input.SortOrder != nil {
		a.SortOrder = *input.SortOrder
	}
	a.UpdatedAt = time.Now()

	if err := s.achievementRepo.Update(ctx, a); err != nil {
		return nil, domain.ErrInternal
	}
	return a, nil
}

func (s *achievementService) Delete(ctx context.Context, id uuid.UUID) error {
	a, err := s.achievementRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrInternal
	}
	if a == nil {
		return domain.ErrAchievementNotFound
	}
	if a.System {
		return domain.ErrAchievementSystemLocked
	}
	if err := s.achievementRepo.Delete(ctx, id); err != nil {
		return domain.ErrInternal
	}
	return nil
}

// --- Read ---

func (s *achievementService) List(ctx context.Context, requesterID *uuid.UUID, category *domain.AchievementCategory) ([]*ports.AchievementWithProgress, error) {
	all, err := s.achievementRepo.List(ctx, ports.AchievementListFilter{
		OnlyActive: true,
		Category:   category,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	unlockDates := map[uuid.UUID]*domain.UserAchievement{}
	if requesterID != nil {
		list, err := s.achievementRepo.GetUnlockedByUser(ctx, *requesterID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		for _, ua := range list {
			unlockDates[ua.AchievementID] = ua
		}
	}

	var evalCtx *evalContext
	if requesterID != nil {
		evalCtx = newEvalContext(ctx, *requesterID, s.statsRepo, s.achievementRepo)
	}

	out := make([]*ports.AchievementWithProgress, 0, len(all))
	for _, a := range all {
		unlockRec, isUnlocked := unlockDates[a.ID]
		if a.Secret && !isUnlocked {
			continue
		}

		item := &ports.AchievementWithProgress{
			Achievement: a,
			Unlocked:    isUnlocked,
			UnlockedAt:  unlockRec,
		}

		if evalCtx != nil {
			current, target, _, err := evaluateCriterion(evalCtx, a.Criterion)
			if err == nil {
				if current > target {
					current = target
				}
				item.Progress = ports.AchievementProgress{Current: current, Target: target}
			}
		} else {
			// Unauthenticated: expose target only, via a lightweight parse.
			if target, err := extractTarget(a.Criterion); err == nil {
				item.Progress = ports.AchievementProgress{Target: target}
			}
		}

		out = append(out, item)
	}
	return out, nil
}

func (s *achievementService) GetByID(ctx context.Context, id uuid.UUID, requesterID *uuid.UUID) (*ports.AchievementWithProgress, error) {
	a, err := s.achievementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrInternal
	}
	if a == nil || !a.Active {
		return nil, domain.ErrAchievementNotFound
	}

	var (
		isUnlocked bool
		unlockedAt *domain.UserAchievement
		progress   ports.AchievementProgress
	)

	if requesterID != nil {
		list, err := s.achievementRepo.GetUnlockedByUser(ctx, *requesterID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		for _, ua := range list {
			if ua.AchievementID == a.ID {
				unlockedAt = ua
				isUnlocked = true
				break
			}
		}

		evalCtx := newEvalContext(ctx, *requesterID, s.statsRepo, s.achievementRepo)
		if current, target, _, err := evaluateCriterion(evalCtx, a.Criterion); err == nil {
			if current > target {
				current = target
			}
			progress = ports.AchievementProgress{Current: current, Target: target}
		}
	} else {
		if target, err := extractTarget(a.Criterion); err == nil {
			progress = ports.AchievementProgress{Target: target}
		}
	}

	if a.Secret && !isUnlocked {
		return nil, domain.ErrAchievementNotFound
	}

	return &ports.AchievementWithProgress{
		Achievement: a,
		Unlocked:    isUnlocked,
		UnlockedAt:  unlockedAt,
		Progress:    progress,
	}, nil
}

func (s *achievementService) ListUnlockedByUser(ctx context.Context, userID uuid.UUID) ([]*ports.UnlockedAchievement, error) {
	unlocks, err := s.achievementRepo.GetUnlockedByUser(ctx, userID)
	if err != nil {
		return nil, domain.ErrInternal
	}
	return s.hydrateUnlocks(ctx, unlocks)
}

func (s *achievementService) ListRecentUnlocksByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*ports.UnlockedAchievement, error) {
	if limit <= 0 {
		limit = 5
	}
	unlocks, err := s.achievementRepo.GetRecentUnlocksByUser(ctx, userID, limit)
	if err != nil {
		return nil, domain.ErrInternal
	}
	return s.hydrateUnlocks(ctx, unlocks)
}

func (s *achievementService) hydrateUnlocks(ctx context.Context, unlocks []*domain.UserAchievement) ([]*ports.UnlockedAchievement, error) {
	if len(unlocks) == 0 {
		return []*ports.UnlockedAchievement{}, nil
	}

	out := make([]*ports.UnlockedAchievement, 0, len(unlocks))
	for _, ua := range unlocks {
		a, err := s.achievementRepo.GetByID(ctx, ua.AchievementID)
		if err != nil {
			return nil, domain.ErrInternal
		}
		if a == nil {
			continue
		}
		out = append(out, &ports.UnlockedAchievement{Achievement: a, UnlockedAt: ua})
	}
	return out, nil
}

// --- Evaluation ---

func (s *achievementService) EvaluateForEvent(ctx context.Context, userID uuid.UUID, category domain.AchievementCategory) ([]*domain.Achievement, error) {
	candidates, err := s.achievementRepo.List(ctx, ports.AchievementListFilter{
		OnlyActive: true,
		Category:   &category,
	})
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	unlocked, err := s.achievementRepo.GetUnlockedIDsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	evalCtx := newEvalContext(ctx, userID, s.statsRepo, s.achievementRepo)

	var newlyUnlocked []*domain.Achievement
	for _, a := range candidates {
		if _, already := unlocked[a.ID]; already {
			continue
		}
		_, _, done, err := evaluateCriterion(evalCtx, a.Criterion)
		if err != nil {
			logger.Logger.Warn("achievement criterion evaluation failed",
				zap.String("achievement", a.Code),
				zap.Error(err),
			)
			continue
		}
		if !done {
			continue
		}

		inserted, err := s.achievementRepo.Unlock(ctx, userID, a.ID)
		if err != nil {
			logger.Logger.Warn("failed to unlock achievement",
				zap.String("achievement", a.Code),
				zap.Stringer("user", userID),
				zap.Error(err),
			)
			continue
		}
		if !inserted {
			continue
		}

		newlyUnlocked = append(newlyUnlocked, a)
		s.emitUnlockNotification(ctx, userID, a)
	}
	return newlyUnlocked, nil
}

func (s *achievementService) emitUnlockNotification(ctx context.Context, userID uuid.UUID, a *domain.Achievement) {
	msg := a.Name
	notif, err := s.notifService.Notify(ctx, ports.NotifyInput{
		UserID:        userID,
		ActorID:       userID,
		Type:          domain.NotificationTypeAchievementUnlocked,
		AchievementID: &a.ID,
		Message:       &msg,
	})
	if err != nil {
		logger.Logger.Warn("failed to create achievement notification",
			zap.String("achievement", a.Code),
			zap.Error(err),
		)
		return
	}
	if notif == nil || s.hub == nil {
		return
	}
	s.hub.SendToUser(userID, ws.Event{
		Type: ws.EventNotificationNew,
		Data: notif,
	})
}

// --- Helpers ---

func validateCatalogInput(code, name, description string, category domain.AchievementCategory, tier domain.AchievementTier, criterion json.RawMessage) error {
	if code == "" || name == "" || description == "" {
		return domain.ErrInvalidInput
	}
	if !domain.ValidAchievementCategories[category] {
		return domain.ErrAchievementInvalidCategory
	}
	if !domain.ValidAchievementTiers[tier] {
		return domain.ErrAchievementInvalidTier
	}
	if err := validateCriterion(criterion); err != nil {
		return domain.ErrAchievementInvalidCriterion
	}
	return nil
}

func extractTarget(raw json.RawMessage) (int, error) {
	var spec criterionSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return 0, err
	}
	switch spec.Kind {
	case criterionWatchedRuntime:
		var p runtimeParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return 0, err
		}
		return p.Minutes, nil
	case criterionRatingGiven:
		var p ratingGivenParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return 0, err
		}
		return p.Threshold, nil
	default:
		var p thresholdParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return 0, err
		}
		return p.Threshold, nil
	}
}
