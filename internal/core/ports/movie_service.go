package ports

import (
	"context"
)

type SearchMoviesInput struct {
	Query    string
	Offset   int
	Limit    int
	Year     int
	Language string
}

type DiscoverMoviesInput struct {
	Offset   int
	Limit    int
	Language string
	YearFrom int
	YearTo   int
	Genres   []int
	WithCast []int
	Sort     string
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
	Offset int
	Limit  int
	Total  int
	Results []MovieSearchResult
}

type MovieDetailsResult struct {
	ID           int            `json:"id"`
	Title        string         `json:"title"`
	Overview     string         `json:"overview"`
	Tagline      string         `json:"tagline"`
	PosterPath   *string        `json:"poster_path"`
	BackdropPath *string        `json:"backdrop_path"`
	ReleaseDate  string         `json:"release_date"`
	Runtime      *int           `json:"runtime"`
	Adult        bool           `json:"adult"`
	Genres       []Genre        `json:"genres"`
	Ratings      MovieRatings   `json:"ratings"`
	Financials   MovieFinancials `json:"financials"`
}

type MovieRatings struct {
	TMDB      RatingInfo  `json:"tmdb"`
	Duskforge *RatingInfo `json:"duskforge"`
}

type RatingInfo struct {
	Rating float64 `json:"rating"`
	Count  int     `json:"count"`
}

type MovieFinancials struct {
	Budget  int64 `json:"budget"`
	Revenue int64 `json:"revenue"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MovieService interface {
	Search(ctx context.Context, input SearchMoviesInput) (*SearchMoviesResult, error)
	Discover(ctx context.Context, input DiscoverMoviesInput) (*SearchMoviesResult, error)
	GetByID(ctx context.Context, movieID int, language string) (*MovieDetailsResult, error)
	GetPopular(ctx context.Context, offset, limit int, language string) (*SearchMoviesResult, error)
}
