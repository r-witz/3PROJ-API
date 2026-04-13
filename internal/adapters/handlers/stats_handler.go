package handlers

import (
	"fmt"
	"math"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StatsHandler struct {
	statsService ports.StatsService
	blockService ports.BlockService
	banCache     ports.BanCache
}

func NewStatsHandler(statsService ports.StatsService, blockService ports.BlockService, banCache ports.BanCache) *StatsHandler {
	return &StatsHandler{statsService: statsService, blockService: blockService, banCache: banCache}
}

type ReviewStatsResponse struct {
	TotalReviews       int            `json:"total_reviews" example:"42"`
	AverageRating      *float64       `json:"average_rating" example:"3.7"`
	RatingDistribution map[string]int `json:"rating_distribution"`
	LikesReceived      int            `json:"likes_received" example:"156"`
	CommentsReceived   int            `json:"comments_received" example:"89"`
}

type CollectionStatsResponse struct {
	TotalCollections int `json:"total_collections" example:"5"`
	TotalItems       int `json:"total_items" example:"234"`
}

type WatchedStatsResponse struct {
	TotalMovies    int    `json:"total_movies" example:"180"`
	TotalRuntime   int    `json:"total_runtime" example:"21600"`
	RuntimeDisplay string `json:"runtime_display" example:"15d 0h 0m"`
}

type SocialStatsResponse struct {
	FollowersCount int `json:"followers_count" example:"150"`
	FollowingCount int `json:"following_count" example:"75"`
}

type UserStatsResponse struct {
	Reviews     ReviewStatsResponse     `json:"reviews"`
	Collections CollectionStatsResponse `json:"collections"`
	Watched     WatchedStatsResponse    `json:"watched"`
	Social      SocialStatsResponse     `json:"social"`
}

// @Summary      Get user statistics
// @Description  Get detailed statistics for a user profile including review stats, collection stats, watch time, and social stats. Returns 404 if the user is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the target user.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Success      200 {object} response.Response{data=UserStatsResponse} "User statistics"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "User not found or banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/stats [get]
func (h *StatsHandler) GetUserStats(c *gin.Context) {
	idStr := c.Param("userId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if IsBannedForCaller(c, h.banCache, id) {
		response.HandleError(c, domain.ErrUserNotFound)
		return
	}

	ctx := c.Request.Context()

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != id {
		if blocked, err := h.blockService.IsBlocked(ctx, currentUserID, id); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	stats, err := h.statsService.GetUserStats(ctx, id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	var avgRating *float64
	if stats.AverageRating != nil {
		rounded := math.Round(*stats.AverageRating*100) / 100
		avgRating = &rounded
	}

	response.Success(c, UserStatsResponse{
		Reviews: ReviewStatsResponse{
			TotalReviews:       stats.ReviewCount,
			AverageRating:      avgRating,
			RatingDistribution: stats.RatingDistrib,
			LikesReceived:      stats.LikesReceived,
			CommentsReceived:   stats.CommentsReceived,
		},
		Collections: CollectionStatsResponse{
			TotalCollections: stats.CollectionCount,
			TotalItems:       stats.TotalItems,
		},
		Watched: WatchedStatsResponse{
			TotalMovies:    stats.WatchedMovies,
			TotalRuntime:   stats.WatchedRuntime,
			RuntimeDisplay: formatRuntime(stats.WatchedRuntime),
		},
		Social: SocialStatsResponse{
			FollowersCount: stats.FollowersCount,
			FollowingCount: stats.FollowingCount,
		},
	})
}

func formatRuntime(minutes int) string {
	days := minutes / (24 * 60)
	remaining := minutes % (24 * 60)
	hours := remaining / 60
	mins := remaining % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
