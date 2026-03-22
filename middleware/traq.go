package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const traQIDContextKey = "traqID"

func TraQHeaderMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traqID := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-User"))
			c.Set(traQIDContextKey, traqID)
			return next(c)
		}
	}
}

func RequireTraQAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traqID, _ := GetTraQID(c)
			if traqID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "traQ authentication required")
			}
			return next(c)
		}
	}
}

func AdminMiddleware(adminIDs map[string]struct{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			traqID, _ := GetTraQID(c)
			if _, ok := adminIDs[traqID]; !ok {
				return echo.NewHTTPError(http.StatusForbidden, "admin only")
			}
			return next(c)
		}
	}
}

func GetTraQID(c echo.Context) (string, bool) {
	traqID, ok := c.Get(traQIDContextKey).(string)
	return traqID, ok
}
