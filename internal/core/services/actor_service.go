package services

import (
	"context"
	"errors"
	"sort"
	"strings"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"
	"duskforge-api/pkg/tmdb"
)

type actorService struct {
	tmdbClient ports.TMDBClient
	reviewRepo ports.ReviewRepository
}

func NewActorService(tmdbClient ports.TMDBClient, reviewRepo ports.ReviewRepository) ports.ActorService {
	return &actorService{
		tmdbClient: tmdbClient,
		reviewRepo: reviewRepo,
	}
}

func (s *actorService) Search(ctx context.Context, input ports.SearchActorsInput) (*ports.SearchActorsResult, error) {
	if input.Query == "" {
		return &ports.SearchActorsResult{
			Offset:  input.Offset,
			Limit:   input.Limit,
			Total:   0,
			Results: []ports.ActorSearchResult{},
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

	result, err := s.tmdbClient.SearchPerson(ctx, tmdb.SearchPersonParams{
		Query:    input.Query,
		Page:     page,
		Language: input.Language,
	})
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	if len(result.Results) == 0 {
		return &ports.SearchActorsResult{
			Offset:  input.Offset,
			Limit:   input.Limit,
			Total:   result.TotalResults,
			Results: []ports.ActorSearchResult{},
		}, nil
	}

	start := offsetInPage
	if start >= len(result.Results) {
		return &ports.SearchActorsResult{
			Offset:  input.Offset,
			Limit:   input.Limit,
			Total:   result.TotalResults,
			Results: []ports.ActorSearchResult{},
		}, nil
	}

	end := start + input.Limit
	if end > len(result.Results) {
		end = len(result.Results)
	}

	slicedResults := result.Results[start:end]
	actors := make([]ports.ActorSearchResult, len(slicedResults))
	for i, p := range slicedResults {
		actors[i] = ports.ActorSearchResult{
			ID:                 p.ID,
			Name:               p.Name,
			ProfilePath:        p.ProfilePath,
			KnownForDepartment: p.KnownForDepartment,
		}
	}

	return &ports.SearchActorsResult{
		Offset:  input.Offset,
		Limit:   len(actors),
		Total:   result.TotalResults,
		Results: actors,
	}, nil
}

func (s *actorService) GetByID(ctx context.Context, actorID int, language string) (*ports.ActorDetailsResult, error) {
	person, err := s.tmdbClient.GetPersonDetails(ctx, actorID, language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrActorNotFound
		}
		return nil, domain.ErrTMDBError
	}

	return &ports.ActorDetailsResult{
		ID:                 person.ID,
		Name:               person.Name,
		Biography:          person.Biography,
		Birthday:           person.Birthday,
		Deathday:           person.Deathday,
		PlaceOfBirth:       person.PlaceOfBirth,
		ProfilePath:        person.ProfilePath,
		KnownForDepartment: person.KnownForDepartment,
	}, nil
}

func (s *actorService) GetFilmography(ctx context.Context, input ports.GetActorFilmographyInput) (*ports.ActorFilmographyResult, error) {
	if input.Limit <= 0 {
		input.Limit = tmdbPageSize
	}
	if input.Limit > tmdbPageSize {
		input.Limit = tmdbPageSize
	}

	credits, err := s.tmdbClient.GetPersonMovieCredits(ctx, input.ActorID, input.Language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrActorNotFound
		}
		return nil, domain.ErrTMDBError
	}

	movieMap := make(map[int]*ports.ActorFilmCredit)

	for _, cast := range credits.Cast {
		if _, exists := movieMap[cast.ID]; !exists {
			var tmdbRating *float64
			if cast.VoteCount > 0 {
				rating := cast.VoteAverage / 2
				tmdbRating = &rating
			}
			movieMap[cast.ID] = &ports.ActorFilmCredit{
				ID:          cast.ID,
				Title:       cast.Title,
				Role:        cast.Character,
				Department:  "Acting",
				PosterPath:  cast.PosterPath,
				ReleaseDate: cast.ReleaseDate,
				TMDBRating:  tmdbRating,
			}
		}
	}

	for _, crew := range credits.Crew {
		if _, exists := movieMap[crew.ID]; !exists {
			var tmdbRating *float64
			if crew.VoteCount > 0 {
				rating := crew.VoteAverage / 2
				tmdbRating = &rating
			}
			movieMap[crew.ID] = &ports.ActorFilmCredit{
				ID:          crew.ID,
				Title:       crew.Title,
				Role:        crew.Job,
				Department:  crew.Department,
				PosterPath:  crew.PosterPath,
				ReleaseDate: crew.ReleaseDate,
				TMDBRating:  tmdbRating,
			}
		}
	}

	allCredits := make([]ports.ActorFilmCredit, 0, len(movieMap))
	tmdbIDs := make([]int, 0, len(movieMap))
	for id, credit := range movieMap {
		allCredits = append(allCredits, *credit)
		tmdbIDs = append(tmdbIDs, id)
	}

	ratings, err := s.reviewRepo.GetAverageRatingsByTMDBIDs(ctx, tmdbIDs)
	if err != nil {
		ratings = make(map[int]float64)
	}
	for i := range allCredits {
		if r, ok := ratings[allCredits[i].ID]; ok {
			allCredits[i].DuskforgeRating = &r
		}
	}

	sortFilmography(allCredits, input.Sort)

	totalResults := len(allCredits)

	start := input.Offset
	if start >= len(allCredits) {
		return &ports.ActorFilmographyResult{
			Offset:  input.Offset,
			Limit:   input.Limit,
			Total:   totalResults,
			Results: []ports.ActorFilmCredit{},
		}, nil
	}

	end := start + input.Limit
	if end > len(allCredits) {
		end = len(allCredits)
	}

	return &ports.ActorFilmographyResult{
		Offset:  input.Offset,
		Limit:   end - start,
		Total:   totalResults,
		Results: allCredits[start:end],
	}, nil
}

func sortFilmography(credits []ports.ActorFilmCredit, sortOption string) {
	if sortOption == "" {
		sortOption = "-release_date"
	}

	ascending := false
	field := sortOption
	switch sortOption[0] {
	case '+':
		ascending = true
		field = sortOption[1:]
	case '-':
		ascending = false
		field = sortOption[1:]
	}

	sort.Slice(credits, func(i, j int) bool {
		var less bool
		switch field {
		case "release_date":
			less = credits[i].ReleaseDate < credits[j].ReleaseDate
		case "popularity":
			ri := float64(0)
			rj := float64(0)
			if credits[i].TMDBRating != nil {
				ri = *credits[i].TMDBRating
			}
			if credits[j].TMDBRating != nil {
				rj = *credits[j].TMDBRating
			}
			less = ri < rj
		case "rating":
			ri := float64(0)
			rj := float64(0)
			if credits[i].TMDBRating != nil {
				ri = *credits[i].TMDBRating
			}
			if credits[j].TMDBRating != nil {
				rj = *credits[j].TMDBRating
			}
			less = ri < rj
		case "name":
			less = strings.ToLower(credits[i].Title) < strings.ToLower(credits[j].Title)
		default:
			less = credits[i].ReleaseDate < credits[j].ReleaseDate
		}

		if ascending {
			return less
		}
		return !less
	})
}
