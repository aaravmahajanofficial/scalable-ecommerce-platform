package stripe

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/paymentintent"
	"github.com/stripe/stripe-go/v81/paymentmethod"
	"github.com/stripe/stripe-go/v81/refund"
	"github.com/stripe/stripe-go/v81/webhook"
)

type Event = stripe.Event

// defines the methods that any of payment client must implement.
type Client interface {
	CreatePaymentIntent(amount int64, currency string, description string, customerID string) (*stripe.PaymentIntent, error)
	CreatePaymentMethod(cardNumber, cardExpMonth, cardExpYear, cardCVC string) (*stripe.PaymentMethod, error)
	CreatePaymentMethodFromToken(paymentMethodID string) (*stripe.PaymentMethod, error)
	AttachPaymentMethodToIntent(paymentMethodID, paymentIntentID string) error
	ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error)
	RefundPayment(paymentIntentID string, amount int64) (*stripe.Refund, error)
	VerifyWebhookSignature(payload []byte, signature string) (Event, error)
}

// stripeClient is the implementation of the Client interface.
type stripeClient struct {
	webhookSecret string
}

// type paypalClient struct {}

func NewStripeClient(apiKey string, webhookSecret string) Client {
	stripe.Key = apiKey

	// since *stripeClient is impplementing Client, it will automatically get converted to the Client interface
	return &stripeClient{webhookSecret: webhookSecret}
}

// PaymentIntent == "planned payment" or order waiting for payment.
func (s *stripeClient) CreatePaymentIntent(amount int64, currency string, description string, customerID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:      stripe.Int64(amount),
		Currency:    stripe.String(currency),
		Description: stripe.String(description),
	}

	if customerID != "" {
		params.Customer = stripe.String(customerID)
	}

	return paymentintent.New(params)
}

// CreatePaymentMethod implements Client.
func (s *stripeClient) CreatePaymentMethod(cardNumber string, cardExpMonth string, cardExpYear string, cardCVC string) (*stripe.PaymentMethod, error) {
	expMonth, err := strconv.ParseInt(cardExpMonth, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid card expiration month: %w", err)
	}

	expYear, err := strconv.ParseInt(cardExpYear, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid card expiration year: %w", err)
	}

	params := &stripe.PaymentMethodParams{
		Type: stripe.String("card"),
		Card: &stripe.PaymentMethodCardParams{
			Number:   stripe.String(cardNumber),
			ExpMonth: stripe.Int64(expMonth),
			ExpYear:  stripe.Int64(expYear),
			CVC:      stripe.String(cardCVC),
		},
	}

	return paymentmethod.New(params)
}

// CreatePaymentMethod implements Client.
func (s *stripeClient) CreatePaymentMethodFromToken(paymentMethodID string) (*stripe.PaymentMethod, error) {
	return paymentmethod.Get(paymentMethodID, nil)
}

// AttachPaymentMethodToIntent implements Client.
func (s *stripeClient) AttachPaymentMethodToIntent(paymentMethodID string, paymentIntentID string) error {
	params := &stripe.PaymentIntentParams{
		PaymentMethod: stripe.String(paymentMethodID),
	}

	_, err := paymentintent.Update(paymentIntentID, params)

	return err
}

// ConfirmPaymentIntent implements Client.
func (s *stripeClient) ConfirmPaymentIntent(paymentIntentID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentConfirmParams{
		PaymentMethod: stripe.String(paymentIntentID),
	}

	return paymentintent.Confirm(paymentIntentID, params)
}

// RefundPayment implements Client.
func (s *stripeClient) RefundPayment(paymentIntentID string, amount int64) (*stripe.Refund, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Amount:        stripe.Int64(amount),
	}

	return refund.New(params)
}

// VerifyWebhookSignature implements Client.
func (s *stripeClient) VerifyWebhookSignature(payload []byte, signature string) (Event, error) {
	if s.webhookSecret == "" {
		return Event{}, errors.New("webhook secret not configured")
	}

	return webhook.ConstructEvent(payload, signature, s.webhookSecret)
}

// 1️⃣ Create a Payment Intent
// → "I want to charge $100 for order #123"
// 2️⃣ Create a Payment Method
// → "This is the customer's Visa card."
// 3️⃣ Attach Payment Method to Intent
// → "Use this Visa card for order #123."
// 4️⃣ Confirm Payment Intent
// → "Charge the card now!"
