package handlers

import (
	"encoding/json"
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

// AchievementProgressResponse represents the caller's progress toward an
// achievement. Current is capped at Target so progress bars never overflow.
type AchievementProgressResponse struct {
	Current int `json:"current" example:"3"`
	Target  int `json:"target" example:"25"`
}

// AchievementResponse is the canonical achievement payload returned everywhere
// achievements are surfaced (list, detail, profile, recent unlocks). Progress
// and unlock fields are populated only when the request is authenticated.
//
// On the catalog List endpoint each entry is a family roll-up: the object
// describes the highest tier the caller has unlocked in that ladder (or the
// bronze tier when nothing is unlocked yet), and `progress.target` is the
// threshold of the next tier — or the current tier's threshold once the
// ladder is maxed out.
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

// CreateAchievementRequest is the admin payload for POST /admin/achievements.
// The `system` flag cannot be set by clients — server always stores false —
// so only seeded migrations can flag an achievement as built-in.
type CreateAchievementRequest struct {
	Code        string          `json:"code" binding:"required" example:"weekend_binger" extensions:"x-order=1"`
	Name        string          `json:"name" binding:"required" example:"Weekend Binger"`
	Description string          `json:"description" binding:"required" example:"Watch 5 films in a single weekend."`
	Category    string          `json:"category" binding:"required" example:"watching" enums:"reviewing,watching,social,collecting,discovery"`
	Tier        string          `json:"tier" binding:"required" example:"silver" enums:"bronze,silver,gold,platinum"`
	IconURL     *string         `json:"icon_url" example:"https://cdn.duskforge.io/achievements/weekend_binger.png"`
	Criterion   json.RawMessage `json:"criterion" binding:"required" swaggertype:"object" example:"{\"kind\":\"watched_count\",\"params\":{\"threshold\":5}}"`
	Secret      bool            `json:"secret" example:"false"`
	Active      *bool           `json:"active" example:"true"`
	SortOrder   int             `json:"sort_order" example:"500"`
}

// UpdateAchievementRequest is the admin payload for PATCH /admin/achievements/:id.
// All fields are optional; only supplied fields change. System achievements
// (the 15 seeded badges) reject any update with 403 ACHIEVEMENT_SYSTEM_LOCKED.
type UpdateAchievementRequest struct {
	Name        *string         `json:"name" example:"Weekend Binger"`
	Description *string         `json:"description" example:"Watch 5 films in a single weekend."`
	Category    *string         `json:"category" example:"watching" enums:"reviewing,watching,social,collecting,discovery"`
	Tier        *string         `json:"tier" example:"gold" enums:"bronze,silver,gold,platinum"`
	IconURL     *string         `json:"icon_url" example:"https://cdn.duskforge.io/achievements/weekend_binger.png"`
	Criterion   json.RawMessage `json:"criterion" swaggertype:"object" example:"{\"kind\":\"watched_count\",\"params\":{\"threshold\":10}}"`
	Secret      *bool           `json:"secret" example:"false"`
	Active      *bool           `json:"active" example:"true"`
	SortOrder   *int            `json:"sort_order" example:"500"`
}

// @Summary      List achievements
// @Description  Returns one entry per progression ladder (family) — not one per tier. A family groups every achievement that shares the same progression signal (e.g. all four `review_count` tiers roll up into a single entry). Each entry describes the caller's state on that ladder:
// @Description  • `family` is a stable string identifier for the ladder (e.g. `review_count`, `watched_runtime`, `rating_given:5`) — use it to key rows in the UI, since `id` changes as the caller climbs tiers.
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

// @Summary      List a user's unlocked achievements
// @Description  Returns every achievement this user has unlocked, ordered by most-recent unlock first. Intended for public profile pages. Each entry has `unlocked=true` and a non-null `unlocked_at`; `progress` is omitted because it's irrelevant once unlocked.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Success      200 {object} response.Response{data=[]AchievementResponse} "Unlocked achievements for the user"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/achievements [get]
func (h *AchievementHandler) ListForUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	items, err := h.achievementSvc.ListUnlockedByUser(c.Request.Context(), userID)
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

// @Summary      Create an achievement (admin)
// @Description  Create a new achievement in the catalog. Requires `admin` or `superadmin` role. The `criterion` field is a JSON object of shape `{"kind":"<kind>","params":{...}}`. Supported kinds: `review_count`, `rating_given`, `watched_count`, `watched_runtime`, `likes_received`, `followers_count`, `comments_authored`, `custom_collections`. Threshold-style kinds take `{"threshold":N}`; `watched_runtime` takes `{"minutes":N}`; `rating_given` takes `{"rating":R,"threshold":N}`. Admin-created achievements are always stored with `system=false` and can be edited or deleted later.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateAchievementRequest true "Achievement definition"
// @Success      201 {object} response.Response{data=AchievementResponse} "Achievement created"
// @Failure      400 {object} response.Response "Invalid input (missing/invalid field or malformed criterion)"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient role (admin required)"
// @Failure      409 {object} response.Response "Achievement code already exists"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/achievements [post]
func (h *AchievementHandler) Create(c *gin.Context) {
	var req CreateAchievementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	input := ports.CreateAchievementInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Category:    domain.AchievementCategory(req.Category),
		Tier:        domain.AchievementTier(req.Tier),
		IconURL:     req.IconURL,
		Criterion:   req.Criterion,
		Secret:      req.Secret,
		Active:      active,
		SortOrder:   req.SortOrder,
	}

	a, err := h.achievementSvc.Create(c.Request.Context(), input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, toAchievementResponse(&ports.AchievementWithProgress{Achievement: a}))
}

