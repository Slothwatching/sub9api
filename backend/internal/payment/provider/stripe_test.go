//go:build unit

package provider

import (
	"bytes"
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
	stripe "github.com/stripe/stripe-go/v85"
)

type stripeRefundBackend struct {
	params []*stripe.RefundCreateParams
}

func (b *stripeRefundBackend) Call(_ string, _ string, _ string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	b.params = append(b.params, params.(*stripe.RefundCreateParams))
	refund := v.(*stripe.Refund)
	refund.ID = "re_123"
	refund.Status = stripe.RefundStatusSucceeded
	return nil
}

func (*stripeRefundBackend) CallStreaming(string, string, string, stripe.ParamsContainer, stripe.StreamingLastResponseSetter) error {
	return nil
}

func (*stripeRefundBackend) CallRaw(string, string, string, []byte, *stripe.Params, stripe.LastResponseSetter) error {
	return nil
}

func (*stripeRefundBackend) CallMultipart(string, string, string, string, *bytes.Buffer, *stripe.Params, stripe.LastResponseSetter) error {
	return nil
}

func (*stripeRefundBackend) SetMaxNetworkRetries(int64) {}

func TestStripeRefundUsesStableAmountSpecificIdempotencyKey(t *testing.T) {
	backend := &stripeRefundBackend{}
	client := stripe.NewClient("sk_test", stripe.WithBackends(&stripe.Backends{API: backend}))
	provider := &Stripe{
		config:      map[string]string{"currency": "CNY"},
		initialized: true,
		sc:          client,
	}

	refund := func(amount string) {
		_, err := provider.Refund(context.Background(), payment.RefundRequest{
			TradeNo: "pi_123",
			OrderID: "sub2_order_456",
			Amount:  amount,
		})
		require.NoError(t, err)
	}

	refund("12.34")
	refund("12.34")
	refund("12.35")

	require.Len(t, backend.params, 3)
	require.Equal(t, int64(1234), *backend.params[0].Amount)
	require.Equal(t, "re-sub2_order_456-1234", *backend.params[0].IdempotencyKey)
	require.Equal(t, backend.params[0].IdempotencyKey, backend.params[1].IdempotencyKey)
	require.Equal(t, int64(1235), *backend.params[2].Amount)
	require.Equal(t, "re-sub2_order_456-1235", *backend.params[2].IdempotencyKey)
	require.NotEqual(t, *backend.params[0].IdempotencyKey, *backend.params[2].IdempotencyKey)
}

type stripePaymentIntentBackend struct {
	stripeRefundBackend
	params []*stripe.PaymentIntentCreateParams
}

func (b *stripePaymentIntentBackend) Call(_ string, _ string, _ string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	b.params = append(b.params, params.(*stripe.PaymentIntentCreateParams))
	pi := v.(*stripe.PaymentIntent)
	pi.ID = "pi_123"
	pi.ClientSecret = "pi_123_secret"
	return nil
}

func TestStripeCreatePaymentSetsReceiptEmailOnlyWhenPayerEmailKnown(t *testing.T) {
	backend := &stripePaymentIntentBackend{}
	client := stripe.NewClient("sk_test", stripe.WithBackends(&stripe.Backends{API: backend}))
	provider := &Stripe{
		config:      map[string]string{"currency": "CNY"},
		initialized: true,
		sc:          client,
	}

	create := func(orderID, email string) {
		_, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
			OrderID:            orderID,
			Amount:             "10.00",
			Subject:            "Sloth Code 10.00",
			InstanceSubMethods: "card,link",
			PayerEmail:         email,
		})
		require.NoError(t, err)
	}

	create("sub2_with_email", "  payer@example.com ")
	create("sub2_without_email", "")

	require.Len(t, backend.params, 2)
	require.NotNil(t, backend.params[0].ReceiptEmail)
	require.Equal(t, "payer@example.com", *backend.params[0].ReceiptEmail)
	require.Nil(t, backend.params[1].ReceiptEmail)
}
