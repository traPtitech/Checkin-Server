package router

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v84"
	"github.com/traPtitech/Checkin-Server/middleware"
	"github.com/traPtitech/Checkin-Server/repository"
	stripeservice "github.com/traPtitech/Checkin-Server/service/stripe"
	traqservice "github.com/traPtitech/Checkin-Server/service/traq"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

type stubMailer struct {
	sendFn func(context.Context, string, string) error
}

func (m stubMailer) SendVerificationEmail(ctx context.Context, email string, verificationURL string) error {
	if m.sendFn == nil {
		return nil
	}
	return m.sendFn(ctx, email, verificationURL)
}

type stubStripeService struct {
	createInvoiceFn           func(context.Context, string, string) (string, error)
	createCheckoutSessionFn   func(context.Context, string) (*stripeservice.CheckoutSession, error)
	getPaymentStatusFn        func(context.Context, string) (string, error)
	handleWebhookFn           func(context.Context, []byte, string) (api.Invoice, error)
	getCustomerFn             func(context.Context, string) (*stripe.Customer, error)
	searchCustomersByEmailFn  func(context.Context, string) ([]*stripe.Customer, error)
	searchCustomersByTraQIDFn func(context.Context, string) ([]*stripe.Customer, error)
	createCustomerFn          func(context.Context, *string, *string, *string) (*stripe.Customer, error)
	updateCustomerFn          func(context.Context, string, *string, *string, *string) (*stripe.Customer, error)
	updateCustomerTraQIDFn    func(context.Context, string, string) (*stripe.Customer, error)
	deleteCustomerFn          func(context.Context, string) (*stripe.Customer, error)
	listInvoicesFn            func(context.Context, stripeservice.ListInvoicesParams) ([]*stripe.Invoice, error)
	listCheckoutSessionsFn    func(context.Context, stripeservice.ListCheckoutSessionsParams) ([]*stripe.CheckoutSession, error)
}

func (s stubStripeService) CreateInvoice(ctx context.Context, customerID string, productID string) (string, error) {
	if s.createInvoiceFn == nil {
		return "", fmt.Errorf("unexpected CreateInvoice call")
	}
	return s.createInvoiceFn(ctx, customerID, productID)
}

func (s stubStripeService) CreateCheckoutSession(ctx context.Context, invoiceID string) (*stripeservice.CheckoutSession, error) {
	if s.createCheckoutSessionFn == nil {
		return nil, fmt.Errorf("unexpected CreateCheckoutSession call")
	}
	return s.createCheckoutSessionFn(ctx, invoiceID)
}

func (s stubStripeService) GetPaymentStatus(ctx context.Context, paymentID string) (string, error) {
	if s.getPaymentStatusFn == nil {
		return "", fmt.Errorf("unexpected GetPaymentStatus call")
	}
	return s.getPaymentStatusFn(ctx, paymentID)
}

func (s stubStripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) (api.Invoice, error) {
	if s.handleWebhookFn == nil {
		return api.Invoice{}, fmt.Errorf("unexpected HandleWebhook call")
	}
	return s.handleWebhookFn(ctx, payload, signature)
}

func (s stubStripeService) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	if s.getCustomerFn == nil {
		return nil, fmt.Errorf("unexpected GetCustomer call")
	}
	return s.getCustomerFn(ctx, customerID)
}

func (s stubStripeService) SearchCustomersByEmail(ctx context.Context, email string) ([]*stripe.Customer, error) {
	if s.searchCustomersByEmailFn == nil {
		return nil, fmt.Errorf("unexpected SearchCustomersByEmail call")
	}
	return s.searchCustomersByEmailFn(ctx, email)
}

func (s stubStripeService) SearchCustomersByTraQID(ctx context.Context, traqID string) ([]*stripe.Customer, error) {
	if s.searchCustomersByTraQIDFn == nil {
		return nil, fmt.Errorf("unexpected SearchCustomersByTraQID call")
	}
	return s.searchCustomersByTraQIDFn(ctx, traqID)
}

