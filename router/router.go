package router

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/repository"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger *zap.Logger
	Repo   repository.Repository
}

func (h *Handlers) Setup() *echo.Echo {
	e := echo.New()
	return e
}
