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
	ErrUserNotFound    = errors.New("user not found")
	ErrNoPasswordSet   = errors.New("no password set for this account")
	ErrIncorrectPassword = errors.New("current password is incorrect")
)

var (
	ErrMovieNotFound = errors.New("movie not found")
	ErrTMDBError     = errors.New("external movie service error")
)

var (
	ErrOAuthAccountNotFound      = errors.New("oauth account not found")
	ErrOAuthAccountAlreadyLinked = errors.New("oauth account already linked to another user")
	ErrOAuthProviderNotSupported = errors.New("oauth provider not supported")
	ErrOAuthStateMismatch        = errors.New("invalid oauth state")
	ErrCannotUnlinkOnlyAuth      = errors.New("cannot unlink the only authentication method")
)
