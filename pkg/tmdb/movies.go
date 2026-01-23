package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) GetConfiguration(ctx context.Context) (*Configuration, error) {
	body, err := c.doRequest(ctx, "GET", "/configuration", nil)
	if err != nil {
		return nil, err
	}

	var config Configuration
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, &RequestError{Operation: "/configuration", Err: err}
	}

	return &config, nil
}

func (c *Client) GetGenres(ctx context.Context, language string) ([]Genre, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	body, err := c.doRequest(ctx, "GET", "/genre/movie/list", params)
	if err != nil {
		return nil, err
	}

	var resp GenreListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: "/genre/movie/list", Err: err}
	}

	return resp.Genres, nil
}

type SearchMoviesParams struct {
	Query              string
	Page               int
	Year               int
	Language           string
	IncludeAdult       bool
	Region             string
	PrimaryReleaseYear int
}

func (c *Client) SearchMovies(ctx context.Context, params SearchMoviesParams) (*SearchMoviesResponse, error) {
	if params.Query == "" {
		return nil, ErrInvalidRequest
	}

	queryParams := url.Values{}
	queryParams.Set("query", params.Query)

	if params.Page > 0 {
		queryParams.Set("page", strconv.Itoa(params.Page))
	}
	if params.Year > 0 {
		queryParams.Set("year", strconv.Itoa(params.Year))
	}
	if params.Language != "" {
		queryParams.Set("language", params.Language)
	}
	if params.IncludeAdult {
		queryParams.Set("include_adult", "true")
	}
	if params.Region != "" {
		queryParams.Set("region", params.Region)
	}
	if params.PrimaryReleaseYear > 0 {
		queryParams.Set("primary_release_year", strconv.Itoa(params.PrimaryReleaseYear))
	}

	body, err := c.doRequest(ctx, "GET", "/search/movie", queryParams)
	if err != nil {
		return nil, err
	}

	var resp SearchMoviesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: "/search/movie", Err: err}
	}

	return &resp, nil
}

func (c *Client) GetMovieDetails(ctx context.Context, movieID int, language string) (*MovieDetails, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	endpoint := fmt.Sprintf("/movie/%d", movieID)
	body, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}

	var movie MovieDetails
	if err := json.Unmarshal(body, &movie); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &movie, nil
}

func (c *Client) GetMovieCredits(ctx context.Context, movieID int, language string) (*Credits, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	endpoint := fmt.Sprintf("/movie/%d/credits", movieID)
	body, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}

	var credits Credits
	if err := json.Unmarshal(body, &credits); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &credits, nil
}

type DiscoverMoviesParams struct {
	Page                  int
	Language              string
	SortBy                SortBy
	IncludeAdult          bool
	IncludeVideo          bool
	PrimaryReleaseYear    int
	PrimaryReleaseDateGTE string
	PrimaryReleaseDateLTE string
	WithGenres            []int
	WithoutGenres         []int
	WithCast              []int
	VoteAverageGTE        float64
	VoteAverageLTE        float64
	VoteCountGTE          int
	WithRuntimeGTE        int
	WithRuntimeLTE        int
	WithOriginalLanguage  string
	Region                string
	Year                  int
}

