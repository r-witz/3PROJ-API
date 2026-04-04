package middleware

import (
	"fmt"
	"time"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// ActivityLogger is a Gin middleware that processes queued activity events
// after the handler completes successfully (2xx status).
func ActivityLogger(activityRepo ports.ActivityRepository) gin.HandlerFunc {
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

		for _, event := range queue {
			switch event.Action {
			case ActivityCreate:
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
			case ActivityDelete:
				_ = activityRepo.DeleteByFields(ctx, event.UserID, event.Type,
					event.ReviewID, event.CollectionID, event.CommentID, event.TMDBID)
			default:
				fmt.Printf("unknown activity event action: %s\n", event.Action)
			}
		}
	}
}
