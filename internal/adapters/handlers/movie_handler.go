package handlers

import (
	"strconv"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	portservices "duskforge-api/internal/core/ports/services"

	"github.com/gin-gonic/gin"
)

type MovieHandler struct {
	movieService portservices.MovieService
}

func NewMovieHandler(movieService portservices.MovieService) *MovieHandler {
	return &MovieHandler{movieService: movieService}
}

// @Summary      Search movies
// @Description  Search for movies by title
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        query query string true "Search query"
// @Param        page query int false "Page number" default(1)
// @Param        year query int false "Filter by release year"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Search results"
// @Failure      400 {object} response.Response "Query parameter is required"
// @Failure      502 {object} response.Response "External service error"
// @Router       /movies/search [get]
func (h *MovieHandler) Search(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		response.BadRequest(c, "Query parameter is required", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	year, _ := strconv.Atoi(c.Query("year"))
	language := middleware.GetLocale(c)

	result, err := h.movieService.Search(c.Request.Context(), portservices.SearchMoviesInput{
		Query:    query,
		Page:     page,
		Year:     year,
		Language: language,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessPaginated(c, result.Results, &response.Pagination{
		Page:       result.Page,
		PerPage:    20,
		Total:      result.TotalResults,
		TotalPages: result.TotalPages,
	})
}

// @Summary      Get movie details
// @Description  Get detailed information about a movie by its TMDB ID
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        id path int true "Movie ID"
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.Response "Movie details"
// @Failure      400 {object} response.Response "Invalid movie ID"
// @Failure      404 {object} response.Response "Movie not found"
// @Failure      502 {object} response.Response "External service error"
// @Router       /movies/{id} [get]
func (h *MovieHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid movie ID", nil)
		return
	}

	language := middleware.GetLocale(c)

	movie, err := h.movieService.GetByID(c.Request.Context(), id, language)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, movie)
}

// @Summary      Get popular movies
// @Description  Get a list of currently popular movies
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Popular movies"
// @Failure      502 {object} response.Response "External service error"
// @Router       /movies/popular [get]
func (h *MovieHandler) GetPopular(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	language := middleware.GetLocale(c)

	result, err := h.movieService.GetPopular(c.Request.Context(), page, language)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessPaginated(c, result.Results, &response.Pagination{
		Page:       result.Page,
		PerPage:    20,
		Total:      result.TotalResults,
		TotalPages: result.TotalPages,
	})
}
