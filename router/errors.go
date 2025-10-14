package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// NotImplementedError returns a new HTTP error for not implemented endpoints
func NotImplementedError() *echo.HTTPError {
	return echo.NewHTTPError(http.StatusNotImplemented, "Not implemented")
}
