package router

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"html/template"
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

var verifyEmailConfirmPageTemplate = template.Must(template.New("verify-email-confirm").Parse(`<!doctype html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Verify Email</title>
  </head>
  <body>
    <main>
      <h1>Verify Email</h1>
      <p>{{ .Email }} の確認を完了します。</p>
      <form method="post">
        <input type="hidden" name="token" value="{{ .Token }}">
        <button type="submit">確認して続行</button>
      </form>
    </main>
  </body>
</html>`))

var verifyEmailErrorPageTemplate = template.Must(template.New("verify-email-error").Parse(`<!doctype html>
<html lang="ja">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Verify Email Error</title>
  </head>
  <body>
    <main>
      <h1>{{ .Title }}</h1>
      <p>{{ .Message }}</p>
    </main>
  </body>
</html>`))

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

func setNoStoreHeaders(ctx echo.Context) {
	ctx.Response().Header().Set(echo.HeaderCacheControl, "no-store")
}

func renderHTMLTemplate(ctx echo.Context, status int, tmpl *template.Template, data any) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	ctx.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	return ctx.HTMLBlob(status, buf.Bytes())
}

func renderVerifyEmailErrorPage(ctx echo.Context, status int, title string, message string) error {
	setNoStoreHeaders(ctx)
	return renderHTMLTemplate(ctx, status, verifyEmailErrorPageTemplate, map[string]string{
		"Title":   title,
		"Message": message,
	})
}

func (h *Handlers) loadPendingVerification(ctx echo.Context, token string) (repository.EmailVerification, error) {
	tokenHash := hashVerificationToken(token)
	item, err := h.Repo.GetEmailVerificationByTokenHash(ctx.Request().Context(), tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.EmailVerification{}, echo.NewHTTPError(http.StatusBadRequest, "invalid verification token")
		}
		h.Logger.Error("failed to load email verification", zap.Error(err))
		return repository.EmailVerification{}, echo.NewHTTPError(http.StatusInternalServerError, "failed to complete email verification")
	}
	if item.UsedAt.Valid {
		return repository.EmailVerification{}, echo.NewHTTPError(http.StatusUnauthorized, "verification token has already been used")
	}
	if time.Now().After(item.ExpiresAt) {
		return repository.EmailVerification{}, echo.NewHTTPError(http.StatusUnauthorized, "verification token has expired")
	}
	return item, nil
}

func renderVerifyEmailHTTPError(ctx echo.Context, err error) error {
	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		return renderVerifyEmailErrorPage(ctx, http.StatusInternalServerError, "Verify Email Error", "failed to complete email verification")
	}
	status := httpErr.Code
	message := "failed to complete email verification"
	if msg, ok := httpErr.Message.(string); ok && strings.TrimSpace(msg) != "" {
		message = msg
	}
	title := "Verify Email Error"
	switch status {
	case http.StatusBadRequest:
		title = "Invalid Verification Token"
	case http.StatusUnauthorized:
		title = "Verification Failed"
	}
	return renderVerifyEmailErrorPage(ctx, status, title, message)
}

// GetCsrf issues a CSRF cookie for browser clients.
func (h *Handlers) GetCsrf(ctx echo.Context) error {
	if _, err := middleware.EnsureCSRFCookie(ctx, h.RequireHTTPS); err != nil {
		h.Logger.Error("failed to issue csrf cookie", zap.Error(err))
		return err
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

// GetVerifyEmailConfirm renders an interstitial page and does not consume the token.
func (h *Handlers) GetVerifyEmailConfirm(ctx echo.Context, params api.GetVerifyEmailConfirmParams) error {
	item, err := h.loadPendingVerification(ctx, params.Token)
	if err != nil {
		return renderVerifyEmailHTTPError(ctx, err)
	}
	setNoStoreHeaders(ctx)
	return renderHTMLTemplate(ctx, http.StatusOK, verifyEmailConfirmPageTemplate, map[string]string{
		"Email": item.Email,
		"Token": params.Token,
	})
}

// PostVerifyEmailConfirm consumes a verification token and logs the user in.
func (h *Handlers) PostVerifyEmailConfirm(ctx echo.Context) error {
	if err := ctx.Request().ParseForm(); err != nil {
		return renderVerifyEmailErrorPage(ctx, http.StatusBadRequest, "Invalid Verification Token", "invalid form body")
	}
	token := strings.TrimSpace(ctx.FormValue("token"))
	if token == "" {
		return renderVerifyEmailErrorPage(ctx, http.StatusBadRequest, "Invalid Verification Token", "verification token is required")
	}

	item, err := h.loadPendingVerification(ctx, token)
	if err != nil {
		return renderVerifyEmailHTTPError(ctx, err)
	}

	rowsAffected, err := h.Repo.MarkEmailVerificationUsed(ctx.Request().Context(), hashVerificationToken(token), time.Now())
	if err != nil {
		h.Logger.Error("failed to consume email verification", zap.Error(err))
		return renderVerifyEmailErrorPage(ctx, http.StatusInternalServerError, "Verify Email Error", "failed to complete email verification")
	}
	if rowsAffected == 0 {
		return renderVerifyEmailErrorPage(ctx, http.StatusUnauthorized, "Verification Failed", "verification token has already been used")
	}

	sessionToken, err := h.JWTConfig.GenerateToken(item.Email)
	if err != nil {
		h.Logger.Error("failed to generate session token", zap.Error(err))
		return renderVerifyEmailErrorPage(ctx, http.StatusInternalServerError, "Verify Email Error", "failed to complete email verification")
	}

	if err := middleware.SetSessionCookie(ctx, h.JWTConfig, sessionToken, h.RequireHTTPS); err != nil {
		return renderVerifyEmailHTTPError(ctx, err)
	}
	if _, err := middleware.EnsureCSRFCookie(ctx, h.RequireHTTPS); err != nil {
		h.Logger.Error("failed to issue csrf cookie during email verification", zap.Error(err))
		return renderVerifyEmailHTTPError(ctx, err)
	}

	setNoStoreHeaders(ctx)
	return ctx.Redirect(http.StatusSeeOther, item.RedirectPath)
}
