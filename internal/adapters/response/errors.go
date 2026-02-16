package response

import (
	"errors"
	"net/http"

	"duskforge-api/internal/core/domain"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorMapping struct {
	Status  int
	Code    string
	Message string
}

var errorMappings = map[error]ErrorMapping{
	domain.ErrNotFound:              {http.StatusNotFound, "NOT_FOUND", "Resource not found"},
	domain.ErrAlreadyExists:         {http.StatusConflict, "ALREADY_EXISTS", "Resource already exists"},
	domain.ErrInvalidInput:          {http.StatusBadRequest, "INVALID_INPUT", "Invalid input provided"},
	domain.ErrUnauthorized:          {http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized"},
	domain.ErrForbidden:             {http.StatusForbidden, "FORBIDDEN", "Access denied"},
	domain.ErrInternal:              {http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"},
	domain.ErrInvalidCredentials:    {http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password"},
	domain.ErrEmailAlreadyExists:    {http.StatusConflict, "EMAIL_EXISTS", "Email is already registered"},
	domain.ErrEmailRequired:         {http.StatusBadRequest, "EMAIL_REQUIRED", "Email is required"},
	domain.ErrUsernameAlreadyExists: {http.StatusConflict, "USERNAME_EXISTS", "Username is already taken"},
	domain.ErrUsernameRequired:      {http.StatusBadRequest, "USERNAME_REQUIRED", "Username is required"},
	domain.ErrUsernameTooShort:      {http.StatusBadRequest, "USERNAME_TOO_SHORT", "Username must be at least 3 characters"},
	domain.ErrUsernameTooLong:       {http.StatusBadRequest, "USERNAME_TOO_LONG", "Username must be at most 50 characters"},
	domain.ErrInvalidEmailFormat:    {http.StatusBadRequest, "INVALID_EMAIL", "Email must be a valid email address"},
	domain.ErrInvalidToken:          {http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token"},
	domain.ErrSessionExpired:        {http.StatusUnauthorized, "SESSION_EXPIRED", "Session has expired"},
	domain.ErrUserBanned:            {http.StatusForbidden, "USER_BANNED", "Account has been banned"},
	domain.ErrUserNotFound:       {http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
	domain.ErrNoPasswordSet:      {http.StatusConflict, "NO_PASSWORD_SET", "No password set for this account"},
	domain.ErrIncorrectPassword:  {http.StatusUnauthorized, "INCORRECT_PASSWORD", "Current password is incorrect"},

	// Password validation errors
	domain.ErrPasswordRequired:      {http.StatusBadRequest, "PASSWORD_REQUIRED", "Password is required"},
	domain.ErrPasswordTooShort:      {http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters"},
	domain.ErrPasswordTooLong:       {http.StatusBadRequest, "PASSWORD_TOO_LONG", "Password must be at most 72 characters"},
	domain.ErrPasswordNoUppercase:   {http.StatusBadRequest, "PASSWORD_NO_UPPERCASE", "Password must contain at least one uppercase letter"},
	domain.ErrPasswordNoLowercase:   {http.StatusBadRequest, "PASSWORD_NO_LOWERCASE", "Password must contain at least one lowercase letter"},
	domain.ErrPasswordNoDigit:       {http.StatusBadRequest, "PASSWORD_NO_DIGIT", "Password must contain at least one digit"},
	domain.ErrPasswordNoSpecialChar: {http.StatusBadRequest, "PASSWORD_NO_SPECIAL", "Password must contain at least one special character"},

	domain.ErrMovieNotFound:      {http.StatusNotFound, "MOVIE_NOT_FOUND", "Movie not found"},
	domain.ErrActorNotFound:      {http.StatusNotFound, "ACTOR_NOT_FOUND", "Actor not found"},
	domain.ErrTMDBError:             {http.StatusBadGateway, "EXTERNAL_SERVICE_ERROR", "Movie service is temporarily unavailable"},

	// OAuth errors
	domain.ErrOAuthAccountNotFound:      {http.StatusNotFound, "OAUTH_NOT_FOUND", "OAuth account not found"},
	domain.ErrOAuthAccountAlreadyLinked: {http.StatusConflict, "OAUTH_ALREADY_LINKED", "OAuth account is already linked to another user"},
	domain.ErrOAuthProviderNotSupported: {http.StatusBadRequest, "OAUTH_PROVIDER_INVALID", "OAuth provider not supported"},
	domain.ErrOAuthStateMismatch:        {http.StatusBadRequest, "OAUTH_STATE_INVALID", "Invalid OAuth state parameter"},
	domain.ErrCannotUnlinkOnlyAuth:      {http.StatusBadRequest, "CANNOT_UNLINK", "Cannot remove the only authentication method"},

	// Collection errors
	domain.ErrCollectionNotFound:           {http.StatusNotFound, "COLLECTION_NOT_FOUND", "Collection not found"},
	domain.ErrCollectionAlreadyExists:      {http.StatusConflict, "COLLECTION_EXISTS", "A collection with this name already exists"},
	domain.ErrCannotModifySystemCollection: {http.StatusForbidden, "SYSTEM_COLLECTION", "Cannot modify a system collection"},
	domain.ErrCannotDeleteSystemCollection: {http.StatusForbidden, "SYSTEM_COLLECTION", "Cannot delete a system collection"},
	domain.ErrCollectionItemAlreadyExists:  {http.StatusConflict, "ITEM_EXISTS", "Item already exists in this collection"},
	domain.ErrCollectionItemNotFound:       {http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found in this collection"},
}

func HandleError(c *gin.Context, err error) {
	for domainErr, mapping := range errorMappings {
		if errors.Is(err, domainErr) {
			Error(c, mapping.Status, mapping.Code, mapping.Message, nil)
			return
		}
	}

	InternalError(c)
}

func HandleValidationError(c *gin.Context, err error) bool {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return false
	}

	for _, fieldErr := range validationErrors {
		switch fieldErr.Field() {
		case "Email":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrEmailRequired)
				return true
			case "email":
				HandleError(c, domain.ErrInvalidEmailFormat)
				return true
			}
		case "Username":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrUsernameRequired)
				return true
			case "min":
				HandleError(c, domain.ErrUsernameTooShort)
				return true
			case "max":
				HandleError(c, domain.ErrUsernameTooLong)
				return true
			}
		case "Password", "NewPassword", "CurrentPassword":
			switch fieldErr.Tag() {
			case "required":
				HandleError(c, domain.ErrPasswordRequired)
				return true
			case "min":
				HandleError(c, domain.ErrPasswordTooShort)
				return true
			case "max":
				HandleError(c, domain.ErrPasswordTooLong)
				return true
			}
		}
	}

	return false
}
