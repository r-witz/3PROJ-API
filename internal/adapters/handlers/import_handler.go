package handlers

import (
	"bytes"
	"io"

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

const maxImportSize = 10 << 20 // 10MB

// ImportLetterboxdResponse represents the import result returned to the client.
type ImportLetterboxdResponse struct {
	Watched   ports.ImportSectionResult `json:"watched"`
	Watchlist ports.ImportSectionResult `json:"watchlist"`
	Ratings   ports.ImportSectionResult `json:"ratings"`
	Reviews   ports.ImportSectionResult `json:"reviews"`
	Failed    []ports.ImportFailure     `json:"failed"`
}

// ImportLetterboxd godoc
// @Summary      Import Letterboxd data
// @Description  Import watched films, watchlist, ratings, and reviews from a Letterboxd account export zip file. Films are resolved to TMDB IDs via search. Existing data is never overwritten — duplicates are skipped.
// @Tags         import
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Letterboxd export zip file"
// @Success      200 {object} response.Response{data=ImportLetterboxdResponse} "Import results"
// @Failure      400 {object} response.Response "Invalid file or file too large"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      500 {object} response.Response "Internal server error"
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
	result, err := h.importService.ImportLetterboxd(c.Request.Context(), userID, reader, int64(len(data)))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, result)
}
