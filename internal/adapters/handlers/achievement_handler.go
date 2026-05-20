package handlers

import (
	"strconv"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AchievementHandler struct {
	achievementSvc ports.AchievementService
}

func NewAchievementHandler(achievementSvc ports.AchievementService) *AchievementHandler {
	return &AchievementHandler{achievementSvc: achievementSvc}
}

type AchievementProgressResponse struct {
	Current int `json:"current" example:"3"`
	Target  int `json:"target" example:"25"`
}

type AchievementResponse struct {
	ID          string                       `json:"id" example:"018f1234-1234-7abc-8def-123456789abc" format:"uuid"`
	Code        string                       `json:"code" example:"first_review"`
	Name        string                       `json:"name" example:"First Impressions"`
	Description string                       `json:"description" example:"Write your first review."`
	Category    string                       `json:"category" example:"reviewing" enums:"reviewing,watching,social,collecting,discovery"`
	Tier        string                       `json:"tier" example:"bronze" enums:"bronze,silver,gold,platinum"`
	IconURL     *string                      `json:"icon_url,omitempty" example:"https://cdn.duskforge.io/achievements/first_review.png"`
	Family      string                       `json:"family,omitempty" example:"review_count"`
	Secret      bool                         `json:"secret" example:"false"`
	System      bool                         `json:"system" example:"true"`
	Unlocked    bool                         `json:"unlocked" example:"true"`
	UnlockedAt  *string                      `json:"unlocked_at,omitempty" example:"2026-04-17T10:00:00Z"`
	Progress    *AchievementProgressResponse `json:"progress,omitempty"`
}

// @Summary      List achievements
// @Description  Returns one entry per progression ladder (family) - not one per tier. A family groups every achievement that shares the same progression signal (e.g. all four `review_count` tiers roll up into a single entry). Each entry describes the caller's state on that ladder:
// @Description  • `family` is a stable string identifier for the ladder (e.g. `review_count`, `watched_runtime`, `rating_given:5`) - use it to key rows in the UI, since `id` changes as the caller climbs tiers.
// @Description  • The top-level achievement fields (`id`, `code`, `name`, `description`, `tier`, `icon_url`) describe the **highest tier the caller has unlocked** in that family. When nothing is unlocked yet, the bronze tier is returned.
// @Description  • `unlocked` is `true` whenever the returned tier has been earned (every case except "nothing unlocked yet"). `unlocked_at` carries the timestamp of that unlock.
// @Description  • `progress.target` is the threshold of the **next tier** to work toward. When the ladder is maxed out (platinum unlocked) `target` stays on the platinum threshold and `current == target`. When nothing is unlocked yet, `target` is the bronze threshold.
// @Description  • `progress.current` is the caller's current value, capped at `target`. For unauthenticated callers `current` is 0.
// @Description  Secret achievements are hidden from unauthenticated callers and from authenticated callers who have not yet unlocked them. Results are ordered by the family's first-tier `sort_order`, and can be filtered to a single category.
// @Tags         achievements
// @Produce      json
// @Security     BearerAuth
// @Param        category query string false "Filter by category" Enums(reviewing, watching, social, collecting, discovery)
// @Success      200 {object} response.Response{data=[]AchievementResponse} "Family roll-up with per-caller progress toward the next tier"
// @Failure      400 {object} response.Response "Invalid category value"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /achievements [get]
func (h *AchievementHandler) List(c *gin.Context) {
	var requester *uuid.UUID
	if id, ok := middleware.GetUserID(c); ok {
		requester = &id
	}

	var catPtr *domain.AchievementCategory
	if raw := c.Query("category"); raw != "" {
		cat := domain.AchievementCategory(raw)
		if !domain.ValidAchievementCategories[cat] {
			response.BadRequest(c, "Invalid category", nil)
			return
		}
		catPtr = &cat
	}

	items, err := h.achievementSvc.List(c.Request.Context(), requester, catPtr)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]AchievementResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toAchievementResponse(item))
	}
	response.Success(c, resp)
}

