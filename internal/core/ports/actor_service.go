package ports

import (
	"context"
)

// Input DTOs
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

// Output DTOs
type ActorSearchResult struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	ProfilePath        *string `json:"profile_path"`
	KnownForDepartment string  `json:"known_for_department"`
	Popularity         float64 `json:"popularity"`
}

type SearchActorsResult struct {
	Offset  int
	Limit   int
	Total   int
	Results []ActorSearchResult
}

type ActorDetailsResult struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Biography          string   `json:"biography"`
	Birthday           *string  `json:"birthday"`
	Deathday           *string  `json:"deathday"`
	PlaceOfBirth       *string  `json:"place_of_birth"`
	ProfilePath        *string  `json:"profile_path"`
	KnownForDepartment string   `json:"known_for_department"`
	Popularity         float64  `json:"popularity"`
	AlsoKnownAs        []string `json:"also_known_as"`
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

// Interface
type ActorService interface {
	Search(ctx context.Context, input SearchActorsInput) (*SearchActorsResult, error)
	GetByID(ctx context.Context, actorID int, language string) (*ActorDetailsResult, error)
	GetFilmography(ctx context.Context, input GetActorFilmographyInput) (*ActorFilmographyResult, error)
}
