package services

import (
	"context"
	"errors"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	portservices "duskforge-api/internal/core/ports/services"
	"duskforge-api/pkg/tmdb"
)

type movieService struct {
	tmdbClient ports.TMDBClient
}

func NewMovieService(tmdbClient ports.TMDBClient) portservices.MovieService {
	return &movieService{tmdbClient: tmdbClient}
}

func (s *movieService) Search(ctx context.Context, input portservices.SearchMoviesInput) (*tmdb.SearchMoviesResponse, error) {
	result, err := s.tmdbClient.SearchMovies(ctx, tmdb.SearchMoviesParams{
		Query:    input.Query,
		Page:     input.Page,
		Year:     input.Year,
		Language: input.Language,
	})
	if err != nil {
		return nil, domain.ErrTMDBError
	}
	return result, nil
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

func (s *movieService) GetPopular(ctx context.Context, page int, language string) (*tmdb.PopularMoviesResponse, error) {
	result, err := s.tmdbClient.GetPopularMovies(ctx, page, language, "")
	if err != nil {
		return nil, domain.ErrTMDBError
	}
	return result, nil
}
