package router

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// PostVerifyEmail handles email verification requests
func (h *Handlers) PostVerifyEmail(ctx echo.Context) error {
	var body struct {
		Email string `json:"email"`
	}

	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if body.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	// Validate that it's an isct email address
	if !strings.HasSuffix(body.Email, "@isct.ac.jp") {
		return echo.NewHTTPError(http.StatusBadRequest, "email must be an isct.ac.jp address")
	}

	// Generate JWT token
	token, err := h.JWTConfig.GenerateToken(body.Email)
	if err != nil {
		h.Logger.Error("failed to generate JWT token", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	// Mock email sending - just log the verification URL
	// Mock email sending
	h.Logger.Info("Mock email sent",
		zap.String("to", body.Email),
	)

	// Return the token directly to the client
	return ctx.JSON(http.StatusOK, map[string]string{
		"token": token,
		"email": body.Email,
	})
}
