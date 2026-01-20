package response

import (
	"errors"
	"net/http"

	"duskforge-api/internal/core/domain"

	"github.com/gin-gonic/gin"
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
	domain.ErrUsernameAlreadyExists: {http.StatusConflict, "USERNAME_EXISTS", "Username is already taken"},
	domain.ErrInvalidToken:          {http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token"},
	domain.ErrSessionExpired:        {http.StatusUnauthorized, "SESSION_EXPIRED", "Session has expired"},
	domain.ErrUserBanned:            {http.StatusForbidden, "USER_BANNED", "Account has been banned"},
	domain.ErrUserNotFound:       {http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
	domain.ErrNoPasswordSet:      {http.StatusConflict, "NO_PASSWORD_SET", "No password set for this account"},
	domain.ErrIncorrectPassword:  {http.StatusUnauthorized, "INCORRECT_PASSWORD", "Current password is incorrect"},

	// Password validation errors
	domain.ErrPasswordTooShort:      {http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters"},
	domain.ErrPasswordTooLong:       {http.StatusBadRequest, "PASSWORD_TOO_LONG", "Password must be at most 72 characters"},
	domain.ErrPasswordNoUppercase:   {http.StatusBadRequest, "PASSWORD_NO_UPPERCASE", "Password must contain at least one uppercase letter"},
	domain.ErrPasswordNoLowercase:   {http.StatusBadRequest, "PASSWORD_NO_LOWERCASE", "Password must contain at least one lowercase letter"},
	domain.ErrPasswordNoDigit:       {http.StatusBadRequest, "PASSWORD_NO_DIGIT", "Password must contain at least one digit"},
	domain.ErrPasswordNoSpecialChar: {http.StatusBadRequest, "PASSWORD_NO_SPECIAL", "Password must contain at least one special character"},

	domain.ErrMovieNotFound:      {http.StatusNotFound, "MOVIE_NOT_FOUND", "Movie not found"},
	domain.ErrTMDBError:             {http.StatusBadGateway, "EXTERNAL_SERVICE_ERROR", "Movie service is temporarily unavailable"},

	// OAuth errors
	domain.ErrOAuthAccountNotFound:      {http.StatusNotFound, "OAUTH_NOT_FOUND", "OAuth account not found"},
	domain.ErrOAuthAccountAlreadyLinked: {http.StatusConflict, "OAUTH_ALREADY_LINKED", "OAuth account is already linked to another user"},
	domain.ErrOAuthProviderNotSupported: {http.StatusBadRequest, "OAUTH_PROVIDER_INVALID", "OAuth provider not supported"},
	domain.ErrOAuthStateMismatch:        {http.StatusBadRequest, "OAUTH_STATE_INVALID", "Invalid OAuth state parameter"},
	domain.ErrCannotUnlinkOnlyAuth:      {http.StatusBadRequest, "CANNOT_UNLINK", "Cannot remove the only authentication method"},
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
