package auth

import (
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

	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}

	return false, &PasswordError{
		Operation: "compare",
		Err:       err,
	}
}
