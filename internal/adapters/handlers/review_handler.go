package handlers

import (
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
	userService   ports.UserService
}

func NewReviewHandler(reviewService ports.ReviewService, userService ports.UserService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService, userService: userService}
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

type ReviewResponse struct {
	ID               string      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	User             UserSummary `json:"user"`
	TMDBID           int         `json:"tmdb_id" example:"550"`
	Rating           float64     `json:"rating" example:"4.5"`
	Content          *string     `json:"content,omitempty" example:"Great movie!"`
	ContainsSpoilers bool        `json:"contains_spoilers" example:"false"`
	LikeCount        int         `json:"like_count" example:"12"`
	LikedByUser      bool        `json:"liked_by_user" example:"false"`
	CreatedAt        string      `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        string      `json:"updated_at" example:"2024-01-15T10:30:00Z"`
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

	response.Created(c, toReviewResponse(review, 0, false, user))
}

// @Summary      Get reviews for a movie
// @Description  List all reviews for a movie by TMDB ID with pagination. Supports sorting by likes (default) or chronological order.
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "TMDB movie ID"
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        sort query string false "Sort order: 'likes' (default) or 'recent'" default(likes)
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
	sort := c.DefaultQuery("sort", "likes")

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	reviews, total, err := h.reviewService.GetByTMDBID(c.Request.Context(), tmdbID, requestingUserID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]ReviewResponse, len(reviews))
	for i, r := range reviews {
		resp[i] = toReviewResponse(r.Review, r.LikeCount, r.LikedByUser, r.User)
	}

	if sort == "likes" {
		sortReviewsByLikes(resp)
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Get a review by ID
// @Description  Get a single review by its ID
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      200 {object} response.Response{data=ReviewResponse} "Review details"
// @Failure      400 {object} response.Response "Invalid review ID"
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

	response.Success(c, toReviewResponse(review.Review, review.LikeCount, review.LikedByUser, review.User))
}

// @Summary      Get reviews by user
// @Description  List all reviews by a specific user with pagination. Supports sorting by likes or chronological order (default).
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Param        sort query string false "Sort order: 'recent' (default) or 'likes'" default(recent)
// @Success      200 {object} response.PaginatedResponse{data=[]ReviewResponse} "List of reviews"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/{userId}/reviews [get]
func (h *ReviewHandler) GetByUserID(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	offset, limit := parsePagination(c)
	sort := c.DefaultQuery("sort", "recent")

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	reviews, total, err := h.reviewService.GetByUserID(c.Request.Context(), userID, requestingUserID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]ReviewResponse, len(reviews))
	for i, r := range reviews {
		resp[i] = toReviewResponse(r.Review, r.LikeCount, r.LikedByUser, r.User)
	}

	if sort == "likes" {
		sortReviewsByLikes(resp)
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

	review, err := h.reviewService.Update(c.Request.Context(), reviewID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	user, _ := h.userService.GetByID(c.Request.Context(), userID)

	response.Success(c, toReviewResponse(review, 0, false, user))
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
// @Description  Like a review
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      204 "Review liked"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
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
// @Description  Remove a like from a review
// @Tags         reviews
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      204 "Review unliked"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
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

func toReviewResponse(review *domain.Review, likeCount int, likedByUser bool, user *domain.User) ReviewResponse {
	resp := ReviewResponse{
		ID:               review.ID.String(),
		TMDBID:           review.TMDBID,
		Rating:           review.Rating,
		Content:          review.Content,
		ContainsSpoilers: review.ContainsSpoilers,
		LikeCount:        likeCount,
		LikedByUser:      likedByUser,
		CreatedAt:        review.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        review.UpdatedAt.Format(time.RFC3339),
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

func sortReviewsByLikes(reviews []ReviewResponse) {
	for i := 1; i < len(reviews); i++ {
		for j := i; j > 0 && reviews[j].LikeCount > reviews[j-1].LikeCount; j-- {
			reviews[j], reviews[j-1] = reviews[j-1], reviews[j]
		}
	}
}
