package handlers

import (
	"strconv"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type ActorHandler struct {
	actorService ports.ActorService
}

func NewActorHandler(actorService ports.ActorService) *ActorHandler {
	return &ActorHandler{actorService: actorService}
}

// @Summary      Search actors by name
// @Description  Search for actors by name. Use this endpoint when users are looking for a specific actor/person.
// @Tags         actors
// @Accept       json
// @Produce      json
// @Param        query query string true "Search query"
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 20)" default(20)
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Search results"
// @Failure      400 {object} response.Response "Query parameter is required"
// @Failure      502 {object} response.Response "External service error"
// @Router       /actors/search [get]
func (h *ActorHandler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		response.BadRequest(c, "Query parameter is required", nil)
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	language := middleware.GetLocale(c)

	result, err := h.actorService.Search(c.Request.Context(), ports.SearchActorsInput{
		Query:    query,
		Offset:   offset,
		Limit:    limit,
		Language: language,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessPaginated(c, result.Results, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total,
	})
}

// @Summary      Get actor details
// @Description  Get detailed information about an actor by their TMDB ID
// @Tags         actors
// @Accept       json
// @Produce      json
// @Param        id path int true "Actor ID"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.Response "Actor details"
// @Failure      400 {object} response.Response "Invalid actor ID"
// @Failure      404 {object} response.Response "Actor not found"
// @Failure      502 {object} response.Response "External service error"
// @Router       /actors/{id} [get]
func (h *ActorHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid actor ID", nil)
		return
	}

	language := middleware.GetLocale(c)

	actor, err := h.actorService.GetByID(c.Request.Context(), id, language)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, actor)
}

// @Summary      Get actor filmography
// @Description  Get the filmography (movies) of an actor by their TMDB ID. Includes both cast and crew credits.
// @Tags         actors
// @Accept       json
// @Produce      json
// @Param        id path int true "Actor ID"
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 20)" default(20)
// @Param        sort query string false "Sort field with direction prefix (+asc, -desc)" Enums(+release_date, -release_date, +popularity, -popularity, +rating, -rating, +name, -name) default(-release_date)
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Actor filmography"
// @Failure      400 {object} response.Response "Invalid actor ID"
// @Failure      404 {object} response.Response "Actor not found"
// @Failure      502 {object} response.Response "External service error"
// @Router       /actors/{id}/movies [get]
func (h *ActorHandler) GetFilmography(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid actor ID", nil)
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sortParam := c.DefaultQuery("sort", "-release_date")
	language := middleware.GetLocale(c)

	result, err := h.actorService.GetFilmography(c.Request.Context(), ports.GetActorFilmographyInput{
		ActorID:  id,
		Offset:   offset,
		Limit:    limit,
		Sort:     sortParam,
		Language: language,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessPaginated(c, result.Results, &response.Pagination{
		Offset: result.Offset,
		Limit:  result.Limit,
		Total:  result.Total,
	})
}
