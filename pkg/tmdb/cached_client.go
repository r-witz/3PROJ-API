package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"duskforge-api/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const cachePrefix = "tmdb:"

const (
	ttlConfiguration    = 7 * 24 * time.Hour
	ttlMovieMetadata    = 24 * time.Hour
	ttlPersonCredits    = 12 * time.Hour
	ttlSearch           = 1 * time.Hour
	ttlMovieListings    = 30 * time.Minute
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

func cacheGet[T any](ctx context.Context, c *CachedClient, key string, dest *T) bool {
	data, err := c.redis.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
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

func withCache[T any](ctx context.Context, c *CachedClient, key string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	var cached T
	if cacheGet(ctx, c, key, &cached) {
		return cached, nil
	}
	result, err := fetch()
	if err != nil {
		return result, err
	}
	cacheSet(ctx, c, key, result, ttl)
	return result, nil
}

func (c *CachedClient) GetConfiguration(ctx context.Context) (*Configuration, error) {
	return withCache(ctx, c, cachePrefix+"configuration", ttlConfiguration, func() (*Configuration, error) {
		return c.client.GetConfiguration(ctx)
	})
}

func (c *CachedClient) GetGenres(ctx context.Context, language string) ([]Genre, error) {
	key := fmt.Sprintf(cachePrefix+"genres:%s", language)
	return withCache(ctx, c, key, ttlMovieMetadata, func() ([]Genre, error) {
		return c.client.GetGenres(ctx, language)
	})
}

func (c *CachedClient) GetMovieDetails(ctx context.Context, movieID int, language string) (*MovieDetails, error) {
	key := fmt.Sprintf(cachePrefix+"movie:%d:%s", movieID, language)
	return withCache(ctx, c, key, ttlMovieMetadata, func() (*MovieDetails, error) {
		return c.client.GetMovieDetails(ctx, movieID, language)
	})
}

func (c *CachedClient) GetMovieCredits(ctx context.Context, movieID int, language string) (*Credits, error) {
	key := fmt.Sprintf(cachePrefix+"movie_credits:%d:%s", movieID, language)
	return withCache(ctx, c, key, ttlMovieMetadata, func() (*Credits, error) {
		return c.client.GetMovieCredits(ctx, movieID, language)
	})
}

func (c *CachedClient) GetMovieWithCredits(ctx context.Context, movieID int, language string) (*MovieDetails, *Credits, error) {
	detailsKey := fmt.Sprintf(cachePrefix+"movie:%d:%s", movieID, language)
	creditsKey := fmt.Sprintf(cachePrefix+"movie_credits:%d:%s", movieID, language)

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
		cacheSet(ctx, c, detailsKey, movie, ttlMovieMetadata)
	}
	if credits != nil {
		cacheSet(ctx, c, creditsKey, credits, ttlMovieMetadata)
	}
	return movie, credits, nil
}

func (c *CachedClient) GetMovieVideos(ctx context.Context, movieID int, language string) (*VideosResponse, error) {
	key := fmt.Sprintf(cachePrefix+"movie_videos:%d:%s", movieID, language)
	return withCache(ctx, c, key, ttlMovieMetadata, func() (*VideosResponse, error) {
		return c.client.GetMovieVideos(ctx, movieID, language)
	})
}

func (c *CachedClient) GetMovieReleaseDates(ctx context.Context, movieID int) (*ReleaseDatesResponse, error) {
	key := fmt.Sprintf(cachePrefix+"movie_release_dates:%d", movieID)
	return withCache(ctx, c, key, ttlMovieMetadata, func() (*ReleaseDatesResponse, error) {
		return c.client.GetMovieReleaseDates(ctx, movieID)
	})
}

func (c *CachedClient) GetPersonDetails(ctx context.Context, personID int, language string) (*PersonDetails, error) {
	key := fmt.Sprintf(cachePrefix+"person:%d:%s", personID, language)
	return withCache(ctx, c, key, ttlMovieMetadata, func() (*PersonDetails, error) {
		return c.client.GetPersonDetails(ctx, personID, language)
	})
}

func (c *CachedClient) GetPersonMovieCredits(ctx context.Context, personID int, language string) (*PersonMovieCredits, error) {
	key := fmt.Sprintf(cachePrefix+"person_credits:%d:%s", personID, language)
	return withCache(ctx, c, key, ttlPersonCredits, func() (*PersonMovieCredits, error) {
		return c.client.GetPersonMovieCredits(ctx, personID, language)
	})
}

func (c *CachedClient) SearchMovies(ctx context.Context, params SearchMoviesParams) (*SearchMoviesResponse, error) {
	key := fmt.Sprintf(cachePrefix+"search_movies:%s:%d:%s:%d:%d:%s:%t",
		params.Query, params.Page, params.Language,
		params.PrimaryReleaseYear, params.Year, params.Region, params.IncludeAdult)
	return withCache(ctx, c, key, ttlSearch, func() (*SearchMoviesResponse, error) {
		return c.client.SearchMovies(ctx, params)
	})
}

func (c *CachedClient) DiscoverMovies(ctx context.Context, params DiscoverMoviesParams) (*DiscoverMoviesResponse, error) {
	key := buildDiscoverKey(params)
	return withCache(ctx, c, key, ttlSearch, func() (*DiscoverMoviesResponse, error) {
		return c.client.DiscoverMovies(ctx, params)
	})
}

func (c *CachedClient) GetPopularMovies(ctx context.Context, page int, language, region string) (*PopularMoviesResponse, error) {
	key := fmt.Sprintf(cachePrefix+"popular:%d:%s:%s", page, language, region)
	return withCache(ctx, c, key, ttlMovieListings, func() (*PopularMoviesResponse, error) {
		return c.client.GetPopularMovies(ctx, page, language, region)
	})
}

func (c *CachedClient) GetTrendingMovies(ctx context.Context, page int, language string) (*TrendingMoviesResponse, error) {
	key := fmt.Sprintf(cachePrefix+"trending:%d:%s", page, language)
	return withCache(ctx, c, key, ttlMovieListings, func() (*TrendingMoviesResponse, error) {
		return c.client.GetTrendingMovies(ctx, page, language)
	})
}

func (c *CachedClient) GetUpcomingMovies(ctx context.Context, page int, language, region string) (*UpcomingMoviesResponse, error) {
	key := fmt.Sprintf(cachePrefix+"upcoming:%d:%s:%s", page, language, region)
	return withCache(ctx, c, key, ttlMovieListings, func() (*UpcomingMoviesResponse, error) {
		return c.client.GetUpcomingMovies(ctx, page, language, region)
	})
}

func (c *CachedClient) SearchPerson(ctx context.Context, params SearchPersonParams) (*SearchPersonResponse, error) {
	key := fmt.Sprintf(cachePrefix+"search_person:%s:%d:%s", params.Query, params.Page, params.Language)
	return withCache(ctx, c, key, ttlSearch, func() (*SearchPersonResponse, error) {
		return c.client.SearchPerson(ctx, params)
	})
}

func (c *CachedClient) ImageURLs() *ImageURLBuilder {
	return c.client.ImageURLs()
}

func buildDiscoverKey(params DiscoverMoviesParams) string {
	kv := map[string]string{
		"page":     strconv.Itoa(params.Page),
		"lang":     params.Language,
		"sort":     string(params.SortBy),
		"adult":    strconv.FormatBool(params.IncludeAdult),
		"video":    strconv.FormatBool(params.IncludeVideo),
		"pryear":   strconv.Itoa(params.PrimaryReleaseYear),
		"prgte":    params.PrimaryReleaseDateGTE,
		"prlte":    params.PrimaryReleaseDateLTE,
		"vgte":     fmt.Sprintf("%.3f", params.VoteAverageGTE),
		"vlte":     fmt.Sprintf("%.3f", params.VoteAverageLTE),
		"vcgte":    strconv.Itoa(params.VoteCountGTE),
		"rtgte":    strconv.Itoa(params.WithRuntimeGTE),
		"rtlte":    strconv.Itoa(params.WithRuntimeLTE),
		"olang":    params.WithOriginalLanguage,
		"region":   params.Region,
		"year":     strconv.Itoa(params.Year),
		"genres":   joinInts(params.WithGenres),
		"xgenres":  joinInts(params.WithoutGenres),
		"cast":     joinInts(params.WithCast),
	}
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{cachePrefix + "discover"}
	for _, k := range keys {
		parts = append(parts, k+"="+kv[k])
	}
	return strings.Join(parts, ":")
}

func joinInts(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	s := make([]string, len(ints))
	for i, v := range ints {
		s[i] = strconv.Itoa(v)
	}
	return strings.Join(s, ",")
}
