package middleware

import (
	"strings"

	"duskforge-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

const ContextKeyLocale = "locale"

func Locale(userService ports.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := "en-US"

		acceptLang := c.GetHeader("Accept-Language")
		if acceptLang != "" {
			locale = mapLocaleToTMDB(acceptLang)
		} else if userID, ok := GetUserID(c); ok {
			user, err := userService.GetByID(c.Request.Context(), userID)
			if err == nil && user != nil {
				locale = mapUserLocaleToTMDB(string(user.Locale))
			}
		}

		c.Set(ContextKeyLocale, locale)
		c.Next()
	}
}

func GetLocale(c *gin.Context) string {
	val, exists := c.Get(ContextKeyLocale)
	if !exists {
		return "en-US"
	}
	locale, ok := val.(string)
	if !ok {
		return "en-US"
	}
	return locale
}

func mapLocaleToTMDB(acceptLang string) string {
	lang := strings.Split(acceptLang, ",")[0]
	lang = strings.Split(lang, ";")[0]
	lang = strings.TrimSpace(lang)
	lang = strings.ToLower(lang)

	switch {
	case strings.HasPrefix(lang, "fr"):
		return "fr-FR"
	case strings.HasPrefix(lang, "es"):
		return "es-ES"
	default:
		return "en-US"
	}
}

func mapUserLocaleToTMDB(locale string) string {
	switch locale {
	case "fr":
		return "fr-FR"
	case "es":
		return "es-ES"
	default:
		return "en-US"
	}
}
