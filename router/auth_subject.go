package router

import (
	"database/sql"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/middleware"
	"go.uber.org/zap"
)

type authSource string

const (
	authSourceSession authSource = "session"
	authSourceProxy   authSource = "proxy"
)

var traQIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

type authSubject struct {
	Source           authSource
	Email            string
	StripeCustomerID string
	TraQID           string
}

func (h *Handlers) resolveSessionSubject(ctx echo.Context) (*authSubject, error) {
	claims, err := middleware.ReadSessionClaims(ctx, h.JWTConfig)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
	}
	if claims == nil {
		return nil, nil
	}

	email := normalizeEmail(claims.Email)
	subject := &authSubject{
		Source: authSourceSession,
		Email:  email,
	}

	user, err := h.Repo.GetUserByMailHash(ctx.Request().Context(), hashEmail(email))
	if err == nil {
		subject.StripeCustomerID = user.StripeCustomerID
		return subject, nil
	}
	if err == sql.ErrNoRows {
		return subject, nil
	}
	h.Logger.Error("failed to fetch user by mail hash during session auth", zap.Error(err))
	return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch user")
}

func (h *Handlers) resolveProxySubject(ctx echo.Context) (*authSubject, bool, error) {
	traqID := getTraQIDFromContext(ctx)
	if traqID == "" {
		return nil, false, nil
	}
	if !traQIDPattern.MatchString(traqID) {
		return nil, true, echo.NewHTTPError(http.StatusBadRequest, "invalid traQ ID")
	}

	customers, err := h.SC.SearchCustomersByTraQID(ctx.Request().Context(), traqID)
	if err != nil {
		h.Logger.Error("failed to search customer by traQ id", zap.String("traq_id", traqID), zap.Error(err))
		return nil, true, echo.NewHTTPError(http.StatusInternalServerError, "failed to resolve traQ authentication")
	}

	switch len(customers) {
	case 0:
		return nil, true, nil
	case 1:
		return &authSubject{
			Source:           authSourceProxy,
			Email:            normalizeEmail(customers[0].Email),
			StripeCustomerID: customers[0].ID,
			TraQID:           traqID,
		}, true, nil
	default:
		return nil, true, echo.NewHTTPError(http.StatusForbidden, "ambiguous traQ authentication")
	}
}

func (h *Handlers) getAuthSubject(ctx echo.Context) (*authSubject, error) {
	sessionSubject, err := h.resolveSessionSubject(ctx)
	if err != nil {
		return nil, err
	}
	proxySubject, proxyHeaderPresent, err := h.resolveProxySubject(ctx)
	if err != nil {
		return nil, err
	}

	if sessionSubject != nil && proxySubject != nil {
		if sessionSubject.StripeCustomerID != "" && proxySubject.StripeCustomerID != "" && sessionSubject.StripeCustomerID != proxySubject.StripeCustomerID {
			return nil, echo.NewHTTPError(http.StatusForbidden, "conflicting authentication context")
		}
		if sessionSubject.Email != "" && proxySubject.Email != "" && sessionSubject.Email != proxySubject.Email {
			return nil, echo.NewHTTPError(http.StatusForbidden, "conflicting authentication context")
		}
		if sessionSubject.TraQID == "" {
			sessionSubject.TraQID = proxySubject.TraQID
		}
	}

	if sessionSubject != nil {
		return sessionSubject, nil
	}
	if proxySubject != nil {
		return proxySubject, nil
	}
	if proxyHeaderPresent {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "verify email is required")
	}
	return nil, echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
}

func (h *Handlers) getExistingCustomerSubject(ctx echo.Context) (*authSubject, error) {
	subject, err := h.getAuthSubject(ctx)
	if err != nil {
		return nil, err
	}
	if subject.StripeCustomerID == "" {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}
	return subject, nil
}

func ensureEmailMatchesSubject(email string, subject *authSubject) (string, error) {
	normalized := normalizeEmail(email)
	if normalized == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}
	if subject.Email != "" && normalized != subject.Email {
		return "", echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	return normalized, nil
}
