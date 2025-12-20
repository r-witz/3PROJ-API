package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateAccessToken(userID uuid.UUID, role string, secret string, expiry time.Duration) (string, error) {
	now := time.Now()

	claims := AccessTokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "duskforge-api",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", &TokenError{
			Operation: "generate_access",
			Err:       err,
		}
	}

	return signedToken, nil
}

func GenerateRefreshToken(sessionID uuid.UUID, secret string, expiry time.Duration) (string, error) {
	now := time.Now()

	claims := RefreshTokenClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "duskforge-api",
			Subject:   sessionID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", &TokenError{
			Operation: "generate_refresh",
			Err:       err,
		}
	}

	return signedToken, nil
}

func ValidateAccessToken(tokenString, secret string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&AccessTokenClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidSignature
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, mapJWTError(err, "validate_access")
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, &TokenError{
			Operation: "validate_access",
			Err:       ErrInvalidClaims,
		}
	}

	return claims, nil
}

func ValidateRefreshToken(tokenString, secret string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&RefreshTokenClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidSignature
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, mapJWTError(err, "validate_refresh")
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, &TokenError{
			Operation: "validate_refresh",
			Err:       ErrInvalidClaims,
		}
	}

	return claims, nil
}

func mapJWTError(err error, operation string) error {
	switch {
	case errors.Is(err, jwt.ErrTokenMalformed):
		return &TokenError{Operation: operation, Err: ErrTokenMalformed}
	case errors.Is(err, jwt.ErrTokenExpired):
		return &TokenError{Operation: operation, Err: ErrTokenExpired}
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return &TokenError{Operation: operation, Err: ErrInvalidToken}
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return &TokenError{Operation: operation, Err: ErrInvalidSignature}
	default:
		return &TokenError{Operation: operation, Err: err}
	}
}
