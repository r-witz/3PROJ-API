package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	exportService ports.ExportService
}

func NewExportHandler(exportService ports.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

// @Summary      Export user data (GDPR)
// @Description  Export all personal data associated with the authenticated user as a JSON file download. Data is generated on-the-fly and not stored on disk.
// @Tags         export
// @Produce      application/json
// @Security     BearerAuth
// @Success      200 {file} file "JSON file containing all user data"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Router       /users/me/export [get]
func (h *ExportHandler) ExportUserData(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	export, err := h.exportService.ExportUserData(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c)
		return
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		response.InternalError(c)
		return
	}

	filename := fmt.Sprintf("duskforge-export-%s.json", time.Now().UTC().Format("2006-01-02"))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/json", data)
}
