package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/tmdb"
)

type movieService struct {
	tmdbClient ports.TMDBClient
	reviewRepo ports.ReviewRepository
}

func NewMovieService(tmdbClient ports.TMDBClient, reviewRepo ports.ReviewRepository) ports.MovieService {
	return &movieService{
		tmdbClient: tmdbClient,
		reviewRepo: reviewRepo,
	}
}

func (s *movieService) Search(ctx context.Context, input ports.SearchMoviesInput) (*ports.SearchMoviesResult, error) {
	if input.Query == "" {
		return &ports.SearchMoviesResult{
			Page:         1,
			TotalPages:   0,
			TotalResults: 0,
			Results:      []ports.MovieSearchResult{},
		}, nil
	}

	result, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    input.Query,
		Page:     input.Page,
		Year:     input.Year,
		Language: input.Language,
	})
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMovies(ctx, result.Results, result.Page, result.TotalPages, result.TotalResults, input.Language)
}

func (s *movieService) Discover(ctx context.Context, input ports.DiscoverMoviesInput) (*ports.SearchMoviesResult, error) {
	params := tmdb.DiscoverMoviesParams{
		Page:       input.Page,
		Language:   input.Language,
		WithGenres: input.Genres,
		WithCast:   input.WithCast,
	}

	// Year range filter
	if input.YearFrom > 0 {
		params.PrimaryReleaseDateGTE = fmt.Sprintf("%d-01-01", input.YearFrom)
	}
	if input.YearTo > 0 {
		params.PrimaryReleaseDateLTE = fmt.Sprintf("%d-12-31", input.YearTo)
	}

	// Sorting
	params.SortBy = parseSort(input.Sort)

	result, err := s.tmdbClient.DiscoverMovies(ctx, params)
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMovies(ctx, result.Results, result.Page, result.TotalPages, result.TotalResults, input.Language)
}

func (s *movieService) GetByID(ctx context.Context, movieID int, language string) (*tmdb.MovieDetails, error) {
	movie, err := s.tmdbClient.GetMovieDetails(ctx, movieID, language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrMovieNotFound
		}
		return nil, domain.ErrTMDBError
	}
	return movie, nil
}

func (s *movieService) GetPopular(ctx context.Context, page int, language string) (*ports.SearchMoviesResult, error) {
	result, err := s.tmdbClient.GetPopularMovies(ctx, page, language, "")
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMovies(ctx, result.Results, result.Page, result.TotalPages, result.TotalResults, language)
}

func parseSort(sort string) tmdb.SortBy {
	if sort == "" {
		return tmdb.SortByPopularityDesc
	}

	order := ".desc"
	field := sort

	if len(sort) > 0 {
		switch sort[0] {
		case '+':
			order = ".asc"
			field = sort[1:]
		case '-':
			order = ".desc"
			field = sort[1:]
		}
	}

	switch field {
	case "rating":
		return tmdb.SortBy("vote_average" + order)
	case "release_date":
		return tmdb.SortBy("primary_release_date" + order)
	case "popularity":
		return tmdb.SortBy("popularity" + order)
	default:
		return tmdb.SortByPopularityDesc
	}
}

func (s *movieService) transformMovies(ctx context.Context, movies []tmdb.MovieSummary, page, totalPages, totalResults int, language string) (*ports.SearchMoviesResult, error) {
	if len(movies) == 0 {
		return &ports.SearchMoviesResult{
			Page:         page,
			TotalPages:   totalPages,
			TotalResults: totalResults,
			Results:      []ports.MovieSearchResult{},
		}, nil
	}

	tmdbIDs := make([]int, len(movies))
	for i, movie := range movies {
		tmdbIDs[i] = movie.ID
	}

	directors := s.fetchDirectors(ctx, tmdbIDs, language)

	ratings, err := s.reviewRepo.GetAverageRatingsByTMDBIDs(ctx, tmdbIDs)
	if err != nil {
		ratings = make(map[int]float64)
	}

	results := make([]ports.MovieSearchResult, len(movies))
	for i, movie := range movies {
		var director *string
		if d, ok := directors[movie.ID]; ok && d != "" {
			director = &d
		}

		var duskforgeRating *float64
		if r, ok := ratings[movie.ID]; ok {
			duskforgeRating = &r
		}

		results[i] = ports.MovieSearchResult{
			ID:              movie.ID,
			Poster:          movie.PosterPath,
			Name:            movie.Title,
			Date:            movie.ReleaseDate,
			Director:        director,
			TMDBRating:      movie.VoteAverage / 2,
			DuskforgeRating: duskforgeRating,
		}
	}

	return &ports.SearchMoviesResult{
		Page:         page,
		TotalPages:   totalPages,
		TotalResults: totalResults,
		Results:      results,
	}, nil
}

func (s *movieService) fetchDirectors(ctx context.Context, movieIDs []int, language string) map[int]string {
	directors := make(map[int]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, movieID := range movieIDs {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			credits, err := s.tmdbClient.GetMovieCredits(ctx, id, language)
			if err != nil {
				return
			}

			for _, crew := range credits.Crew {
				if crew.Job == "Director" {
					mu.Lock()
					if directors[id] == "" {
						directors[id] = crew.Name
					} else {
						directors[id] += ", " + crew.Name
					}
					mu.Unlock()
				}
			}
		}(movieID)
	}

	wg.Wait()
	return directors
}
