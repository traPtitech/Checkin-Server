package stripe

import (
	"context"

	stripeapi "github.com/stripe/stripe-go/v81"
	api "github.com/traPtitech/Checkin-openapi/server"
)

// Service はStripe処理のインターフェース
type Service interface {
	// CreateInvoice はStripe上にInvoiceをドラフトで作成します（確定はしません）。productIDで指定したProductのデフォルトPriceで1件の明細を追加し、作成したInvoiceのIDを返します。
	CreateInvoice(ctx context.Context, customerID string, productID string) (string, error)

	// CreateCheckoutSession は指定したInvoiceを確定し、決済用のHostedInvoiceURLを持つCheckoutSessionを返します
	CreateCheckoutSession(ctx context.Context, invoiceID string) (*CheckoutSession, error)

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

	// UpdateCustomer は顧客情報を更新します
	UpdateCustomer(ctx context.Context, customerID string, email, name, traQID *string) (*stripeapi.Customer, error)

	// UpdateCustomerTraQID は顧客のメタデータにあるtraQIDのみを更新します
	UpdateCustomerTraQID(ctx context.Context, customerID string, traQID string) (*stripeapi.Customer, error)

	// DeleteCustomer は顧客を削除します
	DeleteCustomer(ctx context.Context, customerID string) (*stripeapi.Customer, error)

	ListInvoices(ctx context.Context, limit int) ([]*stripeapi.Invoice, error)
	ListCheckoutSessions(ctx context.Context, limit int) ([]*stripeapi.CheckoutSession, error)
}

// CheckoutSession は決済セッション情報を表します
type CheckoutSession struct {
	ID        string
	URL       string
	ExpiresAt int64
	InvoiceID string
}
