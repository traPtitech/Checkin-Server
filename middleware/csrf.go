package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// NewRandomToken returns a URL-safe random token.
func NewRandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// EnsureCSRFCookie returns the current CSRF token or issues a new one.
func EnsureCSRFCookie(c echo.Context) (string, error) {
	if cookie, err := c.Cookie(CSRFCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value, nil
	}

	token, err := NewRandomToken(32)
	if err != nil {
		return "", err
	}

	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   useSecureCookies(c),
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}

// ValidateCSRF checks the double-submit CSRF token.
func ValidateCSRF(c echo.Context, headerValue string) error {
	cookie, err := c.Cookie(CSRFCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return echo.NewHTTPError(http.StatusForbidden, "missing csrf cookie")
		}
		return echo.NewHTTPError(http.StatusForbidden, "invalid csrf cookie")
	}

	headerToken := strings.TrimSpace(headerValue)
	cookieToken := strings.TrimSpace(cookie.Value)
	if headerToken == "" || cookieToken == "" || headerToken != cookieToken {
		return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
	}
	return nil
}
