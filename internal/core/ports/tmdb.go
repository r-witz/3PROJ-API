package ports

import (
	"context"

	"duskforge-api/pkg/tmdb"
)

type TMDBClient interface {
	SearchMovies(ctx context.Context, params tmdb.SearchMoviesParams) (*tmdb.SearchMoviesResponse, error)
	DiscoverMovies(ctx context.Context, params tmdb.DiscoverMoviesParams) (*tmdb.DiscoverMoviesResponse, error)
	GetPopularMovies(ctx context.Context, page int, language, region string) (*tmdb.PopularMoviesResponse, error)
	GetTrendingMovies(ctx context.Context, page int, language string) (*tmdb.TrendingMoviesResponse, error)
	GetNowPlayingMovies(ctx context.Context, page int, language, region string) (*tmdb.NowPlayingMoviesResponse, error)
	GetMovieDetails(ctx context.Context, movieID int, language string) (*tmdb.MovieDetails, error)
	GetMovieCredits(ctx context.Context, movieID int, language string) (*tmdb.Credits, error)
	GetMovieWithCredits(ctx context.Context, movieID int, language string) (*tmdb.MovieDetails, *tmdb.Credits, error)
	GetGenres(ctx context.Context, language string) ([]tmdb.Genre, error)
	GetConfiguration(ctx context.Context) (*tmdb.Configuration, error)
	ImageURLs() *tmdb.ImageURLBuilder
	SearchPerson(ctx context.Context, params tmdb.SearchPersonParams) (*tmdb.SearchPersonResponse, error)
	GetPersonDetails(ctx context.Context, personID int, language string) (*tmdb.PersonDetails, error)
	GetPersonMovieCredits(ctx context.Context, personID int, language string) (*tmdb.PersonMovieCredits, error)
	GetMovieVideos(ctx context.Context, movieID int, language string) (*tmdb.VideosResponse, error)
	GetMovieReleaseDates(ctx context.Context, movieID int) (*tmdb.ReleaseDatesResponse, error)
}
