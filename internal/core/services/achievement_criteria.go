package services

import (
	"context"
	"encoding/json"
	"fmt"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type criterionKind string

const (
	criterionReviewCount       criterionKind = "review_count"
	criterionWatchedCount      criterionKind = "watched_count"
	criterionWatchedRuntime    criterionKind = "watched_runtime"
	criterionLikesReceived     criterionKind = "likes_received"
	criterionFollowersCount    criterionKind = "followers_count"
	criterionCommentsAuthored  criterionKind = "comments_authored"
	criterionRatingGiven       criterionKind = "rating_given"
	criterionCustomCollections criterionKind = "custom_collections"
)

type criterionSpec struct {
	Kind   criterionKind   `json:"kind"`
	Params json.RawMessage `json:"params"`
}

// categoryForCriterion returns which category signal cluster a given criterion
// belongs to. It lets EvaluateForEvent evaluate only criteria relevant to the
// triggering activity category.
func categoryForCriterion(kind criterionKind) domain.AchievementCategory {
	switch kind {
	case criterionReviewCount, criterionRatingGiven:
		return domain.AchievementCategoryReviewing
	case criterionWatchedCount, criterionWatchedRuntime:
		return domain.AchievementCategoryWatching
	case criterionLikesReceived, criterionFollowersCount, criterionCommentsAuthored:
		return domain.AchievementCategorySocial
	case criterionCustomCollections:
		return domain.AchievementCategoryCollecting
	default:
		return ""
	}
}

// evalContext holds shared signals for a single EvaluateForEvent call so
// multiple criteria reuse the same queries.
type evalContext struct {
	ctx             context.Context
	userID          uuid.UUID
	statsRepo       ports.StatsRepository
	achievementRepo ports.AchievementRepository

	stats         *ports.UserStats
	statsLoaded   bool
	commentsCount *int
	customColls   *int
}

func newEvalContext(ctx context.Context, userID uuid.UUID, statsRepo ports.StatsRepository, achievementRepo ports.AchievementRepository) *evalContext {
	return &evalContext{
		ctx:             ctx,
		userID:          userID,
		statsRepo:       statsRepo,
		achievementRepo: achievementRepo,
	}
}

func (e *evalContext) getStats() (*ports.UserStats, error) {
	if !e.statsLoaded {
		s, err := e.statsRepo.GetUserStats(e.ctx, e.userID)
		if err != nil {
			return nil, err
		}
		e.stats = s
		e.statsLoaded = true
	}
	return e.stats, nil
}

type thresholdParams struct {
	Threshold int `json:"threshold"`
}

type runtimeParams struct {
	Minutes int `json:"minutes"`
}

type ratingGivenParams struct {
	Rating    float64 `json:"rating"`
	Threshold int     `json:"threshold"`
}

// evaluateCriterion resolves the current/target for one achievement against
// the caller's signals. Returns (current, target, done, error).
func evaluateCriterion(e *evalContext, raw json.RawMessage) (int, int, bool, error) {
	var spec criterionSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return 0, 0, false, fmt.Errorf("invalid criterion: %w", err)
	}

	switch spec.Kind {
	case criterionReviewCount:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		stats, err := e.getStats()
		if err != nil {
			return 0, 0, false, err
		}
		return stats.ReviewCount, target, stats.ReviewCount >= target, nil

	case criterionWatchedCount:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		stats, err := e.getStats()
		if err != nil {
			return 0, 0, false, err
		}
		return stats.WatchedMovies, target, stats.WatchedMovies >= target, nil

	case criterionWatchedRuntime:
		var p runtimeParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return 0, 0, false, fmt.Errorf("invalid params: %w", err)
		}
		stats, err := e.getStats()
		if err != nil {
			return 0, 0, false, err
		}
		return stats.WatchedRuntime, p.Minutes, stats.WatchedRuntime >= p.Minutes, nil

	case criterionLikesReceived:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		stats, err := e.getStats()
		if err != nil {
			return 0, 0, false, err
		}
		return stats.LikesReceived, target, stats.LikesReceived >= target, nil

	case criterionFollowersCount:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		stats, err := e.getStats()
		if err != nil {
			return 0, 0, false, err
		}
		return stats.FollowersCount, target, stats.FollowersCount >= target, nil

	case criterionCommentsAuthored:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		if e.commentsCount == nil {
			n, err := e.achievementRepo.CountCommentsByUser(e.ctx, e.userID)
			if err != nil {
				return 0, 0, false, err
			}
			e.commentsCount = &n
		}
		return *e.commentsCount, target, *e.commentsCount >= target, nil

	case criterionRatingGiven:
		var p ratingGivenParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return 0, 0, false, fmt.Errorf("invalid params: %w", err)
		}
		count, err := e.achievementRepo.CountReviewsByUserWithRating(e.ctx, e.userID, p.Rating)
		if err != nil {
			return 0, 0, false, err
		}
		return count, p.Threshold, count >= p.Threshold, nil

	case criterionCustomCollections:
		target, err := parseThreshold(spec.Params)
		if err != nil {
			return 0, 0, false, err
		}
		if e.customColls == nil {
			n, err := e.achievementRepo.CountCustomCollectionsByUser(e.ctx, e.userID)
			if err != nil {
				return 0, 0, false, err
			}
			e.customColls = &n
		}
		return *e.customColls, target, *e.customColls >= target, nil

	default:
		return 0, 0, false, fmt.Errorf("unknown criterion kind: %s", spec.Kind)
	}
}

func parseThreshold(raw json.RawMessage) (int, error) {
	var p thresholdParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, fmt.Errorf("invalid params: %w", err)
	}
	return p.Threshold, nil
}

func validateCriterion(raw json.RawMessage) error {
	var spec criterionSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return err
	}
	switch spec.Kind {
	case criterionReviewCount, criterionWatchedCount, criterionLikesReceived,
		criterionFollowersCount, criterionCommentsAuthored, criterionCustomCollections:
		var p thresholdParams
		if err := json.Unmarshal(spec.Params, &p); err != nil || p.Threshold <= 0 {
			return fmt.Errorf("threshold must be a positive integer")
		}
	case criterionWatchedRuntime:
		var p runtimeParams
		if err := json.Unmarshal(spec.Params, &p); err != nil || p.Minutes <= 0 {
			return fmt.Errorf("minutes must be a positive integer")
		}
	case criterionRatingGiven:
		var p ratingGivenParams
		if err := json.Unmarshal(spec.Params, &p); err != nil {
			return err
		}
		if p.Rating < 0.5 || p.Rating > 5.0 || p.Rating*2 != float64(int(p.Rating*2)) {
			return fmt.Errorf("rating must be a 0.5 increment between 0.5 and 5.0")
		}
		if p.Threshold <= 0 {
			return fmt.Errorf("threshold must be a positive integer")
		}
	default:
		return fmt.Errorf("unknown criterion kind: %s", spec.Kind)
	}
	return nil
}
