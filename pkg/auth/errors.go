package auth

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token has expired")
	ErrTokenMalformed   = errors.New("malformed token")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrInvalidClaims    = errors.New("invalid token claims")
	ErrPasswordTooShort      = errors.New("password too short")
	ErrPasswordTooLong       = errors.New("password too long")
	ErrPasswordNoUppercase   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit       = errors.New("password must contain at least one digit")
	ErrPasswordNoSpecialChar = errors.New("password must contain at least one special character")
	ErrHashingFailed         = errors.New("failed to hash password")
)

type TokenError struct {
	Operation string
	Err       error
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("token error [%s]: %v", e.Operation, e.Err)
}

func (e *TokenError) Unwrap() error {
	return e.Err
}

type PasswordError struct {
	Operation string
	Err       error
}

func (e *PasswordError) Error() string {
	return fmt.Sprintf("password error [%s]: %v", e.Operation, e.Err)
}

func (e *PasswordError) Unwrap() error {
	return e.Err
}
