package oauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"
)

var (
	ErrInvalidState = errors.New("invalid oauth state")
	ErrExpiredState = errors.New("oauth state expired")
)

// StateManager handles generation and validation of OAuth state tokens for CSRF protection
type StateManager struct {
	secret []byte
	expiry time.Duration
}

// NewStateManager creates a new state manager with the given secret and expiry duration
func NewStateManager(secret string, expiry time.Duration) *StateManager {
	return &StateManager{
		secret: []byte(secret),
		expiry: expiry,
	}
}

// Generate creates a new signed state token containing a timestamp and random bytes
func (m *StateManager) Generate() (string, error) {
	// Create payload: 8 bytes timestamp + 16 bytes random
	payload := make([]byte, 24)

	// Add timestamp (Unix seconds)
	timestamp := time.Now().Unix()
	binary.BigEndian.PutUint64(payload[:8], uint64(timestamp))

	// Add random bytes
	if _, err := rand.Read(payload[8:]); err != nil {
		return "", err
	}

	// Create HMAC signature
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	signature := mac.Sum(nil)

	// Combine payload + signature and encode
	state := append(payload, signature...)
	return base64.URLEncoding.EncodeToString(state), nil
}

// Validate verifies the state token signature and checks expiration
func (m *StateManager) Validate(state string) error {
	// Decode state
	data, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return ErrInvalidState
	}

	// State should be 24 bytes payload + 32 bytes HMAC-SHA256 signature
	if len(data) != 56 {
		return ErrInvalidState
	}

	payload := data[:24]
	providedSignature := data[24:]

	// Verify signature
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return ErrInvalidState
	}

	// Check expiration
	timestamp := int64(binary.BigEndian.Uint64(payload[:8]))
	createdAt := time.Unix(timestamp, 0)

	if time.Since(createdAt) > m.expiry {
		return ErrExpiredState
	}

	return nil
}
