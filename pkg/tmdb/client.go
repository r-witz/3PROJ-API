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
	"sync/atomic"
	"time"

	"duskforge-api/pkg/logger"

	"go.uber.org/zap"
)

type Client struct {
	apiKey        string
	httpClient    *http.Client
	rateLimiter   *RateLimiter
	imageURLs     *ImageURLBuilder
	baseURL       string
	configReady   atomic.Bool
	retryCancel   context.CancelFunc
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

func New(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		logger.Logger.Error("TMDB API key is required")
		return nil, ErrUnauthorized
	}

	c := &Client{
		apiKey:      apiKey,
		baseURL:     BaseURL,
		rateLimiter: NewRateLimiter(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.imageURLs = NewImageURLBuilder("")

	logger.Logger.Info("TMDB client initialized",
		zap.String("base_url", c.baseURL),
	)

	return c, nil
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
	c.imageURLs = NewImageURLBuilder(config.Images.SecureBaseURL)
	logger.Logger.Info("TMDB configuration loaded",
		zap.String("image_base_url", config.Images.SecureBaseURL),
	)

	return nil
}

func (c *Client) startConfigRetry() {
	ctx, cancel := context.WithCancel(context.Background())
	c.retryCancel = cancel

	go func() {
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
				c.imageURLs = NewImageURLBuilder(config.Images.SecureBaseURL)
				logger.Logger.Info("TMDB configuration loaded after retry",
					zap.String("image_base_url", config.Images.SecureBaseURL),
				)
				return
			}
		}
	}()
}

func (c *Client) ImageURLs() *ImageURLBuilder {
	return c.imageURLs
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

func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	fullURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if params == nil {
		params = url.Values{}
	}
	params.Set("api_key", c.apiKey)
	fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())

	const maxRetries = 3
	var lastErr error

	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			return nil, &RequestError{Operation: endpoint, Err: err}
		}

		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if isTransientError(err) && attempt < maxRetries-1 {
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
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, &RequestError{Operation: endpoint, Err: err}
		}

		if resp.StatusCode >= 400 {
			return nil, c.handleErrorResponse(resp.StatusCode, body, endpoint)
		}

		return body, nil
	}

	return nil, &RequestError{Operation: endpoint, Err: lastErr}
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
