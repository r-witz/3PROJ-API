package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	return HashPasswordWithCost(password, DefaultBcryptCost)
}

func HashPasswordWithCost(password string, cost int) (string, error) {
	if len(password) < MinPasswordLength {
		return "", &PasswordError{
			Operation: "hash",
			Err:       ErrPasswordTooShort,
		}
	}
	if len(password) > MaxPasswordLength {
		return "", &PasswordError{
			Operation: "hash",
			Err:       ErrPasswordTooLong,
		}
	}

	if err := validatePasswordComplexity(password); err != nil {
		return "", err
	}

	if cost < MinBcryptCost {
		cost = MinBcryptCost
	}
	if cost > MaxBcryptCost {
		cost = MaxBcryptCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", &PasswordError{
			Operation: "hash",
			Err:       ErrHashingFailed,
		}
	}

	return string(hash), nil
}

func ComparePassword(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}

	return false, &PasswordError{
		Operation: "compare",
		Err:       err,
	}
}

func validatePasswordComplexity(password string) error {
	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return &PasswordError{Operation: "validate", Err: ErrPasswordNoUppercase}
	}
	if !hasLower {
		return &PasswordError{Operation: "validate", Err: ErrPasswordNoLowercase}
	}
	if !hasDigit {
		return &PasswordError{Operation: "validate", Err: ErrPasswordNoDigit}
	}
	if !hasSpecial {
		return &PasswordError{Operation: "validate", Err: ErrPasswordNoSpecialChar}
	}

	return nil
}
