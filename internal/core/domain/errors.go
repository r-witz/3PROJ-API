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
	ErrEmailRequired         = errors.New("email is required")
	ErrUsernameAlreadyExists = errors.New("username already taken")
	ErrUsernameRequired      = errors.New("username is required")
	ErrUsernameTooShort      = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong       = errors.New("username must be at most 50 characters")
	ErrInvalidEmailFormat    = errors.New("invalid email format")
	ErrInvalidToken          = errors.New("invalid or expired token")
	ErrSessionExpired        = errors.New("session has expired")
	ErrUserBanned            = errors.New("user account is banned")
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrNoPasswordSet     = errors.New("no password set for this account")
	ErrIncorrectPassword = errors.New("current password is incorrect")
)

var (
	ErrPasswordRequired      = errors.New("password is required")
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong       = errors.New("password must be at most 72 characters")
	ErrPasswordNoUppercase   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit       = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecialChar = errors.New("password must contain at least one special character")
)

var (
	ErrMovieNotFound = errors.New("movie not found")
	ErrActorNotFound = errors.New("actor not found")
	ErrTMDBError     = errors.New("external movie service error")
)

var (
	ErrOAuthAccountNotFound      = errors.New("oauth account not found")
	ErrOAuthAccountAlreadyLinked = errors.New("oauth account already linked to another user")
	ErrOAuthProviderNotSupported = errors.New("oauth provider not supported")
	ErrOAuthStateMismatch        = errors.New("invalid oauth state")
	ErrCannotUnlinkOnlyAuth      = errors.New("cannot unlink the only authentication method")
)

var (
	ErrCollectionNotFound            = errors.New("collection not found")
	ErrCollectionAlreadyExists       = errors.New("collection already exists")
	ErrCannotModifySystemCollection  = errors.New("cannot modify system collection")
	ErrCannotDeleteSystemCollection  = errors.New("cannot delete system collection")
	ErrCollectionItemAlreadyExists   = errors.New("item already exists in collection")
	ErrCollectionItemNotFound        = errors.New("collection item not found")
)
