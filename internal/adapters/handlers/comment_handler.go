package handlers

import (
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CommentHandler struct {
	commentService ports.CommentService
}

func NewCommentHandler(commentService ports.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

type CreateCommentRequest struct {
	Content          string `json:"content" binding:"required" example:"Great review!"`
	ContainsSpoilers bool   `json:"contains_spoilers" example:"false"`
}

type UpdateCommentRequest struct {
	Content          *string `json:"content" example:"Updated comment"`
	ContainsSpoilers *bool   `json:"contains_spoilers" example:"true"`
}

type CommentResponse struct {
	ID               string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID           string `json:"user_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	ReviewID         string `json:"review_id" example:"770e8400-e29b-41d4-a716-446655440000"`
	Content          string `json:"content" example:"Great review!"`
	ContainsSpoilers bool   `json:"contains_spoilers" example:"false"`
	LikeCount        int    `json:"like_count" example:"5"`
	LikedByUser      bool   `json:"liked_by_user" example:"false"`
	CreatedAt        string `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        string `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// @Summary      Create a comment
// @Description  Add a comment to a review
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Param        request body CreateCommentRequest true "Comment details"
// @Success      201 {object} response.Response{data=CommentResponse} "Comment created"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
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

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.CreateCommentInput{
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}

	comment, err := h.commentService.Create(c.Request.Context(), reviewID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, toCommentResponse(comment, 0, false))
}

// @Summary      Get comments for a review
// @Description  List all comments on a review
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Success      200 {object} response.Response{data=[]CommentResponse} "List of comments"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId}/comments [get]
func (h *CommentHandler) GetByReviewID(c *gin.Context) {
	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	comments, err := h.commentService.GetByReviewID(c.Request.Context(), reviewID, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resp := make([]CommentResponse, len(comments))
	for i, cm := range comments {
		resp[i] = toCommentResponse(cm.Comment, cm.LikeCount, cm.LikedByUser)
	}

	response.Success(c, resp)
}

// @Summary      Update a comment
// @Description  Update your own comment
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Param        request body UpdateCommentRequest true "Fields to update"
// @Success      200 {object} response.Response{data=CommentResponse} "Updated comment"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Comment not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /comments/{commentId} [patch]
func (h *CommentHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.UpdateCommentInput{
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}

	comment, err := h.commentService.Update(c.Request.Context(), commentID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCommentResponse(comment, 0, false))
}

// @Summary      Delete a comment
// @Description  Delete your own comment
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      204 "Comment deleted"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden"
// @Failure      404 {object} response.Response "Comment not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /comments/{commentId} [delete]
func (h *CommentHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	if err := h.commentService.Delete(c.Request.Context(), commentID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Like a comment
// @Description  Like a comment
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      204 "Comment liked"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Comment not found"
// @Failure      409 {object} response.Response "Already liked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /comments/{commentId}/like [post]
func (h *CommentHandler) Like(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	if err := h.commentService.Like(c.Request.Context(), commentID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Unlike a comment
// @Description  Remove a like from a comment
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      204 "Comment unliked"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Comment not found or not liked"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /comments/{commentId}/like [delete]
func (h *CommentHandler) Unlike(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	if err := h.commentService.Unlike(c.Request.Context(), commentID, userID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

func toCommentResponse(comment *domain.Comment, likeCount int, likedByUser bool) CommentResponse {
	return CommentResponse{
		ID:               comment.ID.String(),
		UserID:           comment.UserID.String(),
		ReviewID:         comment.ReviewID.String(),
		Content:          comment.Content,
		ContainsSpoilers: comment.ContainsSpoilers,
		LikeCount:        likeCount,
		LikedByUser:      likedByUser,
		CreatedAt:        comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        comment.UpdatedAt.Format(time.RFC3339),
	}
}
