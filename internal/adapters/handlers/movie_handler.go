package handlers

import (
	"strconv"
	"strings"

	"duskforge-api/internal/adapters/middleware"
	"duskforge-api/internal/adapters/response"
	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type MovieHandler struct {
	movieService ports.MovieService
}

func NewMovieHandler(movieService ports.MovieService) *MovieHandler {
	return &MovieHandler{movieService: movieService}
}

// @Summary      Search movies by title
// @Description  Search for movies by title. Use this endpoint when users are looking for a specific movie by name.
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        query query string true "Search query"
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 20)" default(20)
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

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	year, _ := strconv.Atoi(c.Query("year"))
	language := middleware.GetLocale(c)

	result, err := h.movieService.Search(c.Request.Context(), ports.SearchMoviesInput{
		Query:    query,
		Offset:   offset,
		Limit:    limit,
		Year:     year,
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

// @Summary      Discover movies with filters
// @Description  Discover movies with advanced filtering and sorting. Use this for browsing by genre, year range, cast, etc.
// @Tags         movies
// @Accept       json
// @Produce      json
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 20)" default(20)
// @Param        year_from query int false "Filter by starting year"
// @Param        year_to query int false "Filter by ending year"
// @Param        genres query string false "Filter by genre IDs (comma-separated)"
// @Param        cast query string false "Filter by cast/actor IDs (comma-separated)"
// @Param        sort query string false "Sort field with direction prefix (+asc, -desc)" Enums(+popularity, -popularity, +rating, -rating, +release_date, -release_date) default(-popularity)
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Discover results"
// @Failure      502 {object} response.Response "External service error"
// @Router       /movies/discover [get]
func (h *MovieHandler) Discover(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	yearFrom, _ := strconv.Atoi(c.Query("year_from"))
	yearTo, _ := strconv.Atoi(c.Query("year_to"))
	sort := c.DefaultQuery("sort", "-popularity")
	language := middleware.GetLocale(c)

	var genres []int
	if genresStr := c.Query("genres"); genresStr != "" {
		for _, g := range strings.Split(genresStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(g)); err == nil {
				genres = append(genres, id)
			}
		}
	}

	var cast []int
	if castStr := c.Query("cast"); castStr != "" {
		for _, p := range strings.Split(castStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				cast = append(cast, id)
			}
		}
	}

	result, err := h.movieService.Discover(c.Request.Context(), ports.DiscoverMoviesInput{
		Offset:   offset,
		Limit:    limit,
		Language: language,
		YearFrom: yearFrom,
		YearTo:   yearTo,
		Genres:   genres,
		WithCast: cast,
		Sort:     sort,
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
// @Param        offset query int false "Number of items to skip" default(0)
// @Param        limit query int false "Number of items to return (max 20)" default(20)
// @Param        Accept-Language header string false "Language code (e.g., en, fr)"
// @Success      200 {object} response.PaginatedResponse "Popular movies"
// @Failure      502 {object} response.Response "External service error"
// @Router       /movies/popular [get]
func (h *MovieHandler) GetPopular(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	language := middleware.GetLocale(c)

	result, err := h.movieService.GetPopular(c.Request.Context(), offset, limit, language)
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
