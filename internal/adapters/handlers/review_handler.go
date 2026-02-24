package handlers

import (
	"context"
	"strconv"
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReviewHandler struct {
	reviewService ports.ReviewService
	movieService  ports.MovieService
	userService   ports.UserService
	blockService  ports.BlockService
}

func NewReviewHandler(reviewService ports.ReviewService, movieService ports.MovieService, userService ports.UserService, blockService ports.BlockService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService, movieService: movieService, userService: userService, blockService: blockService}
}

type CreateReviewRequest struct {
	Rating           float64 `json:"rating" binding:"required" example:"4.5"`
	Content          *string `json:"content" example:"Great movie!"`
	ContainsSpoilers bool    `json:"contains_spoilers" example:"false"`
}

type UpdateReviewRequest struct {
	Rating           *float64 `json:"rating" example:"4.0"`
	Content          *string  `json:"content" example:"Updated review"`
	ContainsSpoilers *bool    `json:"contains_spoilers" example:"true"`
}

type UserSummary struct {
	ID        string  `json:"id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
}

type MovieSummaryResponse struct {
	ID              int      `json:"id" example:"550"`
	Poster          *string  `json:"poster,omitempty" example:"/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"`
	Name            string   `json:"name" example:"Fight Club"`
	Date            string   `json:"date" example:"1999-10-15"`
	TMDBRating      *float64 `json:"tmdb_rating,omitempty" example:"4.3"`
	DuskforgeRating *float64 `json:"duskforge_rating,omitempty" example:"4.5"`
}

type ReviewResponse struct {
	ID               string               `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	User             UserSummary          `json:"user"`
	Movie            MovieSummaryResponse `json:"movie"`
	Rating           float64              `json:"rating" example:"4.5"`
	Content          *string              `json:"content,omitempty" example:"Great movie!"`
	ContainsSpoilers bool                 `json:"contains_spoilers" example:"false"`
	LikeCount        int                  `json:"like_count" example:"12"`
	CommentCount     int                  `json:"comment_count" example:"5"`
	LikedByUser      bool                 `json:"liked_by_user" example:"false"`
	CreatedAt        string               `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        string               `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// @Summary      Create a review
// @Description  Create a review for a movie by TMDB ID
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "TMDB movie ID"
// @Param        request body CreateReviewRequest true "Review details"
// @Success      201 {object} response.Response{data=ReviewResponse} "Review created"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      409 {object} response.Response "Review already exists for this movie"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /movies/{id}/reviews [post]
func (h *ReviewHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	tmdbID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid movie ID", nil)
		return
	}

	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.CreateReviewInput{
		Rating:           req.Rating,
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}

	review, err := h.reviewService.Create(c.Request.Context(), userID, tmdbID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	user, _ := h.userService.GetByID(c.Request.Context(), userID)
	language := middleware.GetLocale(c)
	movie := h.fetchMovieSummary(c.Request.Context(), tmdbID, language)

	response.Created(c, toReviewResponse(review, 0, 0, false, user, movie))
}

// @Summary      Get reviews for a movie
// @Description  List all reviews for a movie by TMDB ID with pagination and sorting. Only reviews with content are returned. If authenticated, the logged-in user's own review is excluded and reviews by blocked users are filtered out.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "TMDB movie ID"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        sort query string false "Sort field with direction prefix (+asc, -desc)" Enums(+likes, -likes, +rating, -rating, +created_at, -created_at) default(-likes)
// @Success      200 {object} response.PaginatedResponse{data=[]ReviewResponse} "List of reviews"
// @Failure      400 {object} response.Response "Invalid movie ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /movies/{id}/reviews [get]
func (h *ReviewHandler) GetByMovieID(c *gin.Context) {
	tmdbID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid movie ID", nil)
		return
	}

	offset, limit := parsePagination(c)
	sort := parseReviewSort(c.DefaultQuery("sort", "-likes"))

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	reviews, total, err := h.reviewService.GetByTMDBID(c.Request.Context(), tmdbID, requestingUserID, offset, limit, sort)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := h.getHiddenUserIDs(c)

	language := middleware.GetLocale(c)
	movie := h.fetchMovieSummary(c.Request.Context(), tmdbID, language)

	resp := make([]ReviewResponse, 0, len(reviews))
	for _, r := range reviews {
		if _, hidden := hiddenSet[r.Review.UserID]; hidden {
			continue
		}
		resp = append(resp, toReviewResponse(r.Review, r.LikeCount, r.CommentCount, r.LikedByUser, r.User, movie))
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Get a review by ID
// @Description  Get a single review by its ID. Returns 403 if there is a block between the authenticated user and the review author.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      200 {object} response.Response{data=ReviewResponse} "Review details"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId} [get]
func (h *ReviewHandler) GetByID(c *gin.Context) {
	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	review, err := h.reviewService.GetByID(c.Request.Context(), reviewID, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if requestingUserID != nil && review.Review.UserID != *requestingUserID {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), review.Review.UserID, *requestingUserID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	language := middleware.GetLocale(c)
	movie := h.fetchMovieSummary(c.Request.Context(), review.Review.TMDBID, language)

	response.Success(c, toReviewResponse(review.Review, review.LikeCount, review.CommentCount, review.LikedByUser, review.User, movie))
}

// @Summary      Get reviews by user
// @Description  List all reviews by a specific user with pagination and sorting. Returns 403 if there is a block between the authenticated user and the target user.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        tmdb_id query int false "Filter by TMDB movie ID"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        sort query string false "Sort field with direction prefix (+asc, -desc)" Enums(+likes, -likes, +rating, -rating, +created_at, -created_at) default(-created_at)
// @Success      200 {object} response.PaginatedResponse{data=[]ReviewResponse} "List of reviews"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/reviews [get]
func (h *ReviewHandler) GetByUserID(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if currentUserID, ok := middleware.GetUserID(c); ok && currentUserID != userID {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), currentUserID, userID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	var tmdbID *int
	if v := c.Query("tmdb_id"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			response.BadRequest(c, "Invalid tmdb_id", nil)
			return
		}
		tmdbID = &parsed
	}

	offset, limit := parsePagination(c)
	sort := parseReviewSort(c.DefaultQuery("sort", "-created_at"))

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	reviews, total, err := h.reviewService.GetByUserID(c.Request.Context(), userID, tmdbID, requestingUserID, offset, limit, sort)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	language := middleware.GetLocale(c)
	movies := h.fetchMovieSummaries(c.Request.Context(), reviews, language)

	resp := make([]ReviewResponse, len(reviews))
	for i, r := range reviews {
		resp[i] = toReviewResponse(r.Review, r.LikeCount, r.CommentCount, r.LikedByUser, r.User, movies[r.Review.TMDBID])
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Update a review
// @Description  Update your own review
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Param        request body UpdateReviewRequest true "Fields to update"
// @Success      200 {object} response.Response{data=ReviewResponse} "Updated review"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId} [patch]
func (h *ReviewHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.UpdateReviewInput{
		Rating:           req.Rating,
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}

	result, err := h.reviewService.Update(c.Request.Context(), reviewID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	language := middleware.GetLocale(c)
	movie := h.fetchMovieSummary(c.Request.Context(), result.Review.TMDBID, language)

	response.Success(c, toReviewResponse(result.Review, result.LikeCount, result.CommentCount, result.LikedByUser, result.User, movie))
}

// @Summary      Delete a review
// @Description  Delete your own review
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      204 "Review deleted"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId} [delete]
func (h *ReviewHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	if err := h.reviewService.Delete(c.Request.Context(), reviewID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Like a review
// @Description  Like a review. Returns 403 if there is a block between the authenticated user and the review author.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      204 "Review liked"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      409 {object} response.Response "Already liked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId}/like [post]
func (h *ReviewHandler) Like(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	if err := h.reviewService.Like(c.Request.Context(), reviewID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Unlike a review
// @Description  Remove a like from a review. Returns 403 if there is a block between the authenticated user and the review author.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      204 "Review unliked"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Review not found or not liked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId}/like [delete]
func (h *ReviewHandler) Unlike(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	if err := h.reviewService.Unlike(c.Request.Context(), reviewID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

func (h *ReviewHandler) getHiddenUserIDs(c *gin.Context) map[uuid.UUID]struct{} {
	hiddenSet := make(map[uuid.UUID]struct{})
	currentUserID, ok := middleware.GetUserID(c)
	if !ok {
		return hiddenSet
	}
	ctx := c.Request.Context()
	if blockerIDs, err := h.blockService.GetBlockerIDs(ctx, currentUserID); err == nil {
		for _, id := range blockerIDs {
			hiddenSet[id] = struct{}{}
		}
	}
	if blockedIDs, err := h.blockService.GetBlockedIDs(ctx, currentUserID); err == nil {
		for _, id := range blockedIDs {
			hiddenSet[id] = struct{}{}
		}
	}
	return hiddenSet
}

func (h *ReviewHandler) fetchMovieSummary(ctx context.Context, tmdbID int, language string) *MovieSummaryResponse {
	details, err := h.movieService.GetByID(ctx, tmdbID, language)
	if err != nil {
		return &MovieSummaryResponse{ID: tmdbID}
	}
	return &MovieSummaryResponse{
		ID:              details.ID,
		Poster:          details.PosterPath,
		Name:            details.Title,
		Date:            details.ReleaseDate,
		TMDBRating:      details.Ratings.TMDB.Rating,
		DuskforgeRating: details.Ratings.Duskforge.Rating,
	}
}

func (h *ReviewHandler) fetchMovieSummaries(ctx context.Context, reviews []*ports.ReviewWithMeta, language string) map[int]*MovieSummaryResponse {
	seen := make(map[int]struct{})
	var tmdbIDs []int
	for _, r := range reviews {
		if _, ok := seen[r.Review.TMDBID]; !ok {
			seen[r.Review.TMDBID] = struct{}{}
			tmdbIDs = append(tmdbIDs, r.Review.TMDBID)
		}
	}

	movies := make(map[int]*MovieSummaryResponse, len(tmdbIDs))
	for _, id := range tmdbIDs {
		movies[id] = h.fetchMovieSummary(ctx, id, language)
	}
	return movies
}

func toReviewResponse(review *domain.Review, likeCount int, commentCount int, likedByUser bool, user *domain.User, movie *MovieSummaryResponse) ReviewResponse {
	resp := ReviewResponse{
		ID:               review.ID.String(),
		Rating:           review.Rating,
		Content:          review.Content,
		ContainsSpoilers: review.ContainsSpoilers,
		LikeCount:        likeCount,
		CommentCount:     commentCount,
		LikedByUser:      likedByUser,
		CreatedAt:        review.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        review.UpdatedAt.Format(time.RFC3339),
	}

	if movie != nil {
		resp.Movie = *movie
	} else {
		resp.Movie = MovieSummaryResponse{ID: review.TMDBID}
	}

	if user != nil {
		resp.User = UserSummary{
			ID:        user.ID.String(),
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		}
	} else {
		resp.User = UserSummary{
			ID: review.UserID.String(),
		}
	}

	return resp
}

func parsePagination(c *gin.Context) (offset, limit int) {
	offset = 0
	limit = 20

	if v := c.Query("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	return offset, limit
}

func parseReviewSort(s string) ports.ReviewSort {
	if s == "" {
		return ports.ReviewSort{Field: ports.ReviewSortByCreatedAt, Asc: false}
	}

	asc := false
	field := s

	switch s[0] {
	case '+':
		asc = true
		field = s[1:]
	case '-':
		asc = false
		field = s[1:]
	}

	switch field {
	case "likes":
		return ports.ReviewSort{Field: ports.ReviewSortByLikes, Asc: asc}
	case "rating":
		return ports.ReviewSort{Field: ports.ReviewSortByRating, Asc: asc}
	case "created_at":
		return ports.ReviewSort{Field: ports.ReviewSortByCreatedAt, Asc: asc}
	default:
		return ports.ReviewSort{Field: ports.ReviewSortByCreatedAt, Asc: false}
	}
}
