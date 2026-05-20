package services

import (
	"context"
	"errors"
	"fmt"

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
	}

	if input.YearFrom > 0 {
		params.PrimaryReleaseDateGTE = fmt.Sprintf("%d-01-01", input.YearFrom)
	}
	if input.YearTo > 0 {
		params.PrimaryReleaseDateLTE = fmt.Sprintf("%d-12-31", input.YearTo)
	}

	if input.RuntimeGTE > 0 {
		params.WithRuntimeGTE = input.RuntimeGTE
	}
	if input.RuntimeLTE > 0 {
		params.WithRuntimeLTE = input.RuntimeLTE
	}
	if input.OriginalLanguage != "" {
		params.WithOriginalLanguage = input.OriginalLanguage
	}

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

	tmdbRating := ports.RatingInfo{
		Rating: nil,
		Count:  movie.VoteCount,
	}
	if movie.VoteCount > 0 {
		rating := movie.VoteAverage / 2
		tmdbRating.Rating = &rating
	}

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

	result, err := s.tmdbClient.GetTrendingMovies(ctx, page, language)
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	return s.transformMoviesWithOffset(ctx, result.Results, offset, limit, offsetInPage, result.TotalResults, language)
}

func (s *movieService) GetUpcoming(ctx context.Context, offset, limit int, language string) (*ports.SearchMoviesResult, error) {
	if limit <= 0 {
		limit = tmdbPageSize
	}
	if limit > tmdbPageSize {
		limit = tmdbPageSize
	}

	page := (offset / tmdbPageSize) + 1
	offsetInPage := offset % tmdbPageSize

	result, err := s.tmdbClient.GetUpcomingMovies(ctx, page, language, "")
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

	switch sort[0] {
	case '+':
		order = ".asc"
		field = sort[1:]
	case '-':
		order = ".desc"
		field = sort[1:]
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

	ratings, err := s.reviewRepo.GetAverageRatingsByTMDBIDs(ctx, tmdbIDs)
	if err != nil {
		ratings = make(map[int]float64)
	}

	results := make([]ports.MovieSearchResult, len(slicedMovies))
	for i, movie := range slicedMovies {
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

func (s *movieService) GetTrailer(ctx context.Context, movieID int, language string) (*ports.MovieTrailerResult, error) {
	videos, err := s.tmdbClient.GetMovieVideos(ctx, movieID, language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrMovieNotFound
		}
		return nil, domain.ErrTMDBError
	}

	var bestTrailer *tmdb.Video
	for i := range videos.Results {
		v := &videos.Results[i]
		if v.Site != "YouTube" {
			continue
		}
		if v.Type == "Trailer" {
			if bestTrailer == nil || (v.Official && !bestTrailer.Official) {
				bestTrailer = v
			}
		}
	}

	if bestTrailer == nil {
		for i := range videos.Results {
			v := &videos.Results[i]
			if v.Site == "YouTube" {
				bestTrailer = v
				break
			}
		}
	}

	if bestTrailer == nil {
		return &ports.MovieTrailerResult{EmbedURL: nil}, nil
	}

	embedURL := fmt.Sprintf("https://www.youtube.com/embed/%s", bestTrailer.Key)
	return &ports.MovieTrailerResult{EmbedURL: &embedURL}, nil
}

func (s *movieService) GetCast(ctx context.Context, movieID int, language string) (*ports.MovieCastResult, error) {
	credits, err := s.tmdbClient.GetMovieCredits(ctx, movieID, language)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrMovieNotFound
		}
		return nil, domain.ErrTMDBError
	}

	cast := make([]ports.PersonResult, len(credits.Cast))
	for i, c := range credits.Cast {
		cast[i] = ports.PersonResult{
			ID:      c.ID,
			Name:    c.Name,
			Role:    c.Character,
			Picture: c.ProfilePath,
		}
	}

	var directors, writers, crew []ports.PersonResult
	writerJobs := map[string]bool{"Writer": true, "Screenplay": true, "Story": true}

	for _, c := range credits.Crew {
		person := ports.PersonResult{
			ID:      c.ID,
			Name:    c.Name,
			Role:    c.Job,
			Picture: c.ProfilePath,
		}

		switch {
		case c.Job == "Director":
			directors = append(directors, person)
		case writerJobs[c.Job]:
			writers = append(writers, person)
		default:
			crew = append(crew, person)
		}
	}

	return &ports.MovieCastResult{
		Cast:      cast,
		Directors: directors,
		Writers:   writers,
		Crew:      crew,
	}, nil
}

func (s *movieService) GetReleaseDates(ctx context.Context, movieID int) (*ports.MovieReleaseDatesResult, error) {
	releaseDates, err := s.tmdbClient.GetMovieReleaseDates(ctx, movieID)
	if err != nil {
		if errors.Is(err, tmdb.ErrNotFound) {
			return nil, domain.ErrMovieNotFound
		}
		return nil, domain.ErrTMDBError
	}

	regions := make([]ports.RegionReleaseDates, len(releaseDates.Results))
	for i, r := range releaseDates.Results {
		dates := make([]ports.ReleaseDateItem, len(r.ReleaseDates))
		for j, d := range r.ReleaseDates {
			dates[j] = ports.ReleaseDateItem{
				Date:          d.ReleaseDate,
				Type:          releaseTypeName(d.Type),
				Certification: d.Certification,
			}
		}
		regions[i] = ports.RegionReleaseDates{
			Region:       r.ISO31661,
			ReleaseDates: dates,
		}
	}

	return &ports.MovieReleaseDatesResult{Regions: regions}, nil
}

func (s *movieService) GetGenres(ctx context.Context, language string) ([]ports.Genre, error) {
	genres, err := s.tmdbClient.GetGenres(ctx, language)
	if err != nil {
		return nil, domain.ErrTMDBError
	}

	result := make([]ports.Genre, len(genres))
	for i, g := range genres {
		result[i] = ports.Genre{
			ID:   g.ID,
			Name: g.Name,
		}
	}

	return result, nil
}

func releaseTypeName(typeCode int) string {
	switch typeCode {
	case 1:
		return "Premiere"
	case 2:
		return "Theatrical (limited)"
	case 3:
		return "Theatrical"
	case 4:
		return "Digital"
	case 5:
		return "Physical"
	case 6:
		return "TV"
	default:
		return "Unknown"
	}
}
