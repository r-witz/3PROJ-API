package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	adminService  ports.AdminService
	reportService ports.ReportService
}

func NewAdminHandler(adminService ports.AdminService, reportService ports.ReportService) *AdminHandler {
	return &AdminHandler{
		adminService:  adminService,
		reportService: reportService,
	}
}

// --- Request / Response DTOs ---

type CreateReportRequest struct {
	Reason          domain.ReportReason `json:"reason" binding:"required,oneof=spam harassment spoiler inappropriate other" example:"spam"`
	Details         *string             `json:"details,omitempty" example:"This review contains spam links"`
	TargetUserID    *string             `json:"target_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	TargetReviewID  *string             `json:"target_review_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	TargetCommentID *string             `json:"target_comment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
}

type ReportResponse struct {
	ID              string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReporterID      string  `json:"reporter_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Reason          string  `json:"reason" example:"spam"`
	Details         *string `json:"details,omitempty" example:"This review contains spam links"`
	Status          string  `json:"status" example:"pending"`
	TargetUserID    *string `json:"target_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	TargetReviewID  *string `json:"target_review_id,omitempty"`
	TargetCommentID *string `json:"target_comment_id,omitempty"`
	CreatedAt       string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	ResolvedAt      *string `json:"resolved_at,omitempty" example:"2024-01-16T10:30:00Z"`
	ResolverID      *string `json:"resolver_id,omitempty"`
}

type ResolveReportRequest struct {
	Status domain.ReportStatus `json:"status" binding:"required,oneof=resolved dismissed" example:"resolved"`
}

type SetUserRoleRequest struct {
	Role domain.UserRole `json:"role" binding:"required,oneof=user admin superadmin" example:"admin"`
}

type AdminUserResponse struct {
	ID        string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string  `json:"email" example:"user@example.com"`
	Username  string  `json:"username" example:"johndoe"`
	AvatarURL *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Role      string  `json:"role" example:"user"`
	CreatedAt string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	BannedAt  *string `json:"banned_at,omitempty" example:"2024-02-01T10:30:00Z"`
}

// --- Helpers ---

func toReportResponse(r *domain.Report) ReportResponse {
	resp := ReportResponse{
		ID:         r.ID.String(),
		ReporterID: r.ReporterID.String(),
		Reason:     string(r.Reason),
		Details:    r.Details,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.TargetUserID != nil {
		s := r.TargetUserID.String()
		resp.TargetUserID = &s
	}
	if r.TargetReviewID != nil {
		s := r.TargetReviewID.String()
		resp.TargetReviewID = &s
	}
	if r.TargetCommentID != nil {
		s := r.TargetCommentID.String()
		resp.TargetCommentID = &s
	}
	if r.ResolvedAt != nil {
		s := r.ResolvedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ResolvedAt = &s
	}
	if r.ResolverID != nil {
		s := r.ResolverID.String()
		resp.ResolverID = &s
	}
	return resp
}

func toAdminUserResponse(u *domain.User) AdminUserResponse {
	resp := AdminUserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if u.BannedAt != nil {
		s := u.BannedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.BannedAt = &s
	}
	return resp
}

// --- User-facing: Submit Report ---

// @Summary      Submit a report
// @Description  Report a user, review, or comment for moderation. Exactly one target must be specified.
// @Tags         reports
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body CreateReportRequest true "Report details"
// @Success      201 {object} response.Response{data=ReportResponse} "Report created"
// @Failure      400 {object} response.Response "Invalid input"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Target not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /reports [post]
func (h *AdminHandler) SubmitReport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	input := ports.CreateReportInput{
		Reason:  req.Reason,
		Details: req.Details,
	}

	if req.TargetUserID != nil {
		id, err := uuid.Parse(*req.TargetUserID)
		if err != nil {
			response.BadRequest(c, "Invalid target_user_id", nil)
			return
		}
		input.TargetUserID = &id
	}
	if req.TargetReviewID != nil {
		id, err := uuid.Parse(*req.TargetReviewID)
		if err != nil {
			response.BadRequest(c, "Invalid target_review_id", nil)
			return
		}
		input.TargetReviewID = &id
	}
	if req.TargetCommentID != nil {
		id, err := uuid.Parse(*req.TargetCommentID)
		if err != nil {
			response.BadRequest(c, "Invalid target_comment_id", nil)
			return
		}
		input.TargetCommentID = &id
	}

	report, err := h.reportService.Create(c.Request.Context(), userID, input)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, toReportResponse(report))
}

// --- Admin: User Management ---

// @Summary      List users
// @Description  List all users with pagination. Optionally filter to only banned users.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        offset query int false "Offset for pagination" default(0)
// @Param        limit query int false "Limit for pagination (max 100)" default(20)
// @Param        banned query bool false "Filter to only banned users" default(false)
// @Success      200 {object} response.PaginatedResponse{data=[]AdminUserResponse} "List of users"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	offset, limit := parsePagination(c)
	bannedOnly := c.Query("banned") == "true"

	users, total, err := h.adminService.GetUsers(c.Request.Context(), offset, limit, bannedOnly)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	result := make([]AdminUserResponse, len(users))
	for i, u := range users {
		result[i] = toAdminUserResponse(u)
	}

	response.SuccessPaginated(c, result, &response.Pagination{
		Offset: offset,
		Limit:  limit,
		Total:  total,
	})
}

