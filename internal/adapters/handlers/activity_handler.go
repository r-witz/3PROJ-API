package handlers

import (
	"context"
	"strings"
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ActivityHandler struct {
	activityService ports.ActivityService
	movieService    ports.MovieService
	blockService    ports.BlockService
	banCache        ports.BanCache
}

func NewActivityHandler(activityService ports.ActivityService, movieService ports.MovieService, blockService ports.BlockService, banCache ports.BanCache) *ActivityHandler {
	return &ActivityHandler{activityService: activityService, movieService: movieService, blockService: blockService, banCache: banCache}
}

type ReviewSummary struct {
	ID      string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Rating  float64 `json:"rating" example:"4.5"`
	Content *string `json:"content,omitempty" example:"Great movie!"`
	TMDBID  int     `json:"tmdb_id" example:"550"`
}

type CollectionSummary struct {
	ID   string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name string `json:"name" example:"My Favorites"`
	Slug string `json:"slug" example:"my-favorites"`
}

type CommentSummary struct {
	ID      string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content string `json:"content" example:"I agree!"`
}

type ActivityResponse struct {
	ID         string                `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type       string                `json:"type" example:"review_created"`
	User       UserSummary           `json:"user"`
	Review     *ReviewSummary        `json:"review,omitempty"`
	Collection *CollectionSummary    `json:"collection,omitempty"`
	Comment    *CommentSummary       `json:"comment,omitempty"`
	Movie      *MovieSummaryResponse `json:"movie,omitempty"`
	TargetUser *UserSummary          `json:"target_user,omitempty"`
	CreatedAt  string                `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

func parseActivityTypes(c *gin.Context) []domain.ActivityType {
	raw := c.Query("types")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var types []domain.ActivityType
	for _, p := range parts {
		t := domain.ActivityType(strings.TrimSpace(p))
		if domain.ValidActivityTypes[t] {
			types = append(types, t)
		}
	}
	return types
}

// @Summary      Get user activities
// @Description  Get a user's activity feed. Returns 404 if the user is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the target user.
// @Tags         activities
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        types query string false "Comma-separated activity types to filter. Available types: review_created, review_updated, collection_created, collection_item_added, review_liked, comment_liked, user_followed, user_unfollowed, watchlist_item_added, comment_created"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse{data=[]ActivityResponse} "List of activities"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "User not found or banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/activities [get]
func (h *ActivityHandler) GetByUserID(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if IsBannedForCaller(c, h.banCache, userID) {
		response.HandleError(c, domain.ErrUserNotFound)
		return
	}

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != userID && !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), currentUserID, userID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	offset, limit := parsePagination(c)
	types := parseActivityTypes(c)

	items, total, err := h.activityService.GetByUserID(c.Request.Context(), userID, offset, limit, types)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	language := middleware.GetLocale(c)
	movies := h.fetchMoviesForActivities(c.Request.Context(), items, language)

	resp := make([]ActivityResponse, len(items))
	for i, item := range items {
		resp[i] = toActivityResponse(item, movies)
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Get following feed
// @Description  Get the authenticated user's feed of activities from users they follow. Activities from banned users are hidden for non-admin callers.
// @Tags         activities
// @Produce      json
// @Security     BearerAuth
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        types query string false "Comma-separated activity types to filter. Available types: review_created, review_updated, collection_created, collection_item_added, review_liked, comment_liked, user_followed, user_unfollowed, watchlist_item_added, comment_created"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse{data=[]ActivityResponse} "List of activities"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /feed [get]
func (h *ActivityHandler) GetFeed(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	offset, limit := parsePagination(c)
	types := parseActivityTypes(c)

	items, total, err := h.activityService.GetFeedForUser(c.Request.Context(), userID, offset, limit, types)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := GetHiddenUserIDs(c, h.blockService, h.banCache)

	filteredItems := make([]*ports.ActivityFeedItem, 0, len(items))
	for _, item := range items {
		if _, hidden := hiddenSet[item.Activity.UserID]; hidden {
			continue
		}
		filteredItems = append(filteredItems, item)
	}
	hiddenCount := len(items) - len(filteredItems)

	language := middleware.GetLocale(c)
	movies := h.fetchMoviesForActivities(c.Request.Context(), filteredItems, language)

	resp := make([]ActivityResponse, len(filteredItems))
	for i, item := range filteredItems {
		resp[i] = toActivityResponse(item, movies)
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total - hiddenCount,
	})
}

func (h *ActivityHandler) fetchMoviesForActivities(ctx context.Context, items []*ports.ActivityFeedItem, language string) map[int]*MovieSummaryResponse {
	seen := make(map[int]struct{})
	var tmdbIDs []int

	for _, item := range items {
		var id *int
		if item.Activity.TMDBID != nil {
			id = item.Activity.TMDBID
		} else if item.Review != nil {
			id = &item.Review.TMDBID
		}
		if id != nil {
			if _, ok := seen[*id]; !ok {
				seen[*id] = struct{}{}
				tmdbIDs = append(tmdbIDs, *id)
			}
		}
	}

	movies := make(map[int]*MovieSummaryResponse, len(tmdbIDs))
	for _, id := range tmdbIDs {
		details, err := h.movieService.GetByID(ctx, id, language)
		if err != nil {
			movies[id] = &MovieSummaryResponse{ID: id}
			continue
		}
		movies[id] = &MovieSummaryResponse{
			ID:              details.ID,
			Poster:          details.PosterPath,
			Name:            details.Title,
			Date:            details.ReleaseDate,
			TMDBRating:      details.Ratings.TMDB.Rating,
			DuskforgeRating: details.Ratings.Duskforge.Rating,
		}
	}
	return movies
}

func toActivityResponse(item *ports.ActivityFeedItem, movies map[int]*MovieSummaryResponse) ActivityResponse {
	resp := ActivityResponse{
		ID:        item.Activity.ID.String(),
		Type:      string(item.Activity.Type),
		CreatedAt: item.Activity.CreatedAt.Format(time.RFC3339),
	}

	if item.User != nil {
		resp.User = UserSummary{
			ID:        item.User.ID.String(),
			Username:  item.User.Username,
			AvatarURL: item.User.AvatarURL,
		}
	} else {
		resp.User = UserSummary{ID: item.Activity.UserID.String()}
	}

	if item.Review != nil {
		resp.Review = &ReviewSummary{
			ID:      item.Review.ID.String(),
			Rating:  item.Review.Rating,
			Content: item.Review.Content,
			TMDBID:  item.Review.TMDBID,
		}
		if m, ok := movies[item.Review.TMDBID]; ok {
			resp.Movie = m
		}
	}

	if item.Collection != nil {
		resp.Collection = &CollectionSummary{
			ID:   item.Collection.ID.String(),
			Name: item.Collection.Name,
			Slug: item.Collection.Slug,
		}
	}

	if item.Comment != nil {
		resp.Comment = &CommentSummary{
			ID:      item.Comment.ID.String(),
			Content: item.Comment.Content,
		}
	}

	if item.TargetUser != nil {
		resp.TargetUser = &UserSummary{
			ID:        item.TargetUser.ID.String(),
			Username:  item.TargetUser.Username,
			AvatarURL: item.TargetUser.AvatarURL,
		}
	}

	if item.Activity.TMDBID != nil && resp.Movie == nil {
		if m, ok := movies[*item.Activity.TMDBID]; ok {
			resp.Movie = m
		}
	}

	return resp
}
