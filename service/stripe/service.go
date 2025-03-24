package stripe

import (
	"context"

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
}

// CheckoutSession は決済セッション情報を表します
type CheckoutSession struct {
	ID        string
	URL       string
	ExpiresAt int64
	InvoiceID string
}
