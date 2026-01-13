package oauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
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

// StateData contains the data encoded in the state token
type StateData struct {
	Timestamp   int64  `json:"t"`
	Nonce       string `json:"n"`
	RedirectURI string `json:"r,omitempty"`
}

// NewStateManager creates a new state manager with the given secret and expiry duration
func NewStateManager(secret string, expiry time.Duration) *StateManager {
	return &StateManager{
		secret: []byte(secret),
		expiry: expiry,
	}
}

// Generate creates a new signed state token containing a timestamp, random nonce, and optional redirect URI
func (m *StateManager) Generate(redirectURI string) (string, error) {
	// Generate random nonce
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}

	// Create state data
	data := StateData{
		Timestamp:   time.Now().Unix(),
		Nonce:       base64.URLEncoding.EncodeToString(nonceBytes),
		RedirectURI: redirectURI,
	}

	// Serialize to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	// Create HMAC signature
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	signature := mac.Sum(nil)

	// Combine: 4 bytes payload length + payload + signature
	result := make([]byte, 4+len(payload)+len(signature))
	binary.BigEndian.PutUint32(result[:4], uint32(len(payload)))
	copy(result[4:4+len(payload)], payload)
	copy(result[4+len(payload):], signature)

	return base64.URLEncoding.EncodeToString(result), nil
}

// Validate verifies the state token signature and checks expiration, returns the state data
func (m *StateManager) Validate(state string) (*StateData, error) {
	// Decode state
	data, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return nil, ErrInvalidState
	}

	// Minimum size: 4 bytes length + 1 byte payload + 32 bytes signature
	if len(data) < 37 {
		return nil, ErrInvalidState
	}

	// Extract payload length
	payloadLen := binary.BigEndian.Uint32(data[:4])
	if int(payloadLen) > len(data)-36 {
		return nil, ErrInvalidState
	}

	payload := data[4 : 4+payloadLen]
	providedSignature := data[4+payloadLen:]

	// Verify signature (should be 32 bytes for SHA256)
	if len(providedSignature) != 32 {
		return nil, ErrInvalidState
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return nil, ErrInvalidState
	}

	// Parse payload
	var stateData StateData
	if err := json.Unmarshal(payload, &stateData); err != nil {
		return nil, ErrInvalidState
	}

	// Check expiration
	createdAt := time.Unix(stateData.Timestamp, 0)
	if time.Since(createdAt) > m.expiry {
		return nil, ErrExpiredState
	}

	return &stateData, nil
}
