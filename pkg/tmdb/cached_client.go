package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"duskforge-api/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type CachedClient struct {
	client *Client
	redis  *redis.Client
}

func NewCachedClient(client *Client, redisClient *redis.Client) *CachedClient {
	return &CachedClient{
		client: client,
		redis:  redisClient,
	}
}

// cacheGet tries to retrieve a cached value. Returns false on miss or error.
func cacheGet[T any](ctx context.Context, c *CachedClient, key string, dest *T) bool {
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			logger.Logger.Warn("redis GET error", zap.String("key", key), zap.Error(err))
		}
		return false
	}
	if err := json.Unmarshal(data, dest); err != nil {
		logger.Logger.Warn("redis unmarshal error", zap.String("key", key), zap.Error(err))
		return false
	}
	logger.Logger.Debug("cache hit", zap.String("key", key))
	return true
}

// cacheSet stores a value in Redis. Errors are logged but not returned.
func cacheSet(ctx context.Context, c *CachedClient, key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		logger.Logger.Warn("redis marshal error", zap.String("key", key), zap.Error(err))
		return
	}
	if err := c.redis.Set(ctx, key, data, ttl).Err(); err != nil {
		logger.Logger.Warn("redis SET error", zap.String("key", key), zap.Error(err))
	}
}

func (c *CachedClient) GetConfiguration(ctx context.Context) (*Configuration, error) {
	key := "tmdb:configuration"
	var cached Configuration
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 7*24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetGenres(ctx context.Context, language string) ([]Genre, error) {
	key := fmt.Sprintf("tmdb:genres:%s", language)
	var cached []Genre
	if cacheGet(ctx, c, key, &cached) {
		return cached, nil
	}
	result, err := c.client.GetGenres(ctx, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetMovieDetails(ctx context.Context, movieID int, language string) (*MovieDetails, error) {
	key := fmt.Sprintf("tmdb:movie:%d:%s", movieID, language)
	var cached MovieDetails
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetMovieDetails(ctx, movieID, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetMovieCredits(ctx context.Context, movieID int, language string) (*Credits, error) {
	key := fmt.Sprintf("tmdb:movie_credits:%d:%s", movieID, language)
	var cached Credits
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetMovieCredits(ctx, movieID, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetMovieWithCredits(ctx context.Context, movieID int, language string) (*MovieDetails, *Credits, error) {
	detailsKey := fmt.Sprintf("tmdb:movie:%d:%s", movieID, language)
	creditsKey := fmt.Sprintf("tmdb:movie_credits:%d:%s", movieID, language)

	var cachedDetails MovieDetails
	var cachedCredits Credits
	detailsHit := cacheGet(ctx, c, detailsKey, &cachedDetails)
	creditsHit := cacheGet(ctx, c, creditsKey, &cachedCredits)

	if detailsHit && creditsHit {
		return &cachedDetails, &cachedCredits, nil
	}

	movie, credits, err := c.client.GetMovieWithCredits(ctx, movieID, language)
	if err != nil {
		return movie, credits, err
	}
	if movie != nil {
		cacheSet(ctx, c, detailsKey, movie, 24*time.Hour)
	}
	if credits != nil {
		cacheSet(ctx, c, creditsKey, credits, 24*time.Hour)
	}
	return movie, credits, nil
}

func (c *CachedClient) GetMovieVideos(ctx context.Context, movieID int, language string) (*VideosResponse, error) {
	key := fmt.Sprintf("tmdb:movie_videos:%d:%s", movieID, language)
	var cached VideosResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetMovieVideos(ctx, movieID, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetMovieReleaseDates(ctx context.Context, movieID int) (*ReleaseDatesResponse, error) {
	key := fmt.Sprintf("tmdb:movie_release_dates:%d", movieID)
	var cached ReleaseDatesResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetMovieReleaseDates(ctx, movieID)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetPersonDetails(ctx context.Context, personID int, language string) (*PersonDetails, error) {
	key := fmt.Sprintf("tmdb:person:%d:%s", personID, language)
	var cached PersonDetails
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetPersonDetails(ctx, personID, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 24*time.Hour)
	return result, nil
}

func (c *CachedClient) GetPersonMovieCredits(ctx context.Context, personID int, language string) (*PersonMovieCredits, error) {
	key := fmt.Sprintf("tmdb:person_credits:%d:%s", personID, language)
	var cached PersonMovieCredits
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetPersonMovieCredits(ctx, personID, language)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 12*time.Hour)
	return result, nil
}

func (c *CachedClient) SearchMovies(ctx context.Context, params SearchMoviesParams) (*SearchMoviesResponse, error) {
	key := fmt.Sprintf("tmdb:search_movies:%s:%d:%s", params.Query, params.Page, params.Language)
	var cached SearchMoviesResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.SearchMovies(ctx, params)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 1*time.Hour)
	return result, nil
}

func (c *CachedClient) DiscoverMovies(ctx context.Context, params DiscoverMoviesParams) (*DiscoverMoviesResponse, error) {
	key := buildDiscoverKey(params)
	var cached DiscoverMoviesResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.DiscoverMovies(ctx, params)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 1*time.Hour)
	return result, nil
}

func (c *CachedClient) GetPopularMovies(ctx context.Context, page int, language, region string) (*PopularMoviesResponse, error) {
	key := fmt.Sprintf("tmdb:popular:%d:%s:%s", page, language, region)
	var cached PopularMoviesResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.GetPopularMovies(ctx, page, language, region)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 30*time.Minute)
	return result, nil
}

func (c *CachedClient) SearchPerson(ctx context.Context, params SearchPersonParams) (*SearchPersonResponse, error) {
	key := fmt.Sprintf("tmdb:search_person:%s:%d:%s", params.Query, params.Page, params.Language)
	var cached SearchPersonResponse
	if cacheGet(ctx, c, key, &cached) {
		return &cached, nil
	}
	result, err := c.client.SearchPerson(ctx, params)
	if err != nil {
		return nil, err
	}
	cacheSet(ctx, c, key, result, 1*time.Hour)
	return result, nil
}

func (c *CachedClient) ImageURLs() *ImageURLBuilder {
	return c.client.ImageURLs()
}

func buildDiscoverKey(params DiscoverMoviesParams) string {
	parts := []string{"tmdb:discover"}
	parts = append(parts, strconv.Itoa(params.Page))
	parts = append(parts, params.Language)
	parts = append(parts, string(params.SortBy))
	if params.PrimaryReleaseYear > 0 {
		parts = append(parts, strconv.Itoa(params.PrimaryReleaseYear))
	}
	if len(params.WithGenres) > 0 {
		ids := make([]string, len(params.WithGenres))
		for i, id := range params.WithGenres {
			ids[i] = strconv.Itoa(id)
		}
		parts = append(parts, "g:"+strings.Join(ids, ","))
	}
	if len(params.WithCast) > 0 {
		ids := make([]string, len(params.WithCast))
		for i, id := range params.WithCast {
			ids[i] = strconv.Itoa(id)
		}
		parts = append(parts, "c:"+strings.Join(ids, ","))
	}
	if params.VoteAverageGTE > 0 {
		parts = append(parts, fmt.Sprintf("vgte:%.1f", params.VoteAverageGTE))
	}
	if params.VoteAverageLTE > 0 {
		parts = append(parts, fmt.Sprintf("vlte:%.1f", params.VoteAverageLTE))
	}
	if params.WithOriginalLanguage != "" {
		parts = append(parts, "ol:"+params.WithOriginalLanguage)
	}
	if params.Region != "" {
		parts = append(parts, "r:"+params.Region)
	}
	return strings.Join(parts, ":")
}
