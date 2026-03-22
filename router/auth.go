package router

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const defaultVerifyEmailRedirect = "/membership"

type postVerifyEmailRequest struct {
	Email string `json:"email"`
}

type postVerifyEmailResponse struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Redirect string `json:"redirect"`
}

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

// PostVerifyEmail handles email verification requests
func (h *Handlers) PostVerifyEmail(ctx echo.Context) error {
	var body postVerifyEmailRequest
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	email := normalizeEmail(body.Email)
	if email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}
	if !strings.HasSuffix(email, "@isct.ac.jp") {
		return echo.NewHTTPError(http.StatusBadRequest, "email must be an isct.ac.jp address")
	}
	redirect, err := normalizeVerifyRedirect(stringPtr(ctx.QueryParam("redirect")))
	if err != nil {
		return err
	}

	// Generate JWT token
	token, err := h.JWTConfig.GenerateToken(email)
	if err != nil {
		h.Logger.Error("failed to generate JWT token", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	h.Logger.Info("verify email token issued",
		zap.String("to", email),
		zap.String("redirect", redirect),
	)

	return ctx.JSON(http.StatusOK, postVerifyEmailResponse{
		Email:    email,
		Token:    token,
		Redirect: redirect,
	})
}
