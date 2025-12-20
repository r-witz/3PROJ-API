package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"duskforge-api/pkg/logger"

	"go.uber.org/zap"
)

type Client struct {
	apiKey      string
	httpClient  *http.Client
	rateLimiter *RateLimiter
	imageURLs   *ImageURLBuilder
	baseURL     string
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
		return err
	}

	c.imageURLs = NewImageURLBuilder(config.Images.SecureBaseURL)
	logger.Logger.Info("TMDB configuration loaded",
		zap.String("image_base_url", config.Images.SecureBaseURL),
	)

	return nil
}

func (c *Client) ImageURLs() *ImageURLBuilder {
	return c.imageURLs
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

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, &RequestError{Operation: endpoint, Err: err}
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
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

func (c *Client) handleErrorResponse(statusCode int, body []byte, endpoint string) error {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.StatusMessage != "" {
		logger.Logger.Error("TMDB API error",
			zap.String("endpoint", endpoint),
			zap.Int("status_code", statusCode),
			zap.String("message", apiErr.StatusMessage),
		)
		return &apiErr
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusServiceUnavailable:
		return ErrServiceUnavailable
	default:
		return &RequestError{
			Operation: endpoint,
			Err:       fmt.Errorf("unexpected status code: %d", statusCode),
		}
	}
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
	logger.Logger.Info("TMDB client closed")
}
