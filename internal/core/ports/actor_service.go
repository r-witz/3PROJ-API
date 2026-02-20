package ports

import (
	"context"
)


type SearchActorsInput struct {
	Query    string
	Offset   int
	Limit    int
	Language string
}

type GetActorFilmographyInput struct {
	ActorID  int
	Offset   int
	Limit    int
	Sort     string
	Language string
}

type ActorSearchResult struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	ProfilePath        *string `json:"profile_path"`
	KnownForDepartment string  `json:"known_for_department"`
}

type SearchActorsResult struct {
	Offset  int
	Limit   int
	Total   int
	Results []ActorSearchResult
}

type ActorDetailsResult struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Biography          string  `json:"biography"`
	Birthday           *string `json:"birthday"`
	Deathday           *string `json:"deathday"`
	PlaceOfBirth       *string `json:"place_of_birth"`
	ProfilePath        *string `json:"profile_path"`
	KnownForDepartment string  `json:"known_for_department"`
}

type ActorFilmCredit struct {
	ID              int      `json:"id"`
	Title           string   `json:"title"`
	Role            string   `json:"role"`
	Department      string   `json:"department"`
	PosterPath      *string  `json:"poster_path"`
	ReleaseDate     string   `json:"release_date"`
	TMDBRating      *float64 `json:"tmdb_rating"`
	DuskforgeRating *float64 `json:"duskforge_rating"`
}

type ActorFilmographyResult struct {
	Offset  int
	Limit   int
	Total   int
	Results []ActorFilmCredit
}

type ActorService interface {
	Search(ctx context.Context, input SearchActorsInput) (*SearchActorsResult, error)
	GetByID(ctx context.Context, actorID int, language string) (*ActorDetailsResult, error)
	GetFilmography(ctx context.Context, input GetActorFilmographyInput) (*ActorFilmographyResult, error)
}
