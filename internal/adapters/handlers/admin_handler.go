package handlers

import (
	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	ws "duskforge-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	adminService  ports.AdminService
	reportService ports.ReportService
	messageRepo   ports.MessageRepository
	hub           *ws.Hub
}

func NewAdminHandler(adminService ports.AdminService, reportService ports.ReportService, messageRepo ports.MessageRepository, hub *ws.Hub) *AdminHandler {
	return &AdminHandler{
		adminService:  adminService,
		reportService: reportService,
		messageRepo:   messageRepo,
		hub:           hub,
	}
}


type CreateReportRequest struct {
	Reason          domain.ReportReason `json:"reason" binding:"required,oneof=spam harassment spoiler inappropriate other" example:"spam"`
	Details         *string             `json:"details,omitempty" example:"This review contains spam links"`
	TargetUserID    *string             `json:"target_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	TargetReviewID  *string             `json:"target_review_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	TargetCommentID *string             `json:"target_comment_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
}

type ReportResponse struct {
	ID                           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReporterID                   string  `json:"reporter_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Reason                       string  `json:"reason" example:"spam"`
	Details                      *string `json:"details,omitempty" example:"This review contains spam links"`
	Status                       string  `json:"status" example:"pending"`
	TargetUserID                 *string `json:"target_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440002"`
	TargetUserUsername           *string `json:"target_user_username,omitempty" example:"johndoe"`
	TargetUserAvatarURL          *string `json:"target_user_avatar_url,omitempty" example:"https://cdn.example.com/avatars/johndoe.png"`
	TargetUserIsBanned           *bool   `json:"target_user_is_banned,omitempty" example:"false"`
	TargetReviewID               *string `json:"target_review_id,omitempty"`
	TargetReviewContent          *string `json:"target_review_content,omitempty" example:"Great movie!"`
	TargetReviewContainsSpoilers *bool   `json:"target_review_contains_spoilers,omitempty" example:"false"`
	TargetCommentID              *string `json:"target_comment_id,omitempty"`
	TargetCommentContent         *string `json:"target_comment_content,omitempty" example:"I totally agree with this review."`
	TargetCommentContainsSpoilers *bool  `json:"target_comment_contains_spoilers,omitempty" example:"false"`
	CreatedAt                    string  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	ResolvedAt                   *string `json:"resolved_at,omitempty" example:"2024-01-16T10:30:00Z"`
	ResolverID                   *string `json:"resolver_id,omitempty"`
}

type ResolveReportRequest struct {
	Status domain.ReportStatus `json:"status" binding:"required,oneof=pending resolved dismissed" example:"resolved"`
}

type SetUserRoleRequest struct {
	Role domain.UserRole `json:"role" binding:"required,oneof=user admin superadmin" example:"admin"`
}


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

func toReportResponseWithContext(rc *ports.ReportWithContext) ReportResponse {
	resp := toReportResponse(rc.Report)
	if rc.User != nil {
		username := rc.User.Username
		resp.TargetUserUsername = &username
		resp.TargetUserAvatarURL = rc.User.AvatarURL
		isBanned := rc.User.BannedAt != nil
		resp.TargetUserIsBanned = &isBanned
	}
	if rc.Review != nil {
		resp.TargetReviewContent = rc.Review.Content
		resp.TargetReviewContainsSpoilers = &rc.Review.ContainsSpoilers
	}
	if rc.Comment != nil {
		content := rc.Comment.Content
		resp.TargetCommentContent = &content
		resp.TargetCommentContainsSpoilers = &rc.Comment.ContainsSpoilers
	}
	return resp
}


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
// @Failure      403 {object} response.Response "Email not verified"
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


// @Summary      Ban a user
// @Description  Ban a user by their ID. Admins cannot ban other admins or super-admins. A "user.banned" WebSocket event is sent to all users who have a conversation with the banned user.
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

	if partnerIDs, err := h.messageRepo.GetConversationPartnerIDs(c.Request.Context(), targetID); err == nil {
		event := ws.Event{
			Type: ws.EventUserBanned,
			Data: ws.UserBannedPayload{
				UserID: targetID.String(),
			},
		}
		for _, partnerID := range partnerIDs {
			h.hub.SendToUser(partnerID, event)
		}
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


// @Summary      List reports
// @Description  List reports with optional filters. Filter by status and/or target user ID.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status" Enums(pending, resolved, dismissed)
// @Param        user_id query string false "Filter by target user ID" format(uuid)
// @Param        username query string false "Filter by target username"
// @Success      200 {object} response.Response{data=[]ReportResponse} "List of reports"
// @Failure      400 {object} response.Response "Invalid parameters"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Insufficient permissions"
// @Failure      404 {object} response.Response "User not found"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /admin/reports [get]
func (h *AdminHandler) ListReports(c *gin.Context) {
	filter := ports.ReportFilter{}

	if s := c.Query("status"); s != "" {
		status := domain.ReportStatus(s)
		filter.Status = &status
	}

	if u := c.Query("user_id"); u != "" {
		id, err := uuid.Parse(u)
		if err != nil {
			response.BadRequest(c, "Invalid user_id", nil)
			return
		}
		filter.TargetUserID = &id
	}

	if username := c.Query("username"); username != "" {
		filter.TargetUsername = &username
	}

	reports, err := h.reportService.List(c.Request.Context(), filter)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	result := make([]ReportResponse, len(reports))
	for i, r := range reports {
		result[i] = toReportResponseWithContext(r)
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

	response.Success(c, toReportResponseWithContext(report))
}

// @Summary      Update a report's status
// @Description  Update a report's status to pending, resolved, or dismissed. Reverting to pending clears the resolver and resolved_at fields.
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

	response.Success(c, toReportResponseWithContext(report))
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
