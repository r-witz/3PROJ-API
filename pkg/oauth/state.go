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

type StateManager struct {
	secret []byte
	expiry time.Duration
}

type StateData struct {
	Timestamp   int64  `json:"t"`
	Nonce       string `json:"n"`
	RedirectURI string `json:"r,omitempty"`
}

func NewStateManager(secret string, expiry time.Duration) *StateManager {
	return &StateManager{
		secret: []byte(secret),
		expiry: expiry,
	}
}

func (m *StateManager) Generate(redirectURI string) (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}

	data := StateData{
		Timestamp:   time.Now().Unix(),
		Nonce:       base64.URLEncoding.EncodeToString(nonceBytes),
		RedirectURI: redirectURI,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	signature := mac.Sum(nil)

	result := make([]byte, 4+len(payload)+len(signature))
	binary.BigEndian.PutUint32(result[:4], uint32(len(payload)))
	copy(result[4:4+len(payload)], payload)
	copy(result[4+len(payload):], signature)

	return base64.URLEncoding.EncodeToString(result), nil
}

func (m *StateManager) Validate(state string) (*StateData, error) {
	data, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return nil, ErrInvalidState
	}

	if len(data) < 37 {
		return nil, ErrInvalidState
	}

	payloadLen := binary.BigEndian.Uint32(data[:4])
	if int(payloadLen) > len(data)-36 {
		return nil, ErrInvalidState
	}

	payload := data[4 : 4+payloadLen]
	providedSignature := data[4+payloadLen:]

	if len(providedSignature) != 32 {
		return nil, ErrInvalidState
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return nil, ErrInvalidState
	}

	var stateData StateData
	if err := json.Unmarshal(payload, &stateData); err != nil {
		return nil, ErrInvalidState
	}

	createdAt := time.Unix(stateData.Timestamp, 0)
	if time.Since(createdAt) > m.expiry {
		return nil, ErrExpiredState
	}

	return &stateData, nil
}
