package domain

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal error")
)

var (
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrEmailAlreadyExists    = errors.New("email already registered")
	ErrUsernameAlreadyExists = errors.New("username already taken")
	ErrInvalidToken          = errors.New("invalid or expired token")
	ErrSessionExpired        = errors.New("session has expired")
	ErrUserBanned            = errors.New("user account is banned")
)

var (
	ErrUserNotFound = errors.New("user not found")
)

var (
	ErrMovieNotFound = errors.New("movie not found")
	ErrTMDBError     = errors.New("external movie service error")
)