// @Summary      Get achievement by ID
// @Description  Fetch a single achievement. When authenticated, the response carries the caller's unlock state and progress toward the criterion. Returns 404 if the achievement is inactive, missing, or secret and not yet unlocked by the caller.
// @Tags         achievements
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Achievement ID" format(uuid)
// @Success      200 {object} response.Response{data=AchievementResponse} "Achievement detail"
// @Failure      400 {object} response.Response "Invalid achievement ID"
// @Failure      404 {object} response.Response "Achievement not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /achievements/{id} [get]
func (h *AchievementHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid achievement ID", nil)
		return
	}

	var requester *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requester = &uid
	}

	item, err := h.achievementSvc.GetByID(c.Request.Context(), id, requester)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, toAchievementResponse(item))
}

// @Summary      List a user's achievements
// @Description  Returns the family roll-up for the given user - same shape as `GET /achievements`, but with unlock state and progress evaluated against the target user instead of the caller. Intended for public profile pages so visitors can see another user's badges and progression toward the next tier. Secret achievements the target user has not unlocked are hidden.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        category query string false "Filter by category" Enums(reviewing, watching, social, collecting, discovery)
// @Success      200 {object} response.Response{data=[]AchievementResponse} "Family roll-up with the target user's progress"
// @Failure      400 {object} response.Response "Invalid user ID or category"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/achievements [get]
func (h *AchievementHandler) ListForUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	var catPtr *domain.AchievementCategory
	if raw := c.Query("category"); raw != "" {
		cat := domain.AchievementCategory(raw)
		if !domain.ValidAchievementCategories[cat] {
			response.BadRequest(c, "Invalid category", nil)
			return
		}
		catPtr = &cat
	}

	items, err := h.achievementSvc.List(c.Request.Context(), &userID, catPtr)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]AchievementResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toAchievementResponse(item))
	}
	response.Success(c, resp)
}

// @Summary      Recent achievement unlocks for the current user
// @Description  Returns the caller's most recently unlocked achievements, ordered by unlock timestamp descending. Intended for a home-screen "recent wins" widget. Invalid or out-of-range `limit` values fall back to the default of 5.
// @Tags         achievements
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of unlocks to return" default(5) minimum(1) maximum(20)
// @Success      200 {object} response.Response{data=[]AchievementResponse} "Recent unlocks"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/achievements/recent [get]
func (h *AchievementHandler) RecentForMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	limit := 5
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	items, err := h.achievementSvc.ListRecentUnlocksByUser(c.Request.Context(), userID, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]AchievementResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toAchievementResponseUnlocked(item))
	}
	response.Success(c, resp)
}

func toAchievementResponse(item *ports.AchievementWithProgress) AchievementResponse {
	a := item.Achievement
	resp := AchievementResponse{
		ID:          a.ID.String(),
		Code:        a.Code,
		Name:        a.Name,
		Description: a.Description,
		Category:    string(a.Category),
		Tier:        string(a.Tier),
		IconURL:     a.IconURL,
		Family:      item.Family,
		Secret:      a.Secret,
		System:      a.System,
		Unlocked:    item.Unlocked,
	}
	if item.UnlockedAt != nil {
		s := item.UnlockedAt.UnlockedAt.Format("2006-01-02T15:04:05Z")
		resp.UnlockedAt = &s
	}
	if item.Progress.Target > 0 {
		resp.Progress = &AchievementProgressResponse{
			Current: item.Progress.Current,
			Target:  item.Progress.Target,
		}
	}
	return resp
}

func toAchievementResponseUnlocked(item *ports.UnlockedAchievement) AchievementResponse {
	return toAchievementResponse(&ports.AchievementWithProgress{
		Achievement: item.Achievement,
		Unlocked:    true,
		UnlockedAt:  item.UnlockedAt,
		Progress: ports.AchievementProgress{
			Current: 0,
			Target:  0,
		},
	})
}

