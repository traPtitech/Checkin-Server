package stripe

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

// StripeService はStripe APIを使用した支払いサービス実装
type StripeService struct {
	logger        *zap.Logger
	webhookSecret string
}

// CreateCheckoutSession implements Service.
func (s *StripeService) CreateCheckoutSession(ctx context.Context, invoice *api.Invoice) (*CheckoutSession, error) {
	panic("unimplemented")
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

	params := &stripe.CustomerListParams{}
	params.Filters.AddFilter("metadata[traQID]", "", traQID)
	params.Context = ctx

	var customers []*stripe.Customer
	i := customer.List(params)
	for i.Next() {
		customers = append(customers, i.Customer())
	}
	if err := i.Err(); err != nil {
		s.logger.Error("failed to list Stripe customers by metadata", zap.String("key", "traQID"), zap.String("value", traQID), zap.Error(err))
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

// UpdateCustomerTraQID は顧客のメタデータにあるtraQIDを更新します
func (s *StripeService) UpdateCustomerTraQID(ctx context.Context, customerID string, email, name, traQID *string) (*stripe.Customer, error) {
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
		s.logger.Error("failed to update Stripe customer metadata", zap.String("customer_id", customerID), zap.Error(err))
		return nil, err
	}

	return cust, nil
}

// NewStripeService は新しいStripeServiceインスタンスを作成します
func NewStripeService(logger *zap.Logger) (Service, error) {
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