func (s stubStripeService) CreateCustomer(ctx context.Context, email, name, traqID *string) (*stripe.Customer, error) {
	if s.createCustomerFn == nil {
		return nil, fmt.Errorf("unexpected CreateCustomer call")
	}
	return s.createCustomerFn(ctx, email, name, traqID)
}

func (s stubStripeService) UpdateCustomer(ctx context.Context, customerID string, email, name, traqID *string) (*stripe.Customer, error) {
	if s.updateCustomerFn == nil {
		return nil, fmt.Errorf("unexpected UpdateCustomer call")
	}
	return s.updateCustomerFn(ctx, customerID, email, name, traqID)
}

func (s stubStripeService) UpdateCustomerTraQID(ctx context.Context, customerID string, traqID string) (*stripe.Customer, error) {
	if s.updateCustomerTraQIDFn == nil {
		return nil, fmt.Errorf("unexpected UpdateCustomerTraQID call")
	}
	return s.updateCustomerTraQIDFn(ctx, customerID, traqID)
}

func (s stubStripeService) DeleteCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	if s.deleteCustomerFn == nil {
		return nil, fmt.Errorf("unexpected DeleteCustomer call")
	}
	return s.deleteCustomerFn(ctx, customerID)
}

func (s stubStripeService) ListInvoices(ctx context.Context, params stripeservice.ListInvoicesParams) ([]*stripe.Invoice, error) {
	if s.listInvoicesFn == nil {
		return nil, fmt.Errorf("unexpected ListInvoices call")
	}
	return s.listInvoicesFn(ctx, params)
}

func (s stubStripeService) ListCheckoutSessions(ctx context.Context, params stripeservice.ListCheckoutSessionsParams) ([]*stripe.CheckoutSession, error) {
	if s.listCheckoutSessionsFn == nil {
		return nil, fmt.Errorf("unexpected ListCheckoutSessions call")
	}
	return s.listCheckoutSessionsFn(ctx, params)
}

type stubTraQService struct {
	userExistsByNameFn func(context.Context, string) (bool, error)
}

func (s stubTraQService) UserExistsByName(ctx context.Context, traqID string) (bool, error) {
	if s.userExistsByNameFn == nil {
		return false, fmt.Errorf("unexpected UserExistsByName call")
	}
	return s.userExistsByNameFn(ctx, traqID)
}

var _ stripeservice.Service = stubStripeService{}
var _ traqservice.Service = stubTraQService{}

func newScenarioServer(t *testing.T, db *sql.DB, stripeSvc stripeservice.Service, traqSvc traqservice.Service) (*echo.Echo, *middleware.JWTConfig) {
	t.Helper()

	e := echo.New()
	jwtConfig := &middleware.JWTConfig{
		SecretKey:       "test-secret",
		ExpirationHours: 2,
	}
	handlers := Handlers{
		Logger:               zap.NewNop(),
		Repo:                 repository.New(db),
		SC:                   stripeSvc,
		TC:                   traqSvc,
		Mailer:               stubMailer{},
		JWTConfig:            jwtConfig,
		PublicAPIBaseURL:     "http://localhost:5173/api",
		RequireHTTPS:         false,
		VerificationTokenTTL: 15 * time.Minute,
		AdminTraQIDs: map[string]struct{}{
			"admin-user": {},
		},
	}
	handlers.Setup(e)
	return e, jwtConfig
}

