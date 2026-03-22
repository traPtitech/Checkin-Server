package router

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	oapiMiddleware "github.com/oapi-codegen/echo-middleware"
	"github.com/stripe/stripe-go/v84"
	"github.com/traPtitech/Checkin-Server/middleware"
	"github.com/traPtitech/Checkin-Server/repository"
	stripeservice "github.com/traPtitech/Checkin-Server/service/stripe"
	traqservice "github.com/traPtitech/Checkin-Server/service/traq"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

type Handlers struct {
	Logger               *zap.Logger
	Repo                 *repository.Queries
	SC                   stripeservice.Service
	TC                   traqservice.Service
	Mailer               Mailer
	JWTConfig            *middleware.JWTConfig
	AdminTraQIDs         map[string]struct{}
	PublicAPIBaseURL     string
	RequireHTTPS         bool
	VerificationTokenTTL time.Duration
}

// normalizeEmail normalizes an email address
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// hashEmail creates a SHA256 hash of an email address
func hashEmail(email string) string {
	email = normalizeEmail(email)
	hash := sha256.Sum256([]byte(email))
	return hex.EncodeToString(hash[:])
}

// stringPtr returns a pointer to the string value, or nil if the string is empty
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// clampStripeLimit clamps limit to Stripe's allowed range 1..100 for list endpoints.
func clampStripeLimit(limit int) int {
	if limit < 1 {
		return 1
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func requiresAdmin(method, path string) bool {
	switch path {
	case "/list/invoices", "/list/checkout-sessions":
		return method == http.MethodGet
	case "/admin":
		return method == http.MethodGet
	default:
		return false
	}
}

func (h *Handlers) validateTraQID(ctx echo.Context, traqID *string) error {
	if traqID == nil {
		return nil
	}
	if !traQIDPattern.MatchString(strings.TrimSpace(*traqID)) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid traQ ID")
	}
	exists, err := h.TC.UserExistsByName(ctx.Request().Context(), *traqID)
	if err != nil {
		h.Logger.Error("failed to validate traQ id", zap.String("traq_id", *traqID), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate traQ ID")
	}
	if !exists {
		return echo.NewHTTPError(http.StatusBadRequest, "traQ ID does not exist")
	}
	return nil
}

func getTraQIDFromContext(ctx echo.Context) string {
	if traqID, ok := middleware.GetTraQID(ctx); ok {
		return strings.TrimSpace(traqID)
	}
	return strings.TrimSpace(ctx.Request().Header.Get("X-Forwarded-User"))
}

// DeleteAdmin is retained for compatibility with older callers.
func (h *Handlers) DeleteAdmin(ctx echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "admins are managed by ADMIN_TRAQ_IDS environment variable")
}

// GetAdmins implements api.ServerInterface.
func (h *Handlers) GetAdmins(ctx echo.Context) error {
	admins := make([]api.Admin, 0, len(h.AdminTraQIDs))
	for id := range h.AdminTraQIDs {
		admins = append(admins, api.Admin{Id: id})
	}
	sort.Slice(admins, func(i, j int) bool {
		return admins[i].Id < admins[j].Id
	})

	return ctx.JSON(http.StatusOK, admins)
}

// PostAdmin implements api.ServerInterface.
func (h *Handlers) PostAdmin(ctx echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "admins are managed by ADMIN_TRAQ_IDS environment variable")
}

