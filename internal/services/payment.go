package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error)
	GetPaymentByID(ctx context.Context, id string) (*models.Payment, error)
	ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error)
	ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error)
}

type paymentService struct {
	repo         *repository.PaymentRepository
	stripeClient stripe.Client
}

func NewPaymentService(repo *repository.PaymentRepository, stripeClient stripe.Client) PaymentService {
	return &paymentService{repo: repo, stripeClient: stripeClient}
}

// CreatePayment implements PaymentService.
func (p *paymentService) CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error) {

	// new request for payment
	paymentIntent, err := p.stripeClient.CreatePaymentIntent(
		req.Amount, req.Currency, req.Description, req.CustomerID)

	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	// create a payment method & attach it to paymentIntent
	if req.PaymentMethod == "card" {
		// paymentMethod, err := p.stripeClient.CreatePaymentMethod(req.CardNumber, fmt.Sprintf("%d", req.CardExpMonth), fmt.Sprintf("%d", req.CardExpYear), req.CardCVC)
		paymentMethod, err := p.stripeClient.CreatePaymentMethodFromToken(req.Token)

		if err != nil {
			return nil, fmt.Errorf("failed to create payment method: %w", err)
		}

		err = p.stripeClient.AttachPaymentMethodToIntent(paymentMethod.ID, paymentIntent.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to create attach payment method: %w", err)
		}

	}

	// store the payment in the database
	payment := &models.Payment{
		ID:            paymentIntent.ID,
		CustomerID:    req.CustomerID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Description:   req.Description,
		Status:        models.PaymentStatusPending,
		PaymentMethod: req.PaymentMethod,
		StripeID:      paymentIntent.ID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := p.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to store the payment: %w", err)
	}

	return &models.PaymentResponse{
		Payment:       payment,
		ClientSecret:  paymentIntent.ClientSecret,
		PaymentStatus: string(payment.Status),
		Message:       "Payment initiated successfully.",
	}, nil

}

// GetPaymentByID implements PaymentService.
func (p *paymentService) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {

	return p.repo.GetPaymentByID(ctx, id)

}

// ListPaymentsByCustomer implements PaymentService.
func (p *paymentService) ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error) {

	return p.repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

}

// ProcessWebhook implements PaymentService.
func (p *paymentService) ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error) {

	event, err := p.stripeClient.VerifyWebhookSignature(payload, signature)

	if err != nil {
		return stripe.Event{}, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	switch event.Type {

	case "payment_intent.succeeded":
		paymentIntent := event.Data.Object
		stripeID, _ := paymentIntent["id"].(string)

		if stripeID == "" {
			return event, fmt.Errorf("missing payment intent ID in webhook")
		}

		if err := p.repo.UpdatePaymentStatus(ctx, stripeID, models.PaymentStatusSucceeded); err != nil {
			return event, fmt.Errorf("failed to update payment status: %w", err)
		}

	case "payment_intent.payment_failed":
		paymentIntent := event.Data.Object
		stripeID, _ := paymentIntent["id"].(string)

		if stripeID == "" {
			return event, fmt.Errorf("missing payment intent ID in webhook")
		}

		if err := p.repo.UpdatePaymentStatus(ctx, stripeID, models.PaymentStatusFailed); err != nil {
			return event, fmt.Errorf("failed to update payment status: %w", err)
		}

	case "charge.refunded":
		paymentIntent := event.Data.Object
		stripeID, _ := paymentIntent["id"].(string)

		if stripeID == "" {
			return event, fmt.Errorf("missing payment intent ID in webhook")
		}

		if err := p.repo.UpdatePaymentStatus(ctx, stripeID, models.PaymentStatusRefunded); err != nil {
			return event, fmt.Errorf("failed to update payment status: %w", err)
		}
	}
	return stripe.Event{}, nil
}
