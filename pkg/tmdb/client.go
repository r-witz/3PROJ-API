package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"duskforge-api/pkg/logger"

	"go.uber.org/zap"
)

type Client struct {
	apiKeys     []string
	keyIndex    atomic.Uint64
	httpClient  *http.Client
	imageURLs   atomic.Pointer[ImageURLBuilder]
	baseURL     string
	configReady atomic.Bool
	retryCancel context.CancelFunc
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func New(apiKeys string, opts ...ClientOption) (*Client, error) {
	keys := parseAPIKeys(apiKeys)
	if len(keys) == 0 {
		logger.Logger.Error("TMDB API key is required")
		return nil, ErrUnauthorized
	}

	c := &Client{
		apiKeys: keys,
		baseURL: BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.imageURLs.Store(NewImageURLBuilder(""))

	logger.Logger.Info("TMDB client initialized",
		zap.String("base_url", c.baseURL),
		zap.Int("api_key_count", len(keys)),
	)

	return c, nil
}

func parseAPIKeys(raw string) []string {
	var keys []string
	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// nextKey returns the next API key using round-robin.
func (c *Client) nextKey() string {
	idx := c.keyIndex.Add(1) - 1
	return c.apiKeys[idx%uint64(len(c.apiKeys))]
}

func (c *Client) InitializeConfiguration(ctx context.Context) error {
	config, err := c.GetConfiguration(ctx)
	if err != nil {
		logger.Logger.Warn("failed to fetch TMDB configuration, using defaults",
			zap.Error(err),
		)
		c.startConfigRetry()
		return err
	}

	c.configReady.Store(true)
	c.imageURLs.Store(NewImageURLBuilder(config.Images.SecureBaseURL))
	logger.Logger.Info("TMDB configuration loaded",
		zap.String("image_base_url", config.Images.SecureBaseURL),
	)

	return nil
}

func (c *Client) startConfigRetry() {
	ctx, cancel := context.WithCancel(context.Background())
	c.retryCancel = cancel

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Error("tmdb-config-retry panic", zap.Any("panic", r))
			}
		}()
		delay := 10 * time.Second
		maxDelay := 5 * time.Minute

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				config, err := c.GetConfiguration(ctx)
				if err != nil {
					logger.Logger.Warn("TMDB config retry failed, will retry",
						zap.Duration("next_retry", delay),
						zap.Error(err),
					)
					delay = min(delay*2, maxDelay)
					continue
				}

				c.configReady.Store(true)
				c.imageURLs.Store(NewImageURLBuilder(config.Images.SecureBaseURL))
				logger.Logger.Info("TMDB configuration loaded after retry",
					zap.String("image_base_url", config.Images.SecureBaseURL),
				)
				return
			}
		}
	}()
}

func (c *Client) ImageURLs() *ImageURLBuilder {
	return c.imageURLs.Load()
}

func isTransientError(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func (c *Client) buildURL(endpoint, apiKey string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", apiKey)
	return fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, params.Encode())
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	// Try up to len(apiKeys) different keys on 429, then one final wait+retry.
	maxAttempts := len(c.apiKeys) + 1
	var lastErr error
	var retryAfter time.Duration

	for attempt := range maxAttempts {
		apiKey := c.nextKey()
		fullURL := c.buildURL(endpoint, apiKey, params)

		// If the previous attempt got a 429 and we've exhausted all keys, wait before retrying.
		if attempt == len(c.apiKeys) && retryAfter > 0 {
			logger.Logger.Warn("TMDB all keys rate limited, waiting",
				zap.String("endpoint", endpoint),
				zap.Duration("wait", retryAfter),
			)
			select {
			case <-ctx.Done():
				return nil, &RequestError{Operation: endpoint, Err: ctx.Err()}
			case <-time.After(retryAfter):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			return nil, &RequestError{Operation: endpoint, Err: err}
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if isTransientError(err) {
				logger.Logger.Warn("TMDB request failed, retrying",
					zap.String("endpoint", endpoint),
					zap.Int("attempt", attempt+1),
					zap.Error(err),
				)
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			logger.Logger.Error("TMDB request failed",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
			return nil, &RequestError{Operation: endpoint, Err: err}
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, &RequestError{Operation: endpoint, Err: err}
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts-1 {
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
			logger.Logger.Warn("TMDB rate limited, rotating key",
				zap.String("endpoint", endpoint),
				zap.Int("attempt", attempt+1),
			)
			continue
		}

		if resp.StatusCode >= 400 {
			return nil, c.handleErrorResponse(resp.StatusCode, body, endpoint)
		}

		return body, nil
	}

	return nil, &RequestError{Operation: endpoint, Err: lastErr}
}

// parseRetryAfter parses the Retry-After header value (in seconds).
// Falls back to 1 second if missing or unparseable.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 1 * time.Second
	}
	seconds, err := strconv.Atoi(header)
	if err != nil || seconds <= 0 {
		return 1 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func (c *Client) handleErrorResponse(statusCode int, body []byte, endpoint string) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusMessage != "" {
		logger.Logger.Error("TMDB API error",
			zap.String("endpoint", endpoint),
			zap.Int("status_code", statusCode),
			zap.String("message", apiErr.StatusMessage),
		)
		return &apiErr
	}

	return &RequestError{
		Operation: endpoint,
		Err:       fmt.Errorf("unexpected status code: %d", statusCode),
	}
}

func (c *Client) Close() {
	if c.retryCancel != nil {
		c.retryCancel()
	}
	c.httpClient.CloseIdleConnections()
	logger.Logger.Info("TMDB client closed")
}
