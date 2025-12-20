package auth

const (
	DefaultBcryptCost = 12
	MinBcryptCost     = 10
	MaxBcryptCost     = 14

	MinPasswordLength = 8
	MaxPasswordLength = 72 // bcrypt limitation
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)
