package handlers

import (
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CommentHandler struct {
	commentService ports.CommentService
	userService    ports.UserService
	blockService   ports.BlockService
	banCache       ports.BanCache
	notifService   ports.NotificationService
	hub            *ws.Hub
	reviewService  ports.ReviewService
}

func NewCommentHandler(commentService ports.CommentService, userService ports.UserService, blockService ports.BlockService, banCache ports.BanCache, notifService ports.NotificationService, hub *ws.Hub, reviewService ports.ReviewService) *CommentHandler {
	return &CommentHandler{commentService: commentService, userService: userService, blockService: blockService, banCache: banCache, notifService: notifService, hub: hub, reviewService: reviewService}
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
	ID               string      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	User             UserSummary `json:"user"`
	ReviewID         string      `json:"review_id" example:"770e8400-e29b-41d4-a716-446655440000"`
	Content          string      `json:"content" example:"Great review!"`
	ContainsSpoilers bool        `json:"contains_spoilers" example:"false"`
	LikeCount        int         `json:"like_count" example:"5"`
	LikedByUser      bool        `json:"liked_by_user" example:"false"`
	CreatedAt        string      `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        string      `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// @Summary      Create a comment
// @Description  Add a comment to a review. Returns 403 if there is a block between the authenticated user and the review author.
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Param        request body CreateCommentRequest true "Comment details"
// @Success      201 {object} response.Response{data=CommentResponse} "Comment created"
// @Failure      400 {object} response.Response "Invalid request body"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
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

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:    middleware.ActivityCreate,
		Type:      domain.ActivityTypeCommentCreated,
		UserID:    userID,
		CommentID: &comment.ID,
	})

	reviewMeta, err := h.reviewService.GetByID(c.Request.Context(), reviewID, nil)
	if err == nil && reviewMeta != nil {
		notif, _ := h.notifService.Notify(c.Request.Context(), ports.NotifyInput{
			UserID:    reviewMeta.Review.UserID,
			ActorID:   userID,
			Type:      domain.NotificationTypeNewComment,
			CommentID: &comment.ID,
		})
		if notif != nil {
			h.hub.SendToUser(reviewMeta.Review.UserID, ws.Event{
				Type: ws.EventNotificationNew,
				Data: notif,
			})
		}
	}

	user, _ := h.userService.GetByID(c.Request.Context(), userID)

	response.Created(c, toCommentResponse(comment, 0, false, user))
}

// @Summary      Get a comment by ID
// @Description  Get a single comment by its ID, including author details and like metadata. Returns 404 if the comment author is banned (non-admin callers). Returns 403 if there is a block between the authenticated user and the comment author.
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      200 {object} response.Response{data=CommentResponse} "Comment details"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      403 {object} response.Response "User blocked"
// @Failure      404 {object} response.Response "Comment not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /comments/{commentId} [get]
func (h *CommentHandler) GetByID(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	result, err := h.commentService.GetByIDWithMeta(c.Request.Context(), commentID, requestingUserID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	if requestingUserID == nil || result.Comment.UserID != *requestingUserID {
		if IsBannedForCaller(c, h.banCache, result.Comment.UserID) {
			response.HandleError(c, domain.ErrCommentNotFound)
			return
		}
	}

	if requestingUserID != nil && result.Comment.UserID != *requestingUserID && !IsCallerAdmin(c) {
		if blocked, err := h.blockService.IsBlocked(c.Request.Context(), result.Comment.UserID, *requestingUserID); err == nil && blocked {
			response.HandleError(c, domain.ErrUserBlocked)
			return
		}
	}

	response.Success(c, toCommentResponse(result.Comment, result.LikeCount, result.LikedByUser, result.User))
}

// @Summary      Get comments for a review
// @Description  List all comments on a review with pagination. Comments are sorted from oldest to newest. If authenticated, comments by blocked users are filtered out. Comments by banned users are hidden for non-admin callers.
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID" format(uuid)
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination" default(20)
// @Success      200 {object} response.PaginatedResponse{data=[]CommentResponse} "List of comments"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reviews/{reviewId}/comments [get]
func (h *CommentHandler) GetByReviewID(c *gin.Context) {
	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	offset, limit := parsePagination(c)

	var requestingUserID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		requestingUserID = &uid
	}

	comments, total, err := h.commentService.GetByReviewID(c.Request.Context(), reviewID, requestingUserID, offset, limit)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	hiddenSet := GetHiddenUserIDs(c, h.blockService, h.banCache)

	resp := make([]CommentResponse, 0, len(comments))
	hiddenCount := 0
	for _, cm := range comments {
		if _, hidden := hiddenSet[cm.Comment.UserID]; hidden {
			hiddenCount++
			continue
		}
		resp = append(resp, toCommentResponse(cm.Comment, cm.LikeCount, cm.LikedByUser, cm.User))
	}

	response.SuccessPaginated(c, resp, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total - hiddenCount,
	})
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

	result, err := h.commentService.Update(c.Request.Context(), commentID, userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toCommentResponse(result.Comment, result.LikeCount, result.LikedByUser, result.User))
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

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:    middleware.ActivityDelete,
		Type:      domain.ActivityTypeCommentCreated,
		UserID:    userID,
		CommentID: &commentID,
	})

	c.Status(204)
}

// @Summary      Like a comment
// @Description  Like a comment. Returns 403 if there is a block between the authenticated user and the comment author.
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      204 "Comment liked"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
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

	event := middleware.ActivityEvent{
		Action:    middleware.ActivityCreate,
		Type:      domain.ActivityTypeCommentLiked,
		UserID:    userID,
		CommentID: &commentID,
	}

	commentObj, err := h.commentService.GetByID(c.Request.Context(), commentID)
	if err == nil && commentObj != nil {
		authorID := commentObj.UserID
		event.TargetUserID = &authorID

		notif, _ := h.notifService.Notify(c.Request.Context(), ports.NotifyInput{
			UserID:    authorID,
			ActorID:   userID,
			Type:      domain.NotificationTypeLikeComment,
			CommentID: &commentID,
		})
		if notif != nil {
			h.hub.SendToUser(authorID, ws.Event{
				Type: ws.EventNotificationNew,
				Data: notif,
			})
		}
	}

	middleware.QueueActivity(c, event)

	c.Status(204)
}

// @Summary      Unlike a comment
// @Description  Remove a like from a comment. Returns 403 if there is a block between the authenticated user and the comment author.
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID" format(uuid)
// @Success      204 "Comment unliked"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "User blocked"
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

	middleware.QueueActivity(c, middleware.ActivityEvent{
		Action:    middleware.ActivityDelete,
		Type:      domain.ActivityTypeCommentLiked,
		UserID:    userID,
		CommentID: &commentID,
	})

	c.Status(204)
}

func toCommentResponse(comment *domain.Comment, likeCount int, likedByUser bool, user *domain.User) CommentResponse {
	resp := CommentResponse{
		ID:               comment.ID.String(),
		ReviewID:         comment.ReviewID.String(),
		Content:          comment.Content,
		ContainsSpoilers: comment.ContainsSpoilers,
		LikeCount:        likeCount,
		LikedByUser:      likedByUser,
		CreatedAt:        comment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        comment.UpdatedAt.Format(time.RFC3339),
	}

	if user != nil {
		resp.User = UserSummary{
			ID:        user.ID.String(),
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
		}
	} else {
		resp.User = UserSummary{
			ID: comment.UserID.String(),
		}
	}

	return resp
}
