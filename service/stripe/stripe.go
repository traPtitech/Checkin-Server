package stripe

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/stripe/stripe-go/v81"
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

// CreateCheckoutSession implements Service.
func (s *StripeService) CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSession, error) {
	if req == nil || req.CustomerId == "" || req.ProductId == "" {
		return nil, fmt.Errorf("CustomerId and ProductId are required")
	}
	priceID, err := s.priceIDForCurrentTerm()
	if err != nil {
		return nil, err
	}
	invReq := &CreateInvoiceRequest{CustomerID: req.CustomerId, PriceID: priceID}
	result, err := s.CreateInvoice(ctx, invReq)
	if err != nil {
		return nil, err
	}
	return &CheckoutSession{
		ID:        result.ID,
		URL:       result.HostedInvoiceURL,
		ExpiresAt: result.ExpiresAt,
		InvoiceID: result.ID,
	}, nil
}

// CreateInvoice implements Service.
func (s *StripeService) CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*StripeInvoiceResult, error) {
	if req == nil || req.CustomerID == "" || req.PriceID == "" {
		return nil, fmt.Errorf("CustomerID and PriceID are required")
	}
	invParams := &stripe.InvoiceParams{Customer: stripe.String(req.CustomerID)}
	invParams.Context = ctx
	inv, err := invoice.New(invParams)
	if err != nil {
		s.logger.Error("failed to create Stripe invoice", zap.Error(err))
		return nil, err
	}
	itemParams := &stripe.InvoiceItemParams{
		Customer: stripe.String(req.CustomerID),
		Invoice:  stripe.String(inv.ID),
		Price:    stripe.String(req.PriceID),
	}
	itemParams.Context = ctx
	if _, err := invoiceitem.New(itemParams); err != nil {
		s.logger.Error("failed to add invoice item", zap.Error(err))
		return nil, err
	}
	finalParams := &stripe.InvoiceFinalizeInvoiceParams{}
	finalParams.Context = ctx
	inv, err = invoice.FinalizeInvoice(inv.ID, finalParams)
	if err != nil {
		s.logger.Error("failed to finalize Stripe invoice", zap.Error(err))
		return nil, err
	}
	return &StripeInvoiceResult{
		ID:              inv.ID,
		HostedInvoiceURL: inv.HostedInvoiceURL,
		ExpiresAt:       inv.DueDate,
	}, nil
}

// priceIDForCurrentTerm は現在の前期/後期に応じたPrice IDを環境変数から返す。前期: 4/1〜9/30、後期: 10/1〜翌3/31。
func (s *StripeService) priceIDForCurrentTerm() (string, error) {
	now := time.Now()
	month := now.Month()
	isFirstTerm := month >= 4 && month <= 9
	if isFirstTerm {
		if id := os.Getenv("STRIPE_PRICE_ID_FIRST_TERM"); id != "" {
			return id, nil
		}
		return "", fmt.Errorf("STRIPE_PRICE_ID_FIRST_TERM is not set")
	}
	if id := os.Getenv("STRIPE_PRICE_ID_SECOND_TERM"); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("STRIPE_PRICE_ID_SECOND_TERM is not set")
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

// NewStripeService は新しいStripeServiceインスタンスを作成します。
// logger が nil の場合は zap.NewNop() を使用し、ログ出力は行われません。
func NewStripeService(logger *zap.Logger) (Service, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	apiKey := os.Getenv("STRIPE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("STRIPE_API_KEY is not set")
	}

	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET is not set")
	}

	stripe.Key = apiKey

	return &StripeService{
		logger:        logger,
		webhookSecret: webhookSecret,
	}, nil
}
