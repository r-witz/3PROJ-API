package tmdb

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrUnauthorized       = errors.New("invalid or missing API key")
	ErrRateLimited        = errors.New("rate limit exceeded")
	ErrServiceUnavailable = errors.New("TMDB service unavailable")
	ErrInvalidRequest     = errors.New("invalid request parameters")
)

type APIError struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Success       bool   `json:"success"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("TMDB API error (code %d): %s", e.StatusCode, e.StatusMessage)
}

type RequestError struct {
	Operation string
	Err       error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("TMDB request failed [%s]: %v", e.Operation, e.Err)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}