func performJSONRequest(t *testing.T, e *echo.Echo, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func performFormRequest(t *testing.T, e *echo.Echo, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func expectGetUserByMailHashNotFound(mock sqlmock.Sqlmock, mailHash string) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mail_hash, stripe_customer_id, created_at, updated_at FROM users WHERE mail_hash = ? LIMIT 1")).
		WithArgs(mailHash).
		WillReturnError(sql.ErrNoRows)
}

func expectGetUserByMailHash(mock sqlmock.Sqlmock, user repository.User) {
	rows := sqlmock.NewRows([]string{"id", "mail_hash", "stripe_customer_id", "created_at", "updated_at"}).
		AddRow(user.ID, user.MailHash, user.StripeCustomerID, user.CreatedAt, user.UpdatedAt)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mail_hash, stripe_customer_id, created_at, updated_at FROM users WHERE mail_hash = ? LIMIT 1")).
		WithArgs(user.MailHash).
		WillReturnRows(rows)
}

func expectCreateUser(mock sqlmock.Sqlmock, customerID, mailHash string) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, mail_hash, stripe_customer_id) VALUES (?, ?, ?)")).
		WithArgs(customerID, mailHash, customerID).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectDeleteUnusedEmailVerifications(mock sqlmock.Sqlmock, email string) {
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM email_verifications WHERE email = ? AND used_at IS NULL")).
		WithArgs(email).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectCreateEmailVerification(mock sqlmock.Sqlmock, email, redirect string) {
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO email_verifications (token_hash, email, redirect_path, expires_at) VALUES (?, ?, ?, ?)")).
		WithArgs(sqlmock.AnyArg(), email, redirect, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func expectGetEmailVerification(mock sqlmock.Sqlmock, verification repository.EmailVerification) {
	rows := sqlmock.NewRows([]string{"token_hash", "email", "redirect_path", "expires_at", "used_at", "created_at"}).
		AddRow(verification.TokenHash, verification.Email, verification.RedirectPath, verification.ExpiresAt, verification.UsedAt, verification.CreatedAt)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT token_hash, email, redirect_path, expires_at, used_at, created_at
FROM email_verifications
WHERE token_hash = ?
LIMIT 1`)).
		WithArgs(verification.TokenHash).
		WillReturnRows(rows)
}

func expectMarkEmailVerificationUsed(mock sqlmock.Sqlmock, tokenHash string) {
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE email_verifications
SET used_at = ?
WHERE token_hash = ? AND used_at IS NULL`)).
		WithArgs(sqlmock.AnyArg(), tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func sessionCookieHeader(t *testing.T, jwtConfig *middleware.JWTConfig, email, csrf string) string {
	t.Helper()

	token, err := jwtConfig.GenerateToken(email)
	require.NoError(t, err)
	return fmt.Sprintf("%s=%s; %s=%s", middleware.SessionCookieName, token, middleware.CSRFCookieName, csrf)
}

func testStripeCustomer(id, email, name string, traqID *string) *stripe.Customer {
	customer := &stripe.Customer{
		ID:    id,
		Email: email,
		Name:  name,
	}
	if traqID != nil {
		customer.Metadata = map[string]string{"traQID": *traqID}
	}
	return customer
}

func TestProtectedRoutesRequireAuthenticationAndAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	e, _ := newScenarioServer(t, db, stubStripeService{}, stubTraQService{})

	customerRec := performJSONRequest(t, e, http.MethodGet, "/customer?email=test@isct.ac.jp", nil, nil)
	require.Equal(t, http.StatusUnauthorized, customerRec.Code)
	require.JSONEq(t, `{"message":"authentication required"}`, customerRec.Body.String())

	adminRec := performJSONRequest(t, e, http.MethodGet, "/admin", nil, nil)
	require.Equal(t, http.StatusForbidden, adminRec.Code)
	require.JSONEq(t, `{"message":"admin only"}`, adminRec.Body.String())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostVerifyEmailReturnsAcceptedResponse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	e, _ := newScenarioServer(t, db, stubStripeService{}, stubTraQService{})
	expectDeleteUnusedEmailVerifications(mock, "test@isct.ac.jp")
	expectCreateEmailVerification(mock, "test@isct.ac.jp", "/payments")

	rec := performJSONRequest(t, e, http.MethodPost, "/verify-email?redirect=/payments", map[string]string{
		"email": "test@isct.ac.jp",
	}, nil)

	require.Equal(t, http.StatusAccepted, rec.Code)
	var res api.VerifyEmailStartResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Equal(t, "test@isct.ac.jp", res.Email)
	require.Equal(t, "/payments", res.Redirect)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetVerifyEmailConfirmRendersInterstitialPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	e, _ := newScenarioServer(t, db, stubStripeService{}, stubTraQService{})
	rawToken := "abcdefghijklmnopqrstuvwxyz0123456789"
	tokenHash := hashVerificationToken(rawToken)
	now := time.Now()

	expectGetEmailVerification(mock, repository.EmailVerification{
		TokenHash:    tokenHash,
		Email:        "student@isct.ac.jp",
		RedirectPath: "/membership",
		ExpiresAt:    now.Add(10 * time.Minute),
		UsedAt:       sql.NullTime{},
		CreatedAt:    now,
	})
	rec := performJSONRequest(t, e, http.MethodGet, "/verify-email/confirm?token="+rawToken, nil, nil)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "確認して続行")
	require.Contains(t, rec.Body.String(), rawToken)
	require.Empty(t, rec.Result().Cookies())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostVerifyEmailConfirmSetsCookiesAndRedirects(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	e, _ := newScenarioServer(t, db, stubStripeService{}, stubTraQService{})
	rawToken := "abcdefghijklmnopqrstuvwxyz0123456789"
	tokenHash := hashVerificationToken(rawToken)
	now := time.Now()

	expectGetEmailVerification(mock, repository.EmailVerification{
		TokenHash:    tokenHash,
		Email:        "student@isct.ac.jp",
		RedirectPath: "/membership",
		ExpiresAt:    now.Add(10 * time.Minute),
		UsedAt:       sql.NullTime{},
		CreatedAt:    now,
	})
	expectMarkEmailVerificationUsed(mock, tokenHash)

	rec := performFormRequest(t, e, http.MethodPost, "/verify-email/confirm", "token="+rawToken, nil)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/membership", rec.Header().Get("Location"))
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 2)
	var cookieNames []string
	for _, cookie := range cookies {
		cookieNames = append(cookieNames, cookie.Name)
	}
	require.Contains(t, cookieNames, middleware.SessionCookieName)
	require.Contains(t, cookieNames, middleware.CSRFCookieName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostVerifyEmailRejectsInvalidRedirect(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	e, _ := newScenarioServer(t, db, stubStripeService{}, stubTraQService{})

	rec := performJSONRequest(t, e, http.MethodPost, "/verify-email?redirect=https://example.com", map[string]string{
		"email": "test@isct.ac.jp",
	}, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `string doesn't match the regular expression`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostCustomerCreatesUserWithForwardedTraQID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mailHash := hashEmail("new-user@isct.ac.jp")
	expectGetUserByMailHashNotFound(mock, mailHash)
	expectGetUserByMailHashNotFound(mock, mailHash)
	expectCreateUser(mock, "cus_new", mailHash)

	var seenTraQID *string
	stripeSvc := stubStripeService{
		searchCustomersByTraQIDFn: func(context.Context, string) ([]*stripe.Customer, error) {
			return []*stripe.Customer{}, nil
		},
		searchCustomersByEmailFn: func(context.Context, string) ([]*stripe.Customer, error) {
			return []*stripe.Customer{}, nil
		},
		createCustomerFn: func(_ context.Context, email, name, traqID *string) (*stripe.Customer, error) {
			seenTraQID = traqID
			require.NotNil(t, email)
			require.NotNil(t, name)
			require.Equal(t, "new-user@isct.ac.jp", *email)
			require.Equal(t, "New User", *name)
			return testStripeCustomer("cus_new", *email, *name, traqID), nil
		},
	}
	e, jwtConfig := newScenarioServer(t, db, stripeSvc, stubTraQService{})
	csrf := "csrf-create-customer-token"

	rec := performJSONRequest(t, e, http.MethodPost, "/customer", map[string]string{
		"email": "new-user@isct.ac.jp",
		"name":  "New User",
	}, map[string]string{
		"X-Forwarded-User": "traq-user",
		"X-CSRF-Token":     csrf,
		"Cookie":           sessionCookieHeader(t, jwtConfig, "new-user@isct.ac.jp", csrf),
	})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, seenTraQID)
	require.Equal(t, "traq-user", *seenTraQID)
	require.JSONEq(t, `{"id":"cus_new","email":"new-user@isct.ac.jp","name":"New User","traq_id":"traq-user"}`, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPatchCustomerUsesSessionCookieAndForwardedTraQID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	user := repository.User{
		ID:               "cus_existing",
		MailHash:         hashEmail("member@isct.ac.jp"),
		StripeCustomerID: "cus_existing",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	expectGetUserByMailHash(mock, user)

	var seenCustomerID string
	var seenName *string
	var seenTraQID *string
	stripeSvc := stubStripeService{
		searchCustomersByTraQIDFn: func(context.Context, string) ([]*stripe.Customer, error) {
			return []*stripe.Customer{}, nil
		},
		updateCustomerFn: func(_ context.Context, customerID string, email, name, traqID *string) (*stripe.Customer, error) {
			seenCustomerID = customerID
			seenName = name
			seenTraQID = traqID
			return testStripeCustomer(customerID, "member@isct.ac.jp", *name, traqID), nil
		},
	}
	e, jwtConfig := newScenarioServer(t, db, stripeSvc, stubTraQService{})
	csrf := "csrf-patch-customer-token"

	rec := performJSONRequest(t, e, http.MethodPatch, "/customer", map[string]string{
		"email": "member@isct.ac.jp",
		"name":  "Updated Name",
	}, map[string]string{
		"X-Forwarded-User": "proxy-traq",
		"X-CSRF-Token":     csrf,
		"Cookie":           sessionCookieHeader(t, jwtConfig, "member@isct.ac.jp", csrf),
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "cus_existing", seenCustomerID)
	require.NotNil(t, seenName)
	require.Equal(t, "Updated Name", *seenName)
	require.NotNil(t, seenTraQID)
	require.Equal(t, "proxy-traq", *seenTraQID)
	require.JSONEq(t, `{"id":"cus_existing","email":"member@isct.ac.jp","name":"Updated Name","traq_id":"proxy-traq"}`, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostInvoiceUsesSessionCookieAndReturnsPaymentURL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	user := repository.User{
		ID:               "cus_invoice",
		MailHash:         hashEmail("invoice-user@isct.ac.jp"),
		StripeCustomerID: "cus_invoice",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	expectGetUserByMailHash(mock, user)

	stripeSvc := stubStripeService{
		createInvoiceFn: func(_ context.Context, customerID string, productID string) (string, error) {
			require.Equal(t, "cus_invoice", customerID)
			require.Equal(t, "prod_123", productID)
			return "in_123", nil
		},
		createCheckoutSessionFn: func(_ context.Context, invoiceID string) (*stripeservice.CheckoutSession, error) {
			require.Equal(t, "in_123", invoiceID)
			return &stripeservice.CheckoutSession{
				ID:        "cs_123",
				URL:       "https://pay.example.test/in_123",
				InvoiceID: "in_123",
			}, nil
		},
	}
	e, jwtConfig := newScenarioServer(t, db, stripeSvc, stubTraQService{})
	csrf := "csrf-invoice-token"

	rec := performJSONRequest(t, e, http.MethodPost, "/invoice", map[string]string{
		"customer_id": "cus_invoice",
		"product_id":  "prod_123",
	}, map[string]string{
		"X-CSRF-Token": csrf,
		"Cookie":       sessionCookieHeader(t, jwtConfig, "invoice-user@isct.ac.jp", csrf),
	})

	require.Equal(t, http.StatusCreated, rec.Code)
	require.JSONEq(t, `{"invoice_id":"in_123","payment_url":"https://pay.example.test/in_123"}`, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWebhookRouteSkipsOpenAPIValidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	stripeSvc := stubStripeService{
		handleWebhookFn: func(_ context.Context, payload []byte, signature string) (api.Invoice, error) {
			require.Equal(t, "test-signature", signature)
			require.JSONEq(t, `{"id":"evt_test","object":"event","type":"invoice.paid","data":{"object":{"id":"in_test"}}}`, string(payload))
			return api.Invoice{}, nil
		},
	}
	e, _ := newScenarioServer(t, db, stripeSvc, stubTraQService{})

	rec := performJSONRequest(t, e, http.MethodPost, "/webhook/invoice-paid", map[string]any{
		"id":     "evt_test",
		"object": "event",
		"type":   "invoice.paid",
		"data": map[string]any{
			"object": map[string]any{"id": "in_test"},
		},
	}, map[string]string{
		"Stripe-Signature": "test-signature",
	})

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInvoicesForwardsFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	var seen stripeservice.ListInvoicesParams
	stripeSvc := stubStripeService{
		listInvoicesFn: func(_ context.Context, params stripeservice.ListInvoicesParams) ([]*stripe.Invoice, error) {
			seen = params
			return []*stripe.Invoice{}, nil
		},
	}
	e, _ := newScenarioServer(t, db, stripeSvc, stubTraQService{})

	rec := performJSONRequest(t, e, http.MethodGet, "/list/invoices?customer_id=cus_123&subscription_id=sub_123&starting_after=in_prev&ending_before=in_next&status=paid&collection_method=send_invoice&limit=20", nil, map[string]string{
		"X-Forwarded-User": "admin-user",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 20, seen.Limit)
	require.NotNil(t, seen.CustomerID)
	require.Equal(t, "cus_123", *seen.CustomerID)
	require.NotNil(t, seen.SubscriptionID)
	require.Equal(t, "sub_123", *seen.SubscriptionID)
	require.NotNil(t, seen.StartingAfter)
	require.Equal(t, "in_prev", *seen.StartingAfter)
	require.NotNil(t, seen.EndingBefore)
	require.Equal(t, "in_next", *seen.EndingBefore)
	require.NotNil(t, seen.Status)
	require.Equal(t, "paid", *seen.Status)
	require.NotNil(t, seen.CollectionMethod)
	require.Equal(t, "send_invoice", *seen.CollectionMethod)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCheckoutSessionsForwardsFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	var seen stripeservice.ListCheckoutSessionsParams
	stripeSvc := stubStripeService{
		listCheckoutSessionsFn: func(_ context.Context, params stripeservice.ListCheckoutSessionsParams) ([]*stripe.CheckoutSession, error) {
			seen = params
			return []*stripe.CheckoutSession{}, nil
		},
	}
	e, _ := newScenarioServer(t, db, stripeSvc, stubTraQService{})

	rec := performJSONRequest(t, e, http.MethodGet, "/list/checkout-sessions?customer_id=cus_123&subscription_id=sub_123&payment_intent_id=pi_123&starting_after=cs_prev&ending_before=cs_next&status=complete&limit=15", nil, map[string]string{
		"X-Forwarded-User": "admin-user",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 15, seen.Limit)
	require.NotNil(t, seen.CustomerID)
	require.Equal(t, "cus_123", *seen.CustomerID)
	require.NotNil(t, seen.SubscriptionID)
	require.Equal(t, "sub_123", *seen.SubscriptionID)
	require.NotNil(t, seen.PaymentIntentID)
	require.Equal(t, "pi_123", *seen.PaymentIntentID)
	require.NotNil(t, seen.StartingAfter)
	require.Equal(t, "cs_prev", *seen.StartingAfter)
	require.NotNil(t, seen.EndingBefore)
	require.Equal(t, "cs_next", *seen.EndingBefore)
	require.NotNil(t, seen.Status)
	require.Equal(t, "complete", *seen.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}
