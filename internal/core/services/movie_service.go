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

const tmdbPageSize = 20

func (s *movieService) Search(ctx context.Context, input ports.SearchMoviesInput) (*ports.SearchMoviesResult, error) {
	if input.Query == "" {
		return &ports.SearchMoviesResult{
			Offset:  input.Offset,
			Limit:   input.Limit,
			Total:   0,
			Results: []ports.MovieSearchResult{},
		}, nil
	}

	if input.Limit <= 0 {
		input.Limit = tmdbPageSize
	}
	if input.Limit > tmdbPageSize {
		input.Limit = tmdbPageSize
	}

	page := (input.Offset / tmdbPageSize) + 1
	offsetInPage := input.Offset % tmdbPageSize

	result, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    input.Query,
		Page:     page,
		Year:     input.Year,
		Language: input.Language,
	})
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMoviesWithOffset(ctx, result.Results, input.Offset, input.Limit, offsetInPage, result.TotalResults, input.Language)
}

func (s *movieService) Discover(ctx context.Context, input ports.DiscoverMoviesInput) (*ports.SearchMoviesResult, error) {
	if input.Limit <= 0 {
		input.Limit = tmdbPageSize
	}
	if input.Limit > tmdbPageSize {
		input.Limit = tmdbPageSize
	}

	page := (input.Offset / tmdbPageSize) + 1
	offsetInPage := input.Offset % tmdbPageSize

	params := tmdb.DiscoverMoviesParams{
		Page:       page,
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

	return s.transformMoviesWithOffset(ctx, result.Results, input.Offset, input.Limit, offsetInPage, result.TotalResults, input.Language)
}

func (s *movieService) GetByID(ctx context.Context, movieID int, language string) (*ports.MovieDetailsResult, error) {
	movie, err := s.tmdbClient.GetMovieDetails(ctx, movieID, language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrMovieNotFound
		}
		return nil, domain.ErrTMDBError
	}

	genres := make([]ports.Genre, len(movie.Genres))
	for i, g := range movie.Genres {
		genres[i] = ports.Genre{
			ID:   g.ID,
			Name: g.Name,
		}
	}

	// TMDB rating - null if no votes
	tmdbRating := ports.RatingInfo{
		Rating: nil,
		Count:  movie.VoteCount,
	}
	if movie.VoteCount > 0 {
		rating := movie.VoteAverage / 2
		tmdbRating.Rating = &rating
	}

	// Duskforge rating - null if no reviews
	duskforgeRating := ports.RatingInfo{
		Rating: nil,
		Count:  0,
	}
	ratingStats, err := s.reviewRepo.GetRatingStatsByTMDBIDs(ctx, []int{movieID})
	if err == nil {
		if stats, ok := ratingStats[movieID]; ok {
			duskforgeRating.Rating = &stats.Rating
			duskforgeRating.Count = stats.Count
		}
	}

	return &ports.MovieDetailsResult{
		ID:           movie.ID,
		Title:        movie.Title,
		Overview:     movie.Overview,
		Tagline:      movie.Tagline,
		PosterPath:   movie.PosterPath,
		BackdropPath: movie.BackdropPath,
		ReleaseDate:  movie.ReleaseDate,
		Runtime:      movie.Runtime,
		Adult:        movie.Adult,
		Genres:       genres,
		Ratings: ports.MovieRatings{
			TMDB:      tmdbRating,
			Duskforge: duskforgeRating,
		},
		Financials: ports.MovieFinancials{
			Budget:  movie.Budget,
			Revenue: movie.Revenue,
		},
	}, nil
}

func (s *movieService) GetPopular(ctx context.Context, offset, limit int, language string) (*ports.SearchMoviesResult, error) {
	if limit <= 0 {
		limit = tmdbPageSize
	}
	if limit > tmdbPageSize {
		limit = tmdbPageSize
	}

	page := (offset / tmdbPageSize) + 1
	offsetInPage := offset % tmdbPageSize

	result, err := s.tmdbClient.GetPopularMovies(ctx, page, language, "")
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMoviesWithOffset(ctx, result.Results, offset, limit, offsetInPage, result.TotalResults, language)
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

func (s *movieService) transformMoviesWithOffset(ctx context.Context, movies []tmdb.MovieSummary, offset, limit, offsetInPage, totalResults int, language string) (*ports.SearchMoviesResult, error) {
	if len(movies) == 0 {
		return &ports.SearchMoviesResult{
			Offset:  offset,
			Limit:   limit,
			Total:   totalResults,
			Results: []ports.MovieSearchResult{},
		}, nil
	}

	// Slice movies based on offset within page and limit
	start := offsetInPage
	if start >= len(movies) {
		return &ports.SearchMoviesResult{
			Offset:  offset,
			Limit:   limit,
			Total:   totalResults,
			Results: []ports.MovieSearchResult{},
		}, nil
	}

	end := start + limit
	if end > len(movies) {
		end = len(movies)
	}

	slicedMovies := movies[start:end]

	tmdbIDs := make([]int, len(slicedMovies))
	for i, movie := range slicedMovies {
		tmdbIDs[i] = movie.ID
	}

	directors := s.fetchDirectors(ctx, tmdbIDs, language)

	ratings, err := s.reviewRepo.GetAverageRatingsByTMDBIDs(ctx, tmdbIDs)
	if err != nil {
		ratings = make(map[int]float64)
	}

	results := make([]ports.MovieSearchResult, len(slicedMovies))
	for i, movie := range slicedMovies {
		var director *string
		if d, ok := directors[movie.ID]; ok && d != "" {
			director = &d
		}

		var duskforgeRating *float64
		if r, ok := ratings[movie.ID]; ok {
			duskforgeRating = &r
		}

		var tmdbRating *float64
		if movie.VoteCount > 0 {
			rating := movie.VoteAverage / 2
			tmdbRating = &rating
		}

		results[i] = ports.MovieSearchResult{
			ID:              movie.ID,
			Poster:          movie.PosterPath,
			Name:            movie.Title,
			Date:            movie.ReleaseDate,
			Director:        director,
			TMDBRating:      tmdbRating,
			DuskforgeRating: duskforgeRating,
		}
	}

	return &ports.SearchMoviesResult{
		Offset:  offset,
		Limit:   len(results),
		Total:   totalResults,
		Results: results,
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
