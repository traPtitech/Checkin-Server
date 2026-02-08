package stripe

import (
	"context"

	stripeapi "github.com/stripe/stripe-go/v81"
	api "github.com/traPtitech/Checkin-openapi/server"
)

// CreateCheckoutSessionRequest はチェックアウトセッション作成の入力
type CreateCheckoutSessionRequest struct {
	CustomerID string
	ProductID  string
}

// CreateInvoiceRequest はStripe Invoice作成の入力（内部用）
type CreateInvoiceRequest struct {
	CustomerID string
	PriceID    string
}

// StripeInvoiceResult はStripe Invoice作成結果
type StripeInvoiceResult struct {
	ID               string
	HostedInvoiceURL  string
	ExpiresAt        int64
}

// Service はStripe処理のインターフェース
type Service interface {
	// CreateCheckoutSession は決済セッションを作成します
	CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSession, error)

	// CreateInvoice はStripe上にInvoiceを作成します
	CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*StripeInvoiceResult, error)

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

	// UpdateCustomerTraQID は顧客のメタデータにあるtraQIDのみを更新します
	UpdateCustomerTraQID(ctx context.Context, customerID string, traQID string) (*stripeapi.Customer, error)
}

// CheckoutSession は決済セッション情報を表します
type CheckoutSession struct {
	ID        string
	URL       string
	ExpiresAt int64
	InvoiceID string
}
