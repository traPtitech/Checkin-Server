package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	"github.com/stripe/stripe-go/v81/webhook"
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
	if paymentID == "" {
		return "", fmt.Errorf("paymentID is required")
	}
	params := &stripe.InvoiceParams{}
	params.Context = ctx
	inv, err := invoice.Get(paymentID, params)
	if err != nil {
		s.logger.Error("failed to get Stripe invoice", zap.String("payment_id", paymentID), zap.Error(err))
		return "", err
	}
	return string(inv.Status), nil
}

// HandleWebhook implements Service. invoice.paid イベントのみ処理し、api.Invoice に変換して返す。
func (s *StripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) (api.Invoice, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		s.logger.Error("webhook signature verification failed", zap.Error(err))
		return api.Invoice{}, err
	}
	if event.Type != stripe.EventTypeInvoicePaid {
		s.logger.Debug("webhook event ignored (not invoice.paid)", zap.String("event_type", string(event.Type)))
		return api.Invoice{}, nil
	}
	if event.Data == nil {
		return api.Invoice{}, fmt.Errorf("webhook event has no data")
	}
	var inv stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		s.logger.Error("failed to unmarshal invoice from webhook", zap.Error(err))
		return api.Invoice{}, err
	}
	// Line は 1 件のみ許容
	lineCount := 0
	if inv.Lines != nil {
		lineCount = len(inv.Lines.Data)
	}
	if lineCount != 1 {
		return api.Invoice{}, fmt.Errorf("invoice must have exactly one line item, got %d", lineCount)
	}
	line := inv.Lines.Data[0]

	// api.Invoice の Data 要素を組み立て（匿名構造体のため JSON 経由で構築）
	amountDue := inv.AmountDue
	amountPaid := inv.AmountPaid
	amountRemaining := inv.AmountRemaining
	created := inv.Created
	id := inv.ID
	status := api.InvoiceDataStatus(inv.Status)
	var paymentIntent *string
	if inv.PaymentIntent != nil {
		paymentIntent = &inv.PaymentIntent.ID
	}
	var productID *string
	if line.Price != nil && line.Price.Product != nil {
		productID = &line.Price.Product.ID
	}

	var apiCustomer *api.Customer
	if inv.Customer != nil && inv.Customer.ID != "" {
		custParams := &stripe.CustomerParams{}
		custParams.Context = ctx
		cust, err := customer.Get(inv.Customer.ID, custParams)
		if err != nil {
			s.logger.Error("failed to get customer for webhook invoice", zap.String("customer_id", inv.Customer.ID), zap.Error(err))
			return api.Invoice{}, err
		}
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
		apiCustomer = &api.Customer{
			Id:     &cust.ID,
			Email:  email,
			Name:   name,
			TraqId: traqID,
		}
	}

	dataItem := struct {
		AmountDue       *int64                `json:"amount_due,omitempty"`
		AmountPaid      *int64                `json:"amount_paid,omitempty"`
		AmountRemaining *int64                `json:"amount_remaining,omitempty"`
		Created         *int64                `json:"created,omitempty"`
		Customer        *api.Customer         `json:"customer,omitempty"`
		Id              *string               `json:"id,omitempty"`
		PaymentIntent   *string               `json:"payment_intent,omitempty"`
		ProductId       *string               `json:"product_id,omitempty"`
		Status          *api.InvoiceDataStatus `json:"status,omitempty"`
	}{
		AmountDue:       &amountDue,
		AmountPaid:      &amountPaid,
		AmountRemaining: &amountRemaining,
		Created:         &created,
		Customer:        apiCustomer,
		Id:              &id,
		PaymentIntent:   paymentIntent,
		ProductId:       productID,
		Status:          &status,
	}
	dataSlice := []interface{}{dataItem}
	dataJSON, err := json.Marshal(map[string]interface{}{"data": dataSlice, "has_more": false})
	if err != nil {
		return api.Invoice{}, err
	}
	var result api.Invoice
	if err := json.Unmarshal(dataJSON, &result); err != nil {
		return api.Invoice{}, err
	}
	return result, nil
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
