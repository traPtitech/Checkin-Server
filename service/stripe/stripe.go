package stripe

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

// StripeService はStripe APIを使用した支払いサービス実装
type StripeService struct {
	logger        *zap.Logger
	webhookSecret string
}

// CreateInvoice implements Service. ドラフトのInvoiceを作成する。確定はしない。
func (s *StripeService) CreateInvoice(ctx context.Context, customerID string, priceID string) (string, error) {
	if customerID == "" || priceID == "" {
		return "", fmt.Errorf("customerID and priceID are required")
	}
	invParams := &stripe.InvoiceParams{Customer: stripe.String(customerID)}
	invParams.Context = ctx
	inv, err := invoice.New(invParams)
	if err != nil {
		s.logger.Error("failed to create Stripe invoice", zap.Error(err))
		return "", err
	}
	itemParams := &stripe.InvoiceItemParams{
		Customer: stripe.String(customerID),
		Invoice:  stripe.String(inv.ID),
		Price:    stripe.String(priceID),
	}
	itemParams.Context = ctx
	if _, err := invoiceitem.New(itemParams); err != nil {
		s.logger.Error("failed to add invoice item", zap.Error(err))
		return "", err
	}
	return inv.ID, nil
}

// CreateCheckoutSession implements Service. 指定したドラフトInvoiceを確定し、決済用URLを返します。
func (s *StripeService) CreateCheckoutSession(ctx context.Context, invoiceID string) (*CheckoutSession, error) {
	if invoiceID == "" {
		return nil, fmt.Errorf("invoiceID is required")
	}
	finalParams := &stripe.InvoiceFinalizeInvoiceParams{}
	finalParams.Context = ctx
	inv, err := invoice.FinalizeInvoice(invoiceID, finalParams)
	if err != nil {
		s.logger.Error("failed to finalize Stripe invoice", zap.String("invoice_id", invoiceID), zap.Error(err))
		return nil, err
	}
	return &CheckoutSession{
		ID:        inv.ID,
		URL:       inv.HostedInvoiceURL,
		InvoiceID: inv.ID,
	}, nil
}

// GetPaymentStatus implements Service.
func (s *StripeService) GetPaymentStatus(ctx context.Context, paymentID string) (string, error) {
	panic("unimplemented")
}

// HandleWebhook implements Service.
func (s *StripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) (api.Invoice, error) {
	panic("unimplemented")
}

// GetCustomer は顧客IDからStripeの顧客情報を取得します
func (s *StripeService) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customerID is required")
	}

	params := &stripe.CustomerParams{}
	params.Context = ctx

	cust, err := customer.Get(customerID, params)
	if err != nil {
		s.logger.Error("failed to get Stripe customer", zap.String("customer_id", customerID), zap.Error(err))
		return nil, err
	}

	return cust, nil
}

// SearchCustomersByEmail はメールアドレスで顧客情報を検索します
func (s *StripeService) SearchCustomersByEmail(ctx context.Context, email string) ([]*stripe.Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("email", "", email)
	params.Context = ctx

	var customers []*stripe.Customer
	i := customer.List(params)
	for i.Next() {
		customers = append(customers, i.Customer())
	}
	if err := i.Err(); err != nil {
		s.logger.Error("failed to list Stripe customers by email", zap.String("email", email), zap.Error(err))
		return nil, err
	}

	return customers, nil
}

// SearchCustomersByTraQID はメタデータで顧客情報を検索します(traQIDでの検索を想定)
func (s *StripeService) SearchCustomersByTraQID(ctx context.Context, traQID string) ([]*stripe.Customer, error) {
	if traQID == "" {
		return nil, fmt.Errorf("traQID is required")
	}

	params := &stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf("metadata['traQID']:'%s'", traQID),
		},
	}
	params.Context = ctx

	it := customer.Search(params)
	var customers []*stripe.Customer
	for it.Next() {
		customers = append(customers, it.Customer())
	}
	if err := it.Err(); err != nil {
		s.logger.Error("failed to search Stripe customers by metadata", zap.String("key", "traQID"), zap.String("value", traQID), zap.Error(err))
		return nil, err
	}
	return customers, nil
}