// @Summary      Ban a user
// @Description  Ban a user by their ID. Admins cannot ban other admins or super-admins.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to ban" format(uuid)
// @Success      204 "User banned successfully"
// @Failure      400 {object} response.Response "Invalid user ID or cannot ban self"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Cannot ban admin or insufficient permissions"
// @Failure      404 {object} response.Response "User not found"
// @Failure      409 {object} response.Response "User already banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/users/{userId}/ban [post]
func (h *AdminHandler) BanUser(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.adminService.BanUser(c.Request.Context(), adminID, targetID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Unban a user
// @Description  Remove a ban from a user
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID to unban" format(uuid)
// @Success      204 "User unbanned successfully"
// @Failure      400 {object} response.Response "Invalid user ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "User not found"
// @Failure      409 {object} response.Response "User is not banned"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/users/{userId}/ban [delete]
func (h *AdminHandler) UnbanUser(c *gin.Context) {
	adminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	if err := h.adminService.UnbanUser(c.Request.Context(), adminID, targetID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// --- Admin: Content Moderation ---

// @Summary      Delete a review (admin)
// @Description  Delete any review by its ID, regardless of ownership
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        reviewId path string true "Review ID to delete" format(uuid)
// @Success      204 "Review deleted successfully"
// @Failure      400 {object} response.Response "Invalid review ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "Review not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reviews/{reviewId} [delete]
func (h *AdminHandler) DeleteReview(c *gin.Context) {
	reviewID, err := uuid.Parse(c.Param("reviewId"))
	if err != nil {
		response.BadRequest(c, "Invalid review ID", nil)
		return
	}

	if err := h.adminService.DeleteReview(c.Request.Context(), reviewID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// @Summary      Delete a comment (admin)
// @Description  Delete any comment by its ID, regardless of ownership
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        commentId path string true "Comment ID to delete" format(uuid)
// @Success      204 "Comment deleted successfully"
// @Failure      400 {object} response.Response "Invalid comment ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "Comment not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/comments/{commentId} [delete]
func (h *AdminHandler) DeleteComment(c *gin.Context) {
	commentID, err := uuid.Parse(c.Param("commentId"))
	if err != nil {
		response.BadRequest(c, "Invalid comment ID", nil)
		return
	}

	if err := h.adminService.DeleteComment(c.Request.Context(), commentID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// --- Admin: Report Management ---

// @Summary      List reports
// @Description  List reports filtered by status
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status" Enums(pending, resolved, dismissed) default(pending)
// @Success      200 {object} response.Response{data=[]ReportResponse} "List of reports"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reports [get]
func (h *AdminHandler) ListReports(c *gin.Context) {
	status := domain.ReportStatus(c.DefaultQuery("status", string(domain.ReportStatusPending)))

	reports, err := h.reportService.GetByStatus(c.Request.Context(), status)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	result := make([]ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = toReportResponse(r)
	}

	response.Success(c, result)
}

// @Summary      Get a report
// @Description  Get a single report by its ID
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        reportId path string true "Report ID" format(uuid)
// @Success      200 {object} response.Response{data=ReportResponse} "Report details"
// @Failure      400 {object} response.Response "Invalid report ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "Report not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reports/{reportId} [get]
func (h *AdminHandler) GetReport(c *gin.Context) {
	reportID, err := uuid.Parse(c.Param("reportId"))
	if err != nil {
		response.BadRequest(c, "Invalid report ID", nil)
		return
	}

	report, err := h.reportService.GetByID(c.Request.Context(), reportID)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toReportResponse(report))
}

// @Summary      Resolve or dismiss a report
// @Description  Update a report's status to resolved or dismissed
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        reportId path string true "Report ID" format(uuid)
// @Param        body body ResolveReportRequest true "New status"
// @Success      200 {object} response.Response{data=ReportResponse} "Report updated"
// @Failure      400 {object} response.Response "Invalid input"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "Report not found"
// @Failure      409 {object} response.Response "Report already resolved"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reports/{reportId} [patch]
func (h *AdminHandler) ResolveReport(c *gin.Context) {
	resolverID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	reportID, err := uuid.Parse(c.Param("reportId"))
	if err != nil {
		response.BadRequest(c, "Invalid report ID", nil)
		return
	}

	var req ResolveReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	report, err := h.reportService.Resolve(c.Request.Context(), reportID, resolverID, ports.ResolveReportInput{
		Status: req.Status,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, toReportResponse(report))
}

// @Summary      Delete a report
// @Description  Permanently delete a report
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        reportId path string true "Report ID" format(uuid)
// @Success      204 "Report deleted successfully"
// @Failure      400 {object} response.Response "Invalid report ID"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "Report not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reports/{reportId} [delete]
func (h *AdminHandler) DeleteReport(c *gin.Context) {
	reportID, err := uuid.Parse(c.Param("reportId"))
	if err != nil {
		response.BadRequest(c, "Invalid report ID", nil)
		return
	}

	if err := h.reportService.Delete(c.Request.Context(), reportID); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}

// --- Super-Admin: Role Management ---

// @Summary      Set user role
// @Description  Change a user's role. Only super-admins can use this endpoint.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID" format(uuid)
// @Param        body body SetUserRoleRequest true "New role"
// @Success      204 "Role updated successfully"
// @Failure      400 {object} response.Response "Invalid input or cannot change own role"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/users/{userId}/role [patch]
func (h *AdminHandler) SetUserRole(c *gin.Context) {
	superAdminID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	targetID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", nil)
		return
	}

	var req SetUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.adminService.SetUserRole(c.Request.Context(), superAdminID, targetID, req.Role); err != nil {
		response.HandleError(c, err)
		return
	}

	c.Status(204)
}
