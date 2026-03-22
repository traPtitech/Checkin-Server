package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const (
	SessionCookieName = "checkin_session"
	CSRFCookieName    = "checkin_csrf"
	CSRFHeaderName    = "X-CSRF-Token"
)

// JWTClaims represents the claims stored in the JWT token.
type JWTClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	SecretKey       string
	ExpirationHours int
}

// NewJWTConfig creates a new JWT configuration from environment variables.
func NewJWTConfig() *JWTConfig {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		panic("JWT_SECRET environment variable is not set")
	}

	expirationHours := 2
	if envHours := os.Getenv("JWT_EXPIRATION_HOURS"); envHours != "" {
		if hours, err := strconv.Atoi(envHours); err == nil {
			expirationHours = hours
		}
	}

	return &JWTConfig{
		SecretKey:       secretKey,
		ExpirationHours: expirationHours,
	}
}

// GenerateToken generates a new JWT token for the given email.
func (c *JWTConfig) GenerateToken(email string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(c.ExpirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.SecretKey))
}

// ValidateToken validates a JWT token and returns the claims.
func (c *JWTConfig) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func sessionCookieMaxAgeSeconds(config *JWTConfig) int {
	return config.ExpirationHours * 60 * 60
}

func useSecureCookies(c echo.Context) bool {
	if c.Scheme() == "https" {
		return true
	}
	host := c.Request().Host
	return !strings.Contains(host, "localhost") && !strings.HasPrefix(host, "127.0.0.1") && !strings.HasPrefix(host, "[::1]")
}

// SetSessionCookie stores the session JWT in an HttpOnly cookie.
func SetSessionCookie(c echo.Context, config *JWTConfig, token string) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   useSecureCookies(c),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionCookieMaxAgeSeconds(config),
	})
}

// ReadSessionClaims returns the JWT claims from the session cookie if present.
func ReadSessionClaims(c echo.Context, config *JWTConfig) (*JWTClaims, error) {
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, err
	}
	if strings.TrimSpace(cookie.Value) == "" {
		return nil, nil
	}
	return config.ValidateToken(cookie.Value)
}
