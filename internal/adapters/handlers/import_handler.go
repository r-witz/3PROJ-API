package handlers

import (
	"bytes"
	"io"
	"net/http"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	importService ports.ImportService
}

func NewImportHandler(importService ports.ImportService) *ImportHandler {
	return &ImportHandler{importService: importService}
}

const maxImportSize = 10 << 20

type ImportLetterboxdResponse = ports.ImportProgress

// @Summary      Import Letterboxd data
// @Description  Start importing watched films, watchlist, ratings, and reviews from a Letterboxd export zip file. Processing runs in the background. Real-time progress is pushed via WebSocket (event: import.progress). Use GET /import/letterboxd/status as a fallback to poll progress.
// @Tags         import
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Letterboxd export zip file"
// @Success      202 {object} response.Response{data=ImportLetterboxdResponse} "Import started"
// @Failure      400 {object} response.Response "Invalid file or zip contains no Letterboxd data"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
// @Failure      403 {object} response.Response "Email not verified"
// @Router       /import/letterboxd [post]
func (h *ImportHandler) ImportLetterboxd(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Zip file is required", nil)
		return
	}
	defer file.Close()

	if header.Size > maxImportSize {
		response.HandleError(c, domain.ErrImportFileTooLarge)
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxImportSize+1))
	if err != nil {
		response.BadRequest(c, "Failed to read file", nil)
		return
	}
	if int64(len(data)) > maxImportSize {
		response.HandleError(c, domain.ErrImportFileTooLarge)
		return
	}

	reader := bytes.NewReader(data)
	progress, err := h.importService.StartImportLetterboxd(c.Request.Context(), userID, reader, int64(len(data)))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, response.Response{
		Success: true,
		Data:    progress,
	})
}

// @Summary      Get Letterboxd import status
// @Description  Get the current progress of a Letterboxd import. Returns the resolution progress during processing, and the full import result once completed.
// @Tags         import
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Response{data=ImportLetterboxdResponse} "Import progress"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "No import found"
// @Failure      403 {object} response.Response "Email not verified"
// @Router       /import/letterboxd/status [get]
func (h *ImportHandler) GetImportStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	progress := h.importService.GetImportStatus(userID)
	if progress == nil {
		response.Error(c, http.StatusNotFound, "IMPORT_NOT_FOUND", "No import found", nil)
		return
	}

	response.Success(c, progress)
}