// GetCustomer implements api.ServerInterface.
func (h *Handlers) GetCustomer(ctx echo.Context, params api.GetCustomerParams) error {
	ctxReq := ctx.Request().Context()
	subject, err := h.getExistingCustomerSubject(ctx)
	if err != nil {
		return err
	}

	if params.CustomerId != nil {
		if *params.CustomerId != subject.StripeCustomerID {
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
		cust, err := h.SC.GetCustomer(ctxReq, *params.CustomerId)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "customer not found")
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
	}

	if params.Email != nil {
		normalizedEmail := normalizeEmail(*params.Email)
		if subject.Email == "" || normalizedEmail != subject.Email {
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
		cust, err := h.SC.GetCustomer(ctxReq, subject.StripeCustomerID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
	}

	if params.TraqId != nil {
		if !traQIDPattern.MatchString(strings.TrimSpace(*params.TraqId)) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid traQ ID")
		}
		customers, err := h.SC.SearchCustomersByTraQID(ctxReq, *params.TraqId)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		if len(customers) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "customer not found")
		}
		// Verify ownership
		if customers[0].ID != subject.StripeCustomerID {
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(customers[0]))
	}

	return echo.NewHTTPError(http.StatusBadRequest, "one of customerId, email, or traqId is required")
}

// PatchCustomer implements api.ServerInterface.
func (h *Handlers) PatchCustomer(ctx echo.Context, params api.PatchCustomerParams) error {
	if err := middleware.ValidateCSRF(ctx, params.XCSRFToken); err != nil {
		return err
	}

	subject, err := h.getExistingCustomerSubject(ctx)
	if err != nil {
		return err
	}

	var body api.PatchCustomerJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if _, err := ensureEmailMatchesSubject(string(body.Email), subject); err != nil {
		return err
	}

	traqID := body.TraqId
	if traqID == nil {
		if proxyTraQID := getTraQIDFromContext(ctx); proxyTraQID != "" {
			traqID = stringPtr(proxyTraQID)
		}
	}
	if body.TraqId != nil {
		if err := h.validateTraQID(ctx, body.TraqId); err != nil {
			return err
		}
	}

	cust, err := h.SC.UpdateCustomer(ctx.Request().Context(), subject.StripeCustomerID, nil, stringPtr(body.Name), traqID)
	if err != nil {
		h.Logger.Error("failed to update stripe customer", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
}

// PostCustomer implements api.ServerInterface.
func (h *Handlers) PostCustomer(ctx echo.Context, params api.PostCustomerParams) error {
	if err := middleware.ValidateCSRF(ctx, params.XCSRFToken); err != nil {
		return err
	}

	subject, err := h.getAuthSubject(ctx)
	if err != nil {
		return err
	}

	var body api.PostCustomerJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	email, err := ensureEmailMatchesSubject(string(body.Email), subject)
	if err != nil {
		return err
	}
	mailHash := hashEmail(email)

	if subject.Source == authSourceProxy {
		if body.TraqId != nil && *body.TraqId != subject.TraQID {
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
		cust, err := h.SC.GetCustomer(ctx.Request().Context(), subject.StripeCustomerID)
		if err != nil {
			h.Logger.Error("failed to get stripe customer for proxy-authenticated request", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return ctx.JSON(http.StatusOK, mapStripeCustomerToResponse(cust))
	}

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

	traqID := body.TraqId
	if traqID == nil {
		if proxyTraQID := getTraQIDFromContext(ctx); proxyTraQID != "" {
			traqID = stringPtr(proxyTraQID)
		}
	}
	if body.TraqId != nil {
		if err := h.validateTraQID(ctx, body.TraqId); err != nil {
			return err
		}
	}

	customers, err := h.SC.SearchCustomersByEmail(ctx.Request().Context(), email)
	if err != nil {
		h.Logger.Error("failed to search customers by email", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var targetCustomer *stripe.Customer
	createdByRequest := false
	if len(customers) > 0 {
		targetCustomer = customers[0]
	} else {
		targetCustomer, err = h.SC.CreateCustomer(ctx.Request().Context(), &email, stringPtr(body.Name), traqID)
		if err != nil {
			h.Logger.Error("failed to create stripe customer", zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		createdByRequest = true
	}

	err = h.Repo.CreateUser(ctx.Request().Context(), repository.CreateUserParams{
		ID:               targetCustomer.ID,
		MailHash:         mailHash,
		StripeCustomerID: targetCustomer.ID,
	})
	if err != nil {
		if createdByRequest {
			if _, delErr := h.SC.DeleteCustomer(ctx.Request().Context(), targetCustomer.ID); delErr != nil {
				h.Logger.Error("failed to delete stripe customer during rollback", zap.Error(delErr))
			}
		}
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
func (h *Handlers) PostInvoice(ctx echo.Context, params api.PostInvoiceParams) error {
	if err := middleware.ValidateCSRF(ctx, params.XCSRFToken); err != nil {
		return err
	}

	subject, err := h.getExistingCustomerSubject(ctx)
	if err != nil {
		return err
	}

	var body api.PostInvoiceJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if body.ProductId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "product_id is required")
	}
	if body.CustomerId != "" && body.CustomerId != subject.StripeCustomerID {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	invID, err := h.SC.CreateInvoice(ctx.Request().Context(), subject.StripeCustomerID, body.ProductId)
	if err != nil {
		h.Logger.Error("failed to create invoice", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	session, err := h.SC.CreateCheckoutSession(ctx.Request().Context(), invID)
	if err != nil {
		h.Logger.Error("failed to create checkout session", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, api.CreateInvoiceResponse{
		InvoiceId:  invID,
		PaymentUrl: session.URL,
	})
}

// GetCheckoutSessions implements api.ServerInterface.
func (h *Handlers) GetCheckoutSessions(ctx echo.Context, params api.GetCheckoutSessionsParams) error {
	req := stripeservice.ListCheckoutSessionsParams{
		Limit: 10,
	}
	if params.Limit != nil {
		req.Limit = clampStripeLimit(*params.Limit)
	}
	req.CustomerID = params.CustomerId
	req.SubscriptionID = params.SubscriptionId
	req.StartingAfter = params.StartingAfter
	req.EndingBefore = params.EndingBefore
	if params.PaymentIntentId != nil {
		req.PaymentIntentID = params.PaymentIntentId
	}
	if params.Status != nil {
		status := string(*params.Status)
		req.Status = &status
	}

	sessions, err := h.SC.ListCheckoutSessions(ctx.Request().Context(), req)
	if err != nil {
		h.Logger.Error("failed to list checkout sessions", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, sessions)
}

// GetInvoices implements api.ServerInterface.
func (h *Handlers) GetInvoices(ctx echo.Context, params api.GetInvoicesParams) error {
	req := stripeservice.ListInvoicesParams{
		Limit: 10,
	}
	if params.Limit != nil {
		req.Limit = clampStripeLimit(*params.Limit)
	}
	req.CustomerID = params.CustomerId
	req.SubscriptionID = params.SubscriptionId
	req.StartingAfter = params.StartingAfter
	req.EndingBefore = params.EndingBefore
	if params.Status != nil {
		status := string(*params.Status)
		req.Status = &status
	}
	if params.CollectionMethod != nil {
		method := string(*params.CollectionMethod)
		req.CollectionMethod = &method
	}

	invoices, err := h.SC.ListInvoices(ctx.Request().Context(), req)
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

	if invoice.Data != nil && len(*invoice.Data) > 0 {
		h.Logger.Info("Invoice Paid", zap.Any("invoice", invoice))
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *Handlers) Setup(e *echo.Echo) {
	swagger, err := api.GetSwagger()
	if err != nil {
		h.Logger.Error("failed to get swagger", zap.Error(err))
		panic(err)
	}

	e.Use(oapiMiddleware.OapiRequestValidatorWithOptions(swagger, &oapiMiddleware.Options{
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return path == "/webhook/invoice-paid"
		},
	}))
	e.Use(middleware.TraQHeaderMiddleware())
	adminMiddleware := middleware.AdminMiddleware(h.AdminTraQIDs)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			method := c.Request().Method

			if requiresAdmin(method, path) {
				return adminMiddleware(next)(c)
			}

			return next(c)
		}
	})

	api.RegisterHandlers(e, h)
}
