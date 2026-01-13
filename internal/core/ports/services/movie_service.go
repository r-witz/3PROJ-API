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

type MovieSearchResult struct {
	ID              int      `json:"id"`
	Poster          *string  `json:"poster"`
	Name            string   `json:"name"`
	Date            string   `json:"date"`
	Director        *string  `json:"director"`
	TMDBRating      float64  `json:"tmdb_rating"`
	DuskforgeRating *float64 `json:"duskforge_rating"`
}

type SearchMoviesResult struct {
	Page         int
	TotalPages   int
	TotalResults int
	Results      []MovieSearchResult
}

type MovieService interface {
	Search(ctx context.Context, input SearchMoviesInput) (*SearchMoviesResult, error)
	GetByID(ctx context.Context, movieID int, language string) (*tmdb.MovieDetails, error)
	GetPopular(ctx context.Context, page int, language string) (*SearchMoviesResult, error)
}
