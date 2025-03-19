package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/Checkin-Server/repository"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger *zap.Logger
	Repo   repository.Repository
}

// ServerInterface ensures Handlers implements the generated server interface
var _ api.ServerInterface = (*Handlers)(nil)

func (h *Handlers) Setup() *echo.Echo {
	e := echo.New()

	// Register handler as implementing the server interface
	api.RegisterHandlers(e, h)

	return e
}

// DeleteAdmin implements the server interface method to delete an admin
func (h *Handlers) DeleteAdmin(ctx echo.Context, params api.DeleteAdminParams) error {
	h.Logger.Info("DeleteAdmin endpoint called")
	// TODO: Implement using h.Repo.DeleteAdmin
	return ctx.NoContent(http.StatusNotImplemented)
}

// GetAdmins implements the server interface method to get admin list
func (h *Handlers) GetAdmins(ctx echo.Context) error {
	h.Logger.Info("GetAdmins endpoint called")
	// TODO: Implement using h.Repo.GetAdmins
	return ctx.NoContent(http.StatusNotImplemented)
}

// PostAdmin implements the server interface method to create an admin
func (h *Handlers) PostAdmin(ctx echo.Context) error {
	h.Logger.Info("PostAdmin endpoint called")
	// TODO: Implement using h.Repo.CreateAdmin
	return ctx.NoContent(http.StatusNotImplemented)
}

// GetCustomer implements the server interface method to get customer information
func (h *Handlers) GetCustomer(ctx echo.Context, params api.GetCustomerParams) error {
	h.Logger.Info("GetCustomer endpoint called")
	// TODO: Implement using h.Repo.GetCustomer
	return ctx.NoContent(http.StatusNotImplemented)
}

// PatchCustomer implements the server interface method to update customer information
func (h *Handlers) PatchCustomer(ctx echo.Context) error {
	h.Logger.Info("PatchCustomer endpoint called")
	// TODO: Implement using h.Repo.UpdateCustomer
	return ctx.NoContent(http.StatusNotImplemented)
}

// PostCustomer implements the server interface method to create a customer
func (h *Handlers) PostCustomer(ctx echo.Context) error {
	h.Logger.Info("PostCustomer endpoint called")
	// TODO: Implement using h.Repo.CreateCustomer
	return ctx.NoContent(http.StatusNotImplemented)
}

// PostInvoice implements the server interface method to create an invoice
func (h *Handlers) PostInvoice(ctx echo.Context) error {
	h.Logger.Info("PostInvoice endpoint called")
	// TODO: Implement using h.Repo.CreateInvoice
	return ctx.NoContent(http.StatusNotImplemented)
}

// GetCheckoutSessions implements the server interface method to get checkout sessions
func (h *Handlers) GetCheckoutSessions(ctx echo.Context, params api.GetCheckoutSessionsParams) error {
	h.Logger.Info("GetCheckoutSessions endpoint called")
	// TODO: Implement using h.Repo.GetCheckoutSessions
	return ctx.NoContent(http.StatusNotImplemented)
}

// GetInvoices implements the server interface method to get invoices
func (h *Handlers) GetInvoices(ctx echo.Context, params api.GetInvoicesParams) error {
	h.Logger.Info("GetInvoices endpoint called")
	// TODO: Implement using h.Repo.GetInvoices
	return ctx.NoContent(http.StatusNotImplemented)
}

// PostWebhookInvoicePaid implements the server interface method to handle the invoice.paid webhook
func (h *Handlers) PostWebhookInvoicePaid(ctx echo.Context, params api.PostWebhookInvoicePaidParams) error {
	h.Logger.Info("PostWebhookInvoicePaid endpoint called")

	// Parse the invoice data from the request body
	var invoice api.Invoice
	if err := ctx.Bind(&invoice); err != nil {
		h.Logger.Error("Failed to parse webhook payload", zap.Error(err))
		return ctx.NoContent(http.StatusBadRequest)
	}

	// Process the invoice.paid webhook
	if err := h.Repo.HandleInvoicePaidWebhook(invoice); err != nil {
		h.Logger.Error("Failed to handle invoice.paid webhook", zap.Error(err))
		return ctx.NoContent(http.StatusInternalServerError)
	}

	return ctx.NoContent(http.StatusOK)
}
