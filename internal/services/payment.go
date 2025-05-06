package service

import (
	"context"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

type PaymentService interface {
	CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error)
	GetPaymentByID(ctx context.Context, id string) (*models.Payment, error)
	ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error)
	ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error)
}

type paymentService struct {
	repo         repository.PaymentRepository
	stripeClient stripe.Client
}

func NewPaymentService(repo repository.PaymentRepository, stripeClient stripe.Client) PaymentService {
	return &paymentService{repo: repo, stripeClient: stripeClient}
}

// CreatePayment implements PaymentService.
func (s *paymentService) CreatePayment(ctx context.Context, req *models.PaymentRequest) (*models.PaymentResponse, error) {
	// new request for payment
	paymentIntent, err := s.stripeClient.CreatePaymentIntent(
		req.Amount, req.Currency, req.Description, req.CustomerID)
	if err != nil {
		return nil, errors.ThirdPartyError("Failed to create payment intent").WithError(err)
	}

	// create a payment method & attach it to paymentIntent
	if req.PaymentMethod == "card" {
		// paymentMethod, err := p.stripeClient.CreatePaymentMethod(req.CardNumber, fmt.Sprintf("%d", req.CardExpMonth), fmt.Sprintf("%d", req.CardExpYear), req.CardCVC)
		paymentMethod, err := s.stripeClient.CreatePaymentMethodFromToken(req.Token)
		if err != nil {
			return nil, errors.ThirdPartyError("Failed to create payment method").WithError(err)
		}

		err = s.stripeClient.AttachPaymentMethodToIntent(paymentMethod.ID, paymentIntent.ID)
		if err != nil {
			return nil, errors.ThirdPartyError("Failed to attach payment method").WithError(err)
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

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, errors.DatabaseError("Failed to record payment").WithError(err)
	}

	return &models.PaymentResponse{
		Payment:       payment,
		ClientSecret:  paymentIntent.ClientSecret,
		PaymentStatus: string(payment.Status),
		Message:       "Payment initiated successfully.",
	}, nil
}

// GetPaymentByID implements PaymentService.
func (s *paymentService) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {
	payment, err := s.repo.GetPaymentByID(ctx, id)
	if err != nil {
		return nil, errors.DatabaseError("Payment not found").WithError(err)
	}

	return payment, nil
}

// ListPaymentsByCustomer implements PaymentService.
func (s *paymentService) ListPaymentsByCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error) {
	payments, total, err := s.repo.ListPaymentsOfCustomer(ctx, customerID, page, size)
	if err != nil {
		return nil, 0, errors.DatabaseError("Failed to fetch payments").WithError(err)
	}

	return payments, total, nil
}

// ProcessWebhook implements PaymentService.
func (s *paymentService) ProcessWebhook(ctx context.Context, payload []byte, signature string) (stripe.Event, error) {
	event, err := s.stripeClient.VerifyWebhookSignature(payload, signature)
	if err != nil {
		return stripe.Event{}, errors.ThirdPartyError("Webhook signature verification failed").WithError(err)
	}

	switch event.Type {
	case "payment_intent.succeeded":
		paymentIntent := event.Data.Object

		stripeIDInterface, ok := paymentIntent["id"]
		if !ok {

			return event, errors.InternalError("Payment intent ID not found in Stripe response")
		}
		stripeID, ok := stripeIDInterface.(string)
		if !ok {
			return event, errors.InternalError("Payment intent ID is not a string in Stripe response")
		}

		if stripeID == "" {
			return event, errors.ThirdPartyError("Missing payment intent ID in webhook")
		}

		if err := s.repo.UpdatePaymentStatus(ctx, stripeID, models.PaymentStatusSucceeded); err != nil {
			return event, errors.DatabaseError("Failed to update payment status").WithError(err)
		}

	case "payment_intent.payment_failed":
		paymentIntent := event.Data.Object

		stripeIDInterface, ok := paymentIntent["id"]
		if !ok {
			return event, errors.InternalError("Payment intent ID not found in Stripe response")
		}

		stripeID, ok := stripeIDInterface.(string)
		if !ok {
			return event, errors.InternalError("Payment intent ID is not a string in Stripe response")
		}

		if stripeID == "" {
			return event, errors.ThirdPartyError("Missing payment intent ID in webhook")
		}

		if err := s.repo.UpdatePaymentStatus(ctx, stripeID, models.PaymentStatusFailed); err != nil {
			return event, errors.DatabaseError("Failed to update payment status").WithError(err)
		}

	case "charge.refunded":
		chargeObject := event.Data.Object
		paymentIntentID, piOK := chargeObject["payment_intent"].(string)

		if !piOK || paymentIntentID == "" {
			return event, errors.ThirdPartyError("Missing payment intent ID in webhook")
		}

		if err := s.repo.UpdatePaymentStatus(ctx, paymentIntentID, models.PaymentStatusRefunded); err != nil {
			return event, errors.DatabaseError("Failed to update payment status").WithError(err)
		}
	}

	return event, nil
}
