package services

import (
	"context"
	"encoding/json"
	"sort"

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

	type familyMember struct {
		achievement *domain.Achievement
		target      int
	}
	families := map[string][]familyMember{}
	familyOrder := []string{}
	familyMinSort := map[string]int{}

	for _, a := range all {
		if _, isUnlocked := unlockDates[a.ID]; a.Secret && !isUnlocked {
			continue
		}
		key, err := familyKeyForCriterion(a.Criterion)
		if err != nil {
			continue
		}
		target, err := extractTarget(a.Criterion)
		if err != nil {
			continue
		}
		if _, ok := families[key]; !ok {
			familyOrder = append(familyOrder, key)
			familyMinSort[key] = a.SortOrder
		} else if a.SortOrder < familyMinSort[key] {
			familyMinSort[key] = a.SortOrder
		}
		families[key] = append(families[key], familyMember{achievement: a, target: target})
	}

	sort.SliceStable(familyOrder, func(i, j int) bool {
		return familyMinSort[familyOrder[i]] < familyMinSort[familyOrder[j]]
	})

	out := make([]*ports.AchievementWithProgress, 0, len(familyOrder))
	for _, key := range familyOrder {
		members := families[key]
		sort.SliceStable(members, func(i, j int) bool {
			return members[i].target < members[j].target
		})

		highestIdx := -1
		var highestUnlockRec *domain.UserAchievement
		for i := range members {
			if ua, ok := unlockDates[members[i].achievement.ID]; ok {
				highestIdx = i
				highestUnlockRec = ua
			}
		}

		var (
			representative   familyMember
			representativeOK bool
			progressTarget   familyMember
		)
		switch {
		case highestIdx < 0:
			representative = members[0]
			progressTarget = members[0]
		case highestIdx+1 < len(members):
			representative = members[highestIdx]
			representativeOK = true
			progressTarget = members[highestIdx+1]
		default:
			representative = members[highestIdx]
			representativeOK = true
			progressTarget = members[highestIdx]
		}

		item := &ports.AchievementWithProgress{
			Achievement: representative.achievement,
			Unlocked:    representativeOK,
			Family:      key,
		}
		if representativeOK {
			item.UnlockedAt = highestUnlockRec
		}

		if evalCtx != nil {
			current, target, _, err := evaluateCriterion(evalCtx, progressTarget.achievement.Criterion)
			if err == nil {
				if current > target {
					current = target
				}
				item.Progress = ports.AchievementProgress{Current: current, Target: target}
			}
		} else {
			item.Progress = ports.AchievementProgress{Target: progressTarget.target}
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

	seen := make(map[uuid.UUID]struct{}, len(unlocks))
	ids := make([]uuid.UUID, 0, len(unlocks))
	for _, ua := range unlocks {
		if _, ok := seen[ua.AchievementID]; ok {
			continue
		}
		seen[ua.AchievementID] = struct{}{}
		ids = append(ids, ua.AchievementID)
	}

	achievements, err := s.achievementRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, domain.ErrInternal
	}
	byID := make(map[uuid.UUID]*domain.Achievement, len(achievements))
	for _, a := range achievements {
		byID[a.ID] = a
	}

	out := make([]*ports.UnlockedAchievement, 0, len(unlocks))
	for _, ua := range unlocks {
		a, ok := byID[ua.AchievementID]
		if !ok {
			continue
		}
		out = append(out, &ports.UnlockedAchievement{Achievement: a, UnlockedAt: ua})
	}
	return out, nil
}


func (s *achievementService) EvaluateAllForUser(ctx context.Context, userID uuid.UUID) ([]*domain.Achievement, error) {
	categories := []domain.AchievementCategory{
		domain.AchievementCategoryReviewing,
		domain.AchievementCategoryWatching,
		domain.AchievementCategorySocial,
		domain.AchievementCategoryCollecting,
		domain.AchievementCategoryDiscovery,
	}
	var all []*domain.Achievement
	for _, cat := range categories {
		unlocked, err := s.EvaluateForEvent(ctx, userID, cat)
		if err != nil {
			return all, err
		}
		all = append(all, unlocked...)
	}
	return all, nil
}

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
