package middleware

import (
	"context"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const activityQueueKey = "activity_queue"

type ActivityEventAction string

const (
	ActivityCreate ActivityEventAction = "create"
	ActivityDelete ActivityEventAction = "delete"
)

type ActivityEvent struct {
	Action       ActivityEventAction
	Type         domain.ActivityType
	UserID       uuid.UUID
	ReviewID     *uuid.UUID
	CollectionID *uuid.UUID
	CommentID    *uuid.UUID
	TMDBID       *int
	TargetUserID *uuid.UUID

	// SuppressLog skips the activity-feed write but still fires achievement
	// evaluation. Use for side-effect events we don't want users to see (e.g.
	// adding to the "watched" system collection).
	SuppressLog bool
}

// QueueActivity queues an activity event to be processed after the handler completes.
func QueueActivity(c *gin.Context, event ActivityEvent) {
	raw, _ := c.Get(activityQueueKey)
	var queue []ActivityEvent
	if raw != nil {
		queue = raw.([]ActivityEvent)
	}
	queue = append(queue, event)
	c.Set(activityQueueKey, queue)
}

// categoryForActivityType maps an activity type to the achievement category
// whose criteria should be re-evaluated. Returning an empty string means no
// achievement category is relevant for this event.
func categoryForActivityType(t domain.ActivityType) domain.AchievementCategory {
	switch t {
	case domain.ActivityTypeReviewCreated, domain.ActivityTypeReviewUpdated:
		return domain.AchievementCategoryReviewing
	case domain.ActivityTypeCollectionItemAdded, domain.ActivityTypeWatchlistItemAdded:
		return domain.AchievementCategoryWatching
	case domain.ActivityTypeReviewLiked, domain.ActivityTypeCommentLiked,
		domain.ActivityTypeUserFollowed, domain.ActivityTypeCommentCreated:
		return domain.AchievementCategorySocial
	case domain.ActivityTypeCollectionCreated:
		return domain.AchievementCategoryCollecting
	default:
		return ""
	}
}

// ActivityLogger is a Gin middleware that processes queued activity events
// after the handler completes successfully (2xx status). It also invokes the
// achievement evaluator for each distinct (userID, category) pair seen in the
// queue so new unlocks surface in the same request cycle.
func ActivityLogger(activityRepo ports.ActivityRepository, achievementSvc ports.AchievementService) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() >= 400 {
			return
		}

		raw, exists := c.Get(activityQueueKey)
		if !exists {
			return
		}

		queue := raw.([]ActivityEvent)
		ctx := c.Request.Context()

		type userCat struct {
			userID   uuid.UUID
			category domain.AchievementCategory
		}
		evalSet := make(map[userCat]struct{})
		socialTargets := make(map[uuid.UUID]struct{})

		for _, event := range queue {
			switch event.Action {
			case ActivityCreate:
				if !event.SuppressLog {
					_ = activityRepo.Create(ctx, &domain.Activity{
						ID:           uuid.New(),
						UserID:       event.UserID,
						Type:         event.Type,
						ReviewID:     event.ReviewID,
						CollectionID: event.CollectionID,
						CommentID:    event.CommentID,
						TMDBID:       event.TMDBID,
						TargetUserID: event.TargetUserID,
						CreatedAt:    time.Now(),
					})
				}
				if achievementSvc != nil {
					if cat := categoryForActivityType(event.Type); cat != "" {
						evalSet[userCat{event.UserID, cat}] = struct{}{}
					}
					// Peer-affecting events: a like on someone's review or a
					// new follower should re-evaluate social criteria for the
					// recipient too.
					if event.Type == domain.ActivityTypeReviewLiked ||
						event.Type == domain.ActivityTypeCommentLiked ||
						event.Type == domain.ActivityTypeUserFollowed {
						if event.TargetUserID != nil {
							socialTargets[*event.TargetUserID] = struct{}{}
						}
					}
				}
			case ActivityDelete:
				_ = activityRepo.DeleteByFields(ctx, event.UserID, event.Type,
					event.ReviewID, event.CollectionID, event.CommentID, event.TMDBID)
			default:
				logger.Logger.Warn("unknown activity event action", zap.String("action", string(event.Action)))
			}
		}

		if achievementSvc == nil {
			return
		}

		// Evaluations run in the background so the response isn't blocked on
		// downstream DB work. Using context.Background() keeps them alive past
		// the request lifecycle.
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Logger.Error("achievement-evaluator panic", zap.Any("panic", r))
				}
			}()
			bg := context.Background()
			for uc := range evalSet {
				if _, err := achievementSvc.EvaluateForEvent(bg, uc.userID, uc.category); err != nil {
					logger.Logger.Warn("achievement evaluation failed",
						zap.Stringer("user", uc.userID),
						zap.String("category", string(uc.category)),
						zap.Error(err),
					)
				}
			}
			for uid := range socialTargets {
				if _, err := achievementSvc.EvaluateForEvent(bg, uid, domain.AchievementCategorySocial); err != nil {
					logger.Logger.Warn("social achievement evaluation failed",
						zap.Stringer("user", uid),
						zap.Error(err),
					)
				}
			}
		}()
	}
}
