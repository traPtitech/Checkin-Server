package router

import (
	"database/sql"
	"net/http"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/labstack/echo/v4"
	oapiMiddleware "github.com/oapi-codegen/echo-middleware"
	"github.com/stripe/stripe-go/v81"
	"github.com/traPtitech/Checkin-Server/repository"
	stripeservice "github.com/traPtitech/Checkin-Server/service/stripe"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger *zap.Logger
	Repo   *repository.Queries
	SC     stripeservice.Service
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
	ctxReq := ctx.Request().Context()

	if params.CustomerId != nil {
		cust, err := h.SC.GetCustomer(ctxReq, *params.CustomerId)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "customer not found")
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
	}

	if params.Email != nil {
		hash := sha256.Sum256([]byte(*params.Email))
		mailHash := hex.EncodeToString(hash[:])
		user, err := h.Repo.GetUserByMailHash(ctxReq, mailHash)
		if err == nil {
			cust, err := h.SC.GetCustomer(ctxReq, user.StripeCustomerID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
			return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
		}
		
		customers, err := h.SC.SearchCustomersByEmail(ctxReq, *params.Email)
		if err != nil || len(customers) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "customer not found")
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(customers[0]))
	}

	if params.TraqId != nil {
		customers, err := h.SC.SearchCustomersByTraQID(ctxReq, *params.TraqId)
		if err != nil || len(customers) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "customer not found")
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(customers[0]))
	}

	return echo.NewHTTPError(http.StatusBadRequest, "one of customerId, email, or traqId is required")
}

// PatchCustomer implements api.ServerInterface.
func (h *Handlers) PatchCustomer(ctx echo.Context) error {
	email := ctx.Request().Header.Get("X-User-Email")
	if email == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing X-User-Email header")
	}

	hash := sha256.Sum256([]byte(email))
	mailHash := hex.EncodeToString(hash[:])
	user, err := h.Repo.GetUserByMailHash(ctx.Request().Context(), mailHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	var body api.PatchCustomerJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var namePtr *string
	if body.Name != "" {
		namePtr = &body.Name
	}
	
	cust, err := h.SC.UpdateCustomer(ctx.Request().Context(), user.StripeCustomerID, nil, namePtr, body.TraqId)
	if err != nil {
		h.Logger.Error("failed to update stripe customer", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
}

// PostCustomer implements api.ServerInterface.
func (h *Handlers) PostCustomer(ctx echo.Context) error {
	var body api.PostCustomerJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if body.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	hash := sha256.Sum256([]byte(body.Email))
	mailHash := hex.EncodeToString(hash[:])

	user, err := h.Repo.GetUserByMailHash(ctx.Request().Context(), mailHash)
	if err == nil {
		cust, err := h.SC.GetCustomer(ctx.Request().Context(), user.StripeCustomerID)
		if err != nil {
			h.Logger.Error("failed to get stripe customer", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		res := mapStripeCustomerToResponse(cust)
		return ctx.JSON(http.StatusOK, res)
	} else if err != sql.ErrNoRows {
		h.Logger.Error("failed to get user by mail hash", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	customers, err := h.SC.SearchCustomersByEmail(ctx.Request().Context(), body.Email)
	if err != nil {
		h.Logger.Error("failed to search customers by email", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var targetCustomer *stripe.Customer
	if len(customers) > 0 {
		targetCustomer = customers[0]
	} else {
		emailPtr := &body.Email
		namePtr := &body.Name
		var traqIDPtr *string
		if body.TraqId != nil {
			traqIDPtr = body.TraqId
		}
		
		targetCustomer, err = h.SC.CreateCustomer(ctx.Request().Context(), emailPtr, namePtr, traqIDPtr)
		if err != nil {
			h.Logger.Error("failed to create stripe customer", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	err = h.Repo.CreateUser(ctx.Request().Context(), repository.CreateUserParams{
		ID:               targetCustomer.ID,
		MailHash:         mailHash,
		StripeCustomerID: targetCustomer.ID,
	})
	if err != nil {
		h.Logger.Error("failed to create user", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	res := mapStripeCustomerToResponse(targetCustomer)
	return ctx.JSON(http.StatusCreated, res)
}

func mapStripeCustomerToResponse(cust *stripe.Customer) api.Customer {
	var email, name, traqID *string
	if cust.Email != "" {
		email = &cust.Email
	}
	if cust.Name != "" {
		name = &cust.Name
	}
	if cust.Metadata != nil {
		if t, ok := cust.Metadata["traQID"]; ok {
			traqID = &t
		}
	}
	return api.Customer{
		Id:     &cust.ID,
		Email:  email,
		Name:   name,
		TraqId: traqID,
	}
}

// PostInvoice implements api.ServerInterface.
func (h *Handlers) PostInvoice(ctx echo.Context) error {
	email := ctx.Request().Header.Get("X-User-Email")
	if email == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing X-User-Email header")
	}

	hash := sha256.Sum256([]byte(email))
	mailHash := hex.EncodeToString(hash[:])
	user, err := h.Repo.GetUserByMailHash(ctx.Request().Context(), mailHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	var body api.PostInvoiceJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if body.ProductId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product_id is required")
	}

	invID, err := h.SC.CreateInvoice(ctx.Request().Context(), user.StripeCustomerID, body.ProductId)
	if err != nil {
		h.Logger.Error("failed to create invoice", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	session, err := h.SC.CreateCheckoutSession(ctx.Request().Context(), invID)
	if err != nil {
		h.Logger.Error("failed to create checkout session", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"invoice_id": invID,
		"payment_url": session.URL,
	})
}

// GetCheckoutSessions implements api.ServerInterface.
func (h *Handlers) GetCheckoutSessions(ctx echo.Context, params api.GetCheckoutSessionsParams) error {
	limit := 10
	if params.Limit != nil {
		limit = *params.Limit
	}
	sessions, err := h.SC.ListCheckoutSessions(ctx.Request().Context(), limit)
	if err != nil {
		h.Logger.Error("failed to list checkout sessions", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	
	return ctx.JSON(http.StatusOK, sessions)
}

// GetInvoices implements api.ServerInterface.
func (h *Handlers) GetInvoices(ctx echo.Context, params api.GetInvoicesParams) error {
	limit := 10
	if params.Limit != nil {
		limit = *params.Limit
	}
	invoices, err := h.SC.ListInvoices(ctx.Request().Context(), limit)
	if err != nil {
		h.Logger.Error("failed to list invoices", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return ctx.JSON(http.StatusOK, invoices)
}

// PostWebhookInvoicePaid implements api.ServerInterface.
func (h *Handlers) PostWebhookInvoicePaid(ctx echo.Context, params api.PostWebhookInvoicePaidParams) error {
	payload, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	sig := ctx.Request().Header.Get("Stripe-Signature")
	
	invoice, err := h.SC.HandleWebhook(ctx.Request().Context(), payload, sig)
	if err != nil {
		h.Logger.Error("webhook handling failed", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	
	h.Logger.Info("Invoice Paid", zap.Any("invoice", invoice))
	
	return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) Setup(e *echo.Echo) {
	swagger, err := api.GetSwagger()
	if err != nil {
		h.Logger.Error("failed to get swagger", zap.Error(err))
		panic(err)
	}

	e.Use(oapiMiddleware.OapiRequestValidator(swagger))

	api.RegisterHandlers(e, h)
}
