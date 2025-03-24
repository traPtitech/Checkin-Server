package stripe

import (
	"context"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v81"
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