func (c *Client) DiscoverMovies(ctx context.Context, params DiscoverMoviesParams) (*DiscoverMoviesResponse, error) {
	queryParams := url.Values{}

	if params.Page > 0 {
		queryParams.Set("page", strconv.Itoa(params.Page))
	}
	if params.Language != "" {
		queryParams.Set("language", params.Language)
	}
	if params.SortBy != "" {
		queryParams.Set("sort_by", string(params.SortBy))
	}
	if params.IncludeAdult {
		queryParams.Set("include_adult", "true")
	}
	if params.IncludeVideo {
		queryParams.Set("include_video", "true")
	}
	if params.PrimaryReleaseYear > 0 {
		queryParams.Set("primary_release_year", strconv.Itoa(params.PrimaryReleaseYear))
	}
	if params.PrimaryReleaseDateGTE != "" {
		queryParams.Set("primary_release_date.gte", params.PrimaryReleaseDateGTE)
	}
	if params.PrimaryReleaseDateLTE != "" {
		queryParams.Set("primary_release_date.lte", params.PrimaryReleaseDateLTE)
	}
	if len(params.WithGenres) > 0 {
		ids := make([]string, len(params.WithGenres))
		for i, id := range params.WithGenres {
			ids[i] = strconv.Itoa(id)
		}
		queryParams.Set("with_genres", strings.Join(ids, ","))
	}
	if len(params.WithoutGenres) > 0 {
		ids := make([]string, len(params.WithoutGenres))
		for i, id := range params.WithoutGenres {
			ids[i] = strconv.Itoa(id)
		}
		queryParams.Set("without_genres", strings.Join(ids, ","))
	}
	if len(params.WithCast) > 0 {
		ids := make([]string, len(params.WithCast))
		for i, id := range params.WithCast {
			ids[i] = strconv.Itoa(id)
		}
		queryParams.Set("with_cast", strings.Join(ids, ","))
	}
	if params.VoteAverageGTE > 0 {
		queryParams.Set("vote_average.gte", strconv.FormatFloat(params.VoteAverageGTE, 'f', 1, 64))
	}
	if params.VoteAverageLTE > 0 {
		queryParams.Set("vote_average.lte", strconv.FormatFloat(params.VoteAverageLTE, 'f', 1, 64))
	}
	if params.VoteCountGTE > 0 {
		queryParams.Set("vote_count.gte", strconv.Itoa(params.VoteCountGTE))
	}
	if params.WithRuntimeGTE > 0 {
		queryParams.Set("with_runtime.gte", strconv.Itoa(params.WithRuntimeGTE))
	}
	if params.WithRuntimeLTE > 0 {
		queryParams.Set("with_runtime.lte", strconv.Itoa(params.WithRuntimeLTE))
	}
	if params.WithOriginalLanguage != "" {
		queryParams.Set("with_original_language", params.WithOriginalLanguage)
	}
	if params.Region != "" {
		queryParams.Set("region", params.Region)
	}
	if params.Year > 0 {
		queryParams.Set("year", strconv.Itoa(params.Year))
	}

	body, err := c.doRequest(ctx, "GET", "/discover/movie", queryParams)
	if err != nil {
		return nil, err
	}

	var resp DiscoverMoviesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: "/discover/movie", Err: err}
	}

	return &resp, nil
}

func (c *Client) GetPopularMovies(ctx context.Context, page int, language, region string) (*PopularMoviesResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if language != "" {
		params.Set("language", language)
	}
	if region != "" {
		params.Set("region", region)
	}

	body, err := c.doRequest(ctx, "GET", "/movie/popular", params)
	if err != nil {
		return nil, err
	}

	var resp PopularMoviesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: "/movie/popular", Err: err}
	}

	return &resp, nil
}

func (c *Client) GetMovieWithCredits(ctx context.Context, movieID int, language string) (*MovieDetails, *Credits, error) {
	type detailsResult struct {
		movie *MovieDetails
		err   error
	}
	type creditsResult struct {
		credits *Credits
		err     error
	}

	detailsCh := make(chan detailsResult, 1)
	creditsCh := make(chan creditsResult, 1)

	go func() {
		movie, err := c.GetMovieDetails(ctx, movieID, language)
		detailsCh <- detailsResult{movie: movie, err: err}
	}()

	go func() {
		credits, err := c.GetMovieCredits(ctx, movieID, language)
		creditsCh <- creditsResult{credits: credits, err: err}
	}()

	detailsRes := <-detailsCh
	creditsRes := <-creditsCh

	if detailsRes.err != nil {
		return nil, nil, detailsRes.err
	}
	if creditsRes.err != nil {
		return detailsRes.movie, nil, creditsRes.err
	}

	return detailsRes.movie, creditsRes.credits, nil
}

func (c *Client) GetMovieVideos(ctx context.Context, movieID int, language string) (*VideosResponse, error) {
	params := url.Values{}
	if language != "" {
		params.Set("language", language)
	}

	endpoint := fmt.Sprintf("/movie/%d/videos", movieID)
	body, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}

	var resp VideosResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &resp, nil
}

func (c *Client) GetMovieReleaseDates(ctx context.Context, movieID int) (*ReleaseDatesResponse, error) {
	endpoint := fmt.Sprintf("/movie/%d/release_dates", movieID)
	body, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp ReleaseDatesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	return &resp, nil
}
