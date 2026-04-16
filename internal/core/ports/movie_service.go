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
	Offset           int
	Limit            int
	Language         string
	YearFrom         int
	YearTo           int
	Genres           []int
	Sort             string
	RuntimeGTE       int
	RuntimeLTE       int
	OriginalLanguage string
}

type MovieSearchResult struct {
	ID              int      `json:"id"`
	Poster          *string  `json:"poster"`
	Name            string   `json:"name"`
	Date            string   `json:"date"`
	TMDBRating      *float64 `json:"tmdb_rating"`
	DuskforgeRating *float64 `json:"duskforge_rating"`
	UserRating      *float64 `json:"user_rating"`
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
	TMDB      RatingInfo `json:"tmdb"`
	Duskforge RatingInfo `json:"duskforge"`
}

type RatingInfo struct {
	Rating *float64 `json:"rating"`
	Count  int      `json:"count"`
}

type MovieFinancials struct {
	Budget  int64 `json:"budget"`
	Revenue int64 `json:"revenue"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MovieTrailerResult struct {
	EmbedURL *string `json:"embed_url"`
}

type PersonResult struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Role    string  `json:"role"`
	Picture *string `json:"picture"`
}

type MovieCastResult struct {
	Cast      []PersonResult `json:"cast"`
	Directors []PersonResult `json:"directors"`
	Writers   []PersonResult `json:"writers"`
	Crew      []PersonResult `json:"crew"`
}

type ReleaseDateItem struct {
	Date          string `json:"date"`
	Type          string `json:"type"`
	Certification string `json:"certification"`
}

type RegionReleaseDates struct {
	Region       string            `json:"region"`
	ReleaseDates []ReleaseDateItem `json:"release_dates"`
}

type MovieReleaseDatesResult struct {
	Regions []RegionReleaseDates `json:"regions"`
}

type MovieService interface {
	Search(ctx context.Context, input SearchMoviesInput) (*SearchMoviesResult, error)
	Discover(ctx context.Context, input DiscoverMoviesInput) (*SearchMoviesResult, error)
	GetByID(ctx context.Context, movieID int, language string) (*MovieDetailsResult, error)
	GetPopular(ctx context.Context, offset, limit int, language string) (*SearchMoviesResult, error)
	GetUpcoming(ctx context.Context, offset, limit int, language string) (*SearchMoviesResult, error)
	GetTrailer(ctx context.Context, movieID int, language string) (*MovieTrailerResult, error)
	GetCast(ctx context.Context, movieID int, language string) (*MovieCastResult, error)
	GetReleaseDates(ctx context.Context, movieID int) (*MovieReleaseDatesResult, error)
	GetGenres(ctx context.Context, language string) ([]Genre, error)
}
