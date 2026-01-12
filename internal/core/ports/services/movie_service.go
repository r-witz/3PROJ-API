package services

import (
	"context"

	"duskforge-api/pkg/tmdb"
)

type SearchMoviesInput struct {
	Query    string
	Page     int
	Year     int
	Language string
}

type MovieService interface {
	Search(ctx context.Context, input SearchMoviesInput) (*tmdb.SearchMoviesResponse, error)
	GetByID(ctx context.Context, movieID int, language string) (*tmdb.MovieDetails, error)
	GetPopular(ctx context.Context, page int, language string) (*tmdb.PopularMoviesResponse, error)
}
