package router

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/repository"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger *zap.Logger
	Repo   repository.Repository
}

// DeleteAdmin implements api.ServerInterface.
func (h *Handlers) DeleteAdmin(ctx echo.Context, params api.DeleteAdminParams) error {
	return NotImplementedError()
}

// GetAdmins implements api.ServerInterface.
func (h *Handlers) GetAdmins(ctx echo.Context) error {
	return NotImplementedError()
}

// PostAdmin implements api.ServerInterface.
func (h *Handlers) PostAdmin(ctx echo.Context) error {
	return NotImplementedError()
}

// GetCustomer implements api.ServerInterface.
func (h *Handlers) GetCustomer(ctx echo.Context, params api.GetCustomerParams) error {
	return NotImplementedError()
}

// PatchCustomer implements api.ServerInterface.
func (h *Handlers) PatchCustomer(ctx echo.Context) error {
	return NotImplementedError()
}

// PostCustomer implements api.ServerInterface.
func (h *Handlers) PostCustomer(ctx echo.Context) error {
	return NotImplementedError()
}

// PostInvoice implements api.ServerInterface.
func (h *Handlers) PostInvoice(ctx echo.Context) error {
	return NotImplementedError()
}

// GetCheckoutSessions implements api.ServerInterface.
func (h *Handlers) GetCheckoutSessions(ctx echo.Context, params api.GetCheckoutSessionsParams) error {
	return NotImplementedError()
}

// GetInvoices implements api.ServerInterface.
func (h *Handlers) GetInvoices(ctx echo.Context, params api.GetInvoicesParams) error {
	return NotImplementedError()
}

// PostWebhookInvoicePaid implements api.ServerInterface.
func (h *Handlers) PostWebhookInvoicePaid(ctx echo.Context, params api.PostWebhookInvoicePaidParams) error {
	return NotImplementedError()
}

func (h *Handlers) Setup() *echo.Echo {
	e := echo.New()
	api.RegisterHandlers(e, h)

	return e
}
