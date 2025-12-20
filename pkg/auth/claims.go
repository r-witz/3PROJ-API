package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessTokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	SessionID uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}