// ListInvoices lists invoices.
func (s *StripeService) ListInvoices(ctx context.Context, limit int) ([]*stripe.Invoice, error) {
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}
	params := &stripe.InvoiceListParams{}
	params.Limit = stripe.Int64(int64(limit))
	params.Context = ctx
	iter := invoice.List(params)
	var invoices []*stripe.Invoice
	for iter.Next() {
		invoices = append(invoices, iter.Invoice())
	}
	return invoices, iter.Err()
}

// ListCheckoutSessions lists checkout sessions.
func (s *StripeService) ListCheckoutSessions(ctx context.Context, limit int) ([]*stripe.CheckoutSession, error) {
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}
	params := &stripe.CheckoutSessionListParams{}
	params.Limit = stripe.Int64(int64(limit))
	params.Context = ctx
	iter := session.List(params)
	var sessions []*stripe.CheckoutSession
	for iter.Next() {
		sessions = append(sessions, iter.CheckoutSession())
	}
	return sessions, iter.Err()
}

// CreateCustomer は新しい顧客を作成します
func (s *StripeService) CreateCustomer(ctx context.Context, email, name, traQID *string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{}
	if email != nil {
		params.Email = stripe.String(*email)
	}
	if name != nil {
		params.Name = stripe.String(*name)
	}
	if traQID != nil {
		if params.Metadata == nil {
			params.Metadata = make(map[string]string)
		}
		params.Metadata["traQID"] = *traQID
	}
	params.Context = ctx

	cust, err := customer.New(params)
	if err != nil {
		s.logger.Error("failed to create Stripe customer", zap.Error(err))
		return nil, err
	}

	return cust, nil
}

// UpdateCustomer は顧客情報を更新します
func (s *StripeService) UpdateCustomer(ctx context.Context, customerID string, email, name, traQID *string) (*stripe.Customer, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customerID is required")
	}

	params := &stripe.CustomerParams{}
	if email != nil {
		params.Email = stripe.String(*email)
	}
	if name != nil {
		params.Name = stripe.String(*name)
	}
	if traQID != nil {
		if params.Metadata == nil {
			params.Metadata = make(map[string]string)
		}
		params.Metadata["traQID"] = *traQID
	}
	params.Context = ctx

	cust, err := customer.Update(customerID, params)
	if err != nil {
		s.logger.Error("failed to update Stripe customer", zap.String("customer_id", customerID), zap.Error(err))
		return nil, err
	}

	return cust, nil
}

// UpdateCustomerTraQID は顧客のメタデータにあるtraQIDのみを更新します
func (s *StripeService) UpdateCustomerTraQID(ctx context.Context, customerID string, traQID string) (*stripe.Customer, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customerID is required")
	}
	if traQID == "" {
		return nil, fmt.Errorf("traQID is required")
	}

	params := &stripe.CustomerParams{}
	params.Metadata = map[string]string{"traQID": traQID}
	params.Context = ctx

	cust, err := customer.Update(customerID, params)
	if err != nil {
		s.logger.Error("failed to update Stripe customer traQID", zap.String("customer_id", customerID), zap.Error(err))
		return nil, err
	}

	return cust, nil
}

// DeleteCustomer は顧客を削除します
func (s *StripeService) DeleteCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customerID is required")
	}

	params := &stripe.CustomerParams{}
	params.Context = ctx

	cust, err := customer.Del(customerID, params)
	if err != nil {
		s.logger.Error("failed to delete Stripe customer", zap.String("customer_id", customerID), zap.Error(err))
		return nil, err
	}
	return cust, nil
}

// NewStripeService は新しいStripeServiceインスタンスを作成します。
// logger が nil の場合は zap.NewNop() を使用し、ログ出力は行われません。
func NewStripeService(logger *zap.Logger) (Service, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	apiKey := os.Getenv("STRIPE_SECRET_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("STRIPE_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("STRIPE_SECRET_KEY or STRIPE_API_KEY is not set")
	}

	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		logger.Warn("STRIPE_WEBHOOK_SECRET is not set, webhook verification will fail")
	}

	stripe.Key = apiKey

	return &StripeService{
		logger:        logger,
		webhookSecret: webhookSecret,
	}, nil
}
