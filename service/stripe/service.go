package stripe

import (
	"context"

	stripeapi "github.com/stripe/stripe-go/v81"
	api "github.com/traPtitech/Checkin-openapi/server"
)

// Service はStripe処理のインターフェース
type Service interface {
	// CreateCheckoutSession は決済セッションを作成します
	CreateCheckoutSession(ctx context.Context, invoice *api.Invoice) (*CheckoutSession, error)

	// GetPaymentStatus は支払いステータスを取得します
	GetPaymentStatus(ctx context.Context, paymentID string) (string, error)

	// HandleWebhook はWebhookイベントを処理します
	HandleWebhook(ctx context.Context, payload []byte, signature string) (api.Invoice, error)

	// GetCustomer はStripeの顧客情報を取得します
	GetCustomer(ctx context.Context, customerID string) (*stripeapi.Customer, error)

	// SearchCustomersByEmail はメールアドレスで顧客情報を検索します
	SearchCustomersByEmail(ctx context.Context, email string) ([]*stripeapi.Customer, error)

	// SearchCustomersByTraQID はメタデータにあるtraQIDで顧客情報を検索します
	SearchCustomersByTraQID(ctx context.Context, traQID string) ([]*stripeapi.Customer, error)

	// CreateCustomer は新しい顧客を作成します
	CreateCustomer(ctx context.Context, email, name, traQID *string) (*stripeapi.Customer, error)

	// UpdateCustomerTraQID は顧客のメタデータにあるtraQIDを更新します
	UpdateCustomerTraQID(ctx context.Context, customerID string, email, name, traQID *string) (*stripeapi.Customer, error)
}

// CheckoutSession は決済セッション情報を表します
type CheckoutSession struct {
	ID        string
	URL       string
	ExpiresAt int64
	InvoiceID string
}
