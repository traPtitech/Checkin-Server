package router

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/middleware"
	"github.com/traPtitech/Checkin-Server/repository"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

const (
	defaultVerifyEmailRedirect  = "/membership"
	defaultPublicAPIBaseURL     = "http://localhost:5173/api"
	defaultVerificationTokenTTL = 15 * time.Minute
)

func normalizeVerifyRedirect(raw *string) (string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return defaultVerifyEmailRedirect, nil
	}
	redirect := strings.TrimSpace(*raw)
	if !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		return "", echo.NewHTTPError(http.StatusBadRequest, "redirect must be a relative path")
	}
	return redirect, nil
}

func hashVerificationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h *Handlers) verificationTokenTTL() time.Duration {
	if h.VerificationTokenTTL > 0 {
		return h.VerificationTokenTTL
	}
	return defaultVerificationTokenTTL
}

func (h *Handlers) publicAPIBaseURL() string {
	if trimmed := strings.TrimRight(strings.TrimSpace(h.PublicAPIBaseURL), "/"); trimmed != "" {
		return trimmed
	}
	return defaultPublicAPIBaseURL
}

func (h *Handlers) mailer() Mailer {
	if h.Mailer != nil {
		return h.Mailer
	}
	return MockMailer{Logger: h.Logger}
}

func (h *Handlers) buildVerificationURL(token string) string {
	return h.publicAPIBaseURL() + "/verify-email/confirm?token=" + url.QueryEscape(token)
}

// GetCsrf issues a CSRF cookie for browser clients.
func (h *Handlers) GetCsrf(ctx echo.Context) error {
	if _, err := middleware.EnsureCSRFCookie(ctx); err != nil {
		h.Logger.Error("failed to issue csrf cookie", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue csrf token")
	}
	return ctx.NoContent(http.StatusNoContent)
}

// PostVerifyEmail starts the email verification flow.
func (h *Handlers) PostVerifyEmail(ctx echo.Context, params api.PostVerifyEmailParams) error {
	var body api.PostVerifyEmailJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	email := normalizeEmail(string(body.Email))
	if email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}
	if !strings.HasSuffix(email, "@isct.ac.jp") {
		return echo.NewHTTPError(http.StatusBadRequest, "email must be an isct.ac.jp address")
	}
	redirect, err := normalizeVerifyRedirect(params.Redirect)
	if err != nil {
		return err
	}

	token, err := middleware.NewRandomToken(32)
	if err != nil {
		h.Logger.Error("failed to generate verification token", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate verification token")
	}
	tokenHash := hashVerificationToken(token)

	if err := h.Repo.DeleteUnusedEmailVerificationsByEmail(ctx.Request().Context(), email); err != nil {
		h.Logger.Error("failed to invalidate old email verifications", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start email verification")
	}

	err = h.Repo.CreateEmailVerification(ctx.Request().Context(), repository.CreateEmailVerificationParams{
		TokenHash:    tokenHash,
		Email:        email,
		RedirectPath: redirect,
		ExpiresAt:    time.Now().Add(h.verificationTokenTTL()),
	})
	if err != nil {
		h.Logger.Error("failed to persist email verification", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start email verification")
	}

	if err := h.mailer().SendVerificationEmail(ctx.Request().Context(), email, h.buildVerificationURL(token)); err != nil {
		h.Logger.Error("failed to send verification email", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
	}

	return ctx.JSON(http.StatusAccepted, api.VerifyEmailStartResponse{
		Email:    email,
		Redirect: redirect,
	})
}

// GetVerifyEmailConfirm consumes a verification token and logs the user in.
func (h *Handlers) GetVerifyEmailConfirm(ctx echo.Context, params api.GetVerifyEmailConfirmParams) error {
	tokenHash := hashVerificationToken(params.Token)
	item, err := h.Repo.GetEmailVerificationByTokenHash(ctx.Request().Context(), tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid verification token")
		}
		h.Logger.Error("failed to load email verification", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to complete email verification")
	}
	if item.UsedAt.Valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "verification token has already been used")
	}
	if time.Now().After(item.ExpiresAt) {
		return echo.NewHTTPError(http.StatusUnauthorized, "verification token has expired")
	}

	rowsAffected, err := h.Repo.MarkEmailVerificationUsed(ctx.Request().Context(), tokenHash, time.Now())
	if err != nil {
		h.Logger.Error("failed to consume email verification", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to complete email verification")
	}
	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "verification token has already been used")
	}

	sessionToken, err := h.JWTConfig.GenerateToken(item.Email)
	if err != nil {
		h.Logger.Error("failed to generate session token", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to complete email verification")
	}

	middleware.SetSessionCookie(ctx, h.JWTConfig, sessionToken)
	if _, err := middleware.EnsureCSRFCookie(ctx); err != nil {
		h.Logger.Error("failed to issue csrf cookie during email verification", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to complete email verification")
	}

	return ctx.Redirect(http.StatusSeeOther, item.RedirectPath)
}
