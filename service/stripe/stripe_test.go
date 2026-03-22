package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	api "github.com/traPtitech/Checkin-openapi/server"
	"go.uber.org/zap"
)

func TestHandleWebhookAcceptsSignedInvoicePaidEvent(t *testing.T) {
	secret := "whsec_test_secret"
	payload, err := json.Marshal(map[string]any{
		"id":          "evt_test_webhook",
		"object":      "event",
		"api_version": "2026-02-25.clover",
		"type":        "invoice.paid",
		"data": map[string]any{
			"object": map[string]any{
				"id":               "in_test_123",
				"object":           "invoice",
				"amount_due":       1000,
				"amount_paid":      1000,
				"amount_remaining": 0,
				"created":          1700000000,
				"status":           "paid",
				"lines": map[string]any{
					"object":   "list",
					"has_more": false,
					"url":      "/v1/invoices/in_test_123/lines",
					"data": []map[string]any{
						{
							"id":     "il_test_123",
							"object": "line_item",
							"price": map[string]any{
								"id":     "price_test_123",
								"object": "price",
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	signature := signedWebhookSignature(secret, payload, strconv.FormatInt(time.Now().Unix(), 10))
	svc := &StripeService{
		logger:        zap.NewNop(),
		webhookSecret: secret,
	}

	result, err := svc.HandleWebhook(context.Background(), payload, signature)
	require.NoError(t, err)
	require.NotNil(t, result.Data)
	require.Len(t, *result.Data, 1)
	require.NotNil(t, (*result.Data)[0].Id)
	require.Equal(t, "in_test_123", *(*result.Data)[0].Id)
	require.NotNil(t, (*result.Data)[0].Status)
	require.Equal(t, api.InvoiceDataStatus("paid"), *(*result.Data)[0].Status)
}

func signedWebhookSignature(secret string, payload []byte, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return fmt.Sprintf("t=%s,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
}