// @Summary      Update an achievement (admin)
// @Description  Partially update an admin-created achievement. Only supplied fields change. Requires `admin` or `superadmin` role. The `code` field is immutable. Attempting to modify a system (seeded) achievement returns 403 `ACHIEVEMENT_SYSTEM_LOCKED`.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Achievement ID" format(uuid)
// @Param        body body UpdateAchievementRequest true "Partial fields to update"
// @Success      200 {object} response.Response{data=AchievementResponse} "Updated achievement"
// @Failure      400 {object} response.Response "Invalid input (bad ID, invalid tier/category, malformed criterion)"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient role OR achievement is system-locked"
// @Failure      404 {object} response.Response "Achievement not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/achievements/{id} [patch]
func (h *AchievementHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid achievement ID", nil)
		return
	}

	var req UpdateAchievementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.UpdateAchievementInput{
		Name:        req.Name,
		Description: req.Description,
		IconURL:     req.IconURL,
		Criterion:   req.Criterion,
		Secret:      req.Secret,
		Active:      req.Active,
		SortOrder:   req.SortOrder,
	}
	if req.Category != nil {
		cat := domain.AchievementCategory(*req.Category)
		input.Category = &cat
	}
	if req.Tier != nil {
		tier := domain.AchievementTier(*req.Tier)
		input.Tier = &tier
	}

	a, err := h.achievementSvc.Update(c.Request.Context(), id, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, toAchievementResponse(&ports.AchievementWithProgress{Achievement: a}))
}

// @Summary      Delete an achievement (admin)
// @Description  Hard-delete an admin-created achievement. Cascades to `user_achievements`, so every user who had unlocked this badge will lose it. Requires `admin` or `superadmin` role. Attempting to delete a system (seeded) achievement returns 403 `ACHIEVEMENT_SYSTEM_LOCKED`.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Achievement ID" format(uuid)
// @Success      204 "Achievement deleted"
// @Failure      400 {object} response.Response "Invalid achievement ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient role OR achievement is system-locked"
// @Failure      404 {object} response.Response "Achievement not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/achievements/{id} [delete]
func (h *AchievementHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid achievement ID", nil)
		return
	}
	if err := h.achievementSvc.Delete(c.Request.Context(), id); err != nil {
		response.HandleError(c, err)
		return
	}
	c.Status(204)
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

