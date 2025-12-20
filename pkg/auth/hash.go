package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func CompareTokenHash(token, storedHash string) bool {
	return HashToken(token) == storedHash
}
