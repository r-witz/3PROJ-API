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
	domain.ErrUserNotFound:          {http.StatusNotFound, "USER_NOT_FOUND", "User not found"},
	domain.ErrMovieNotFound:         {http.StatusNotFound, "MOVIE_NOT_FOUND", "Movie not found"},
	domain.ErrTMDBError:             {http.StatusBadGateway, "EXTERNAL_SERVICE_ERROR", "Movie service is temporarily unavailable"},
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
