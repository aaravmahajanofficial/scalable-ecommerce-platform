package service_test

import (
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repoMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	stripeMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
)

func TestNewPaymentService(t *testing.T) {
	mockRepo := repoMocks.NewMockPaymentRepository(t)
	mockStripeClient := stripeMocks.NewMockClient(t)
	service := service.NewPaymentService(mockRepo, mockStripeClient)
	assert.NotNil(t, service)
}

func TestCreatePayment(t *testing.T) {
	ctx := t.Context()

	testUserID := uuid.New().String()
	testPaymentIntentID := "pi_123"
	testPaymentMethodID := "pm_456"
	testClientSecret := "pi_123_secret_abc"

	reqCard := &models.PaymentRequest{
		CustomerID:    testUserID,
		Amount:        1000,
		Currency:      "usd",
		Description:   "Test Card Payment",
		PaymentMethod: "card",
		Token:         "tok_visa",
	}

	reqOther := &models.PaymentRequest{
		CustomerID:    testUserID,
		Amount:        2000,
		Currency:      "eur",
		Description:   "Test Other Payment",
		PaymentMethod: "ideal",
	}

	mockPaymentIntent := &stripe.PaymentIntent{
		ID:           testPaymentIntentID,
		Amount:       reqCard.Amount,
		Currency:     stripe.Currency(reqCard.Currency),
		Description:  reqCard.Description,
		ClientSecret: testClientSecret,
		Status:       stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockPaymentMethod := &stripe.PaymentMethod{
		ID:   testPaymentMethodID,
		Type: stripe.PaymentMethodTypeCard,
	}

	expectedPayment := &models.Payment{
		ID:            testPaymentIntentID,
		CustomerID:    reqCard.CustomerID,
		Amount:        reqCard.Amount,
		Currency:      reqCard.Currency,
		Description:   reqCard.Description,
		Status:        models.PaymentStatusPending,
		PaymentMethod: reqCard.PaymentMethod,
		StripeID:      testPaymentIntentID,
	}

	t.Run("Success - Card Payment", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.MatchedBy(func(p *models.Payment) bool {
			return p.ID == testPaymentIntentID && p.CustomerID == reqCard.CustomerID && p.Amount == reqCard.Amount
		})).Return(nil).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqCard)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, testClientSecret, resp.ClientSecret)
		assert.Equal(t, string(models.PaymentStatusPending), resp.PaymentStatus)
		assert.NotNil(t, resp.Payment)
		assert.Equal(t, expectedPayment.ID, resp.Payment.ID)
		assert.Equal(t, expectedPayment.CustomerID, resp.Payment.CustomerID)
		assert.Equal(t, expectedPayment.Amount, resp.Payment.Amount)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Success - Non-Card Payment", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		mockPaymentIntentOther := &stripe.PaymentIntent{
			ID:           "pi_789",
			Amount:       reqOther.Amount,
			Currency:     stripe.Currency(reqOther.Currency),
			Description:  reqOther.Description,
			ClientSecret: "pi_789_secret_def",
			Status:       stripe.PaymentIntentStatusRequiresPaymentMethod,
		}

		mockStripeClient.On("CreatePaymentIntent", reqOther.Amount, reqOther.Currency, reqOther.Description, reqOther.CustomerID).Return(mockPaymentIntentOther, nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.MatchedBy(func(p *models.Payment) bool {
			return p.ID == mockPaymentIntentOther.ID && p.CustomerID == reqOther.CustomerID && p.Amount == reqOther.Amount
		})).Return(nil).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqOther)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockPaymentIntentOther.ClientSecret, resp.ClientSecret)
		assert.Equal(t, string(models.PaymentStatusPending), resp.PaymentStatus)
		assert.NotNil(t, resp.Payment)
		assert.Equal(t, mockPaymentIntentOther.ID, resp.Payment.ID)

		// Assert
		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
		mockStripeClient.AssertNotCalled(t, "CreatePaymentMethodFromToken")
		mockStripeClient.AssertNotCalled(t, "AttachPaymentMethodToIntent")
	})

	t.Run("Failure - CreatePaymentIntent Fails", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		stripeErr := errors.New("stripe API error")
		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(nil, stripeErr).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqCard)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.ErrorIs(t, err, stripeErr) // Check underlying error

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - CreatePaymentMethodFromToken Fails", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		stripeErr := errors.New("stripe token error")

		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(nil, stripeErr).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqCard)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.ErrorIs(t, err, stripeErr)

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
		mockStripeClient.AssertNotCalled(t, "AttachPaymentMethodToIntent")
	})

	t.Run("Failure - AttachPaymentMethodToIntent Fails", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		stripeErr := errors.New("stripe attach error")

		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(stripeErr).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqCard)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.ErrorIs(t, err, stripeErr)

		mockRepo.AssertNotCalled(t, "CreatePayment")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - Repository CreatePayment Fails", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		dbErr := errors.New("database insert error")

		mockStripeClient.On("CreatePaymentIntent", reqCard.Amount, reqCard.Currency, reqCard.Description, reqCard.CustomerID).Return(mockPaymentIntent, nil).Once()
		mockStripeClient.On("CreatePaymentMethodFromToken", reqCard.Token).Return(mockPaymentMethod, nil).Once()
		mockStripeClient.On("AttachPaymentMethodToIntent", mockPaymentMethod.ID, mockPaymentIntent.ID).Return(nil).Once()
		mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).Return(dbErr).Once()

		// Act
		resp, err := paymentService.CreatePayment(ctx, reqCard)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})
}

func TestGetPaymentByID(t *testing.T) {
	ctx := t.Context()
	mockStripeClient := stripeMocks.NewMockClient(t)

	testPaymentID := uuid.New().String()
	expectedPayment := &models.Payment{
		ID:         testPaymentID,
		CustomerID: uuid.New().String(),
		Amount:     500,
		Currency:   "gbp",
		Status:     models.PaymentStatusSucceeded,
		CreatedAt:  time.Now().Add(-time.Hour),
		UpdatedAt:  time.Now(),
	}

	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		mockRepo.On("GetPaymentByID", ctx, testPaymentID).Return(expectedPayment, nil).Once()

		// Act
		payment, err := paymentService.GetPaymentByID(ctx, testPaymentID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, payment)
		assert.Equal(t, expectedPayment, payment)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		repoErr := errors.New("payment not found in DB")
		mockRepo.On("GetPaymentByID", ctx, testPaymentID).Return(nil, repoErr).Once()

		// Act
		payment, err := paymentService.GetPaymentByID(ctx, testPaymentID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, payment)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code) // Service wraps it
		assert.ErrorIs(t, err, repoErr)

		mockRepo.AssertExpectations(t)
	})
}

func TestListPaymentsByCustomer(t *testing.T) {
	ctx := t.Context()
	mockStripeClient := stripeMocks.NewMockClient(t)

	testCustomerID := uuid.New().String()
	page := 1
	size := 10
	expectedTotal := 5
	expectedPayments := []*models.Payment{
		{ID: uuid.New().String(), CustomerID: testCustomerID, Amount: 100},
		{ID: uuid.New().String(), CustomerID: testCustomerID, Amount: 200},
	}

	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		mockRepo.On("ListPaymentsOfCustomer", ctx, testCustomerID, page, size).Return(expectedPayments, expectedTotal, nil).Once()

		// Act
		payments, total, err := paymentService.ListPaymentsByCustomer(ctx, testCustomerID, page, size)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedPayments, payments)
		assert.Equal(t, expectedTotal, total)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Repository Error", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		repoErr := errors.New("failed to query payments")
		mockRepo.On("ListPaymentsOfCustomer", ctx, testCustomerID, page, size).Return(nil, 0, repoErr).Once()

		// Act
		payments, total, err := paymentService.ListPaymentsByCustomer(ctx, testCustomerID, page, size)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, payments)
		assert.Equal(t, 0, total)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code) // Service wraps it
		assert.ErrorIs(t, err, repoErr)

		mockRepo.AssertExpectations(t)
	})
}

func TestProcessWebhook(t *testing.T) {
	ctx := t.Context()

	payload := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded", "data": {"object": {"id": "pi_abc"}}}`)
	signature := "whsec_sig"
	stripePaymentIntentID := "pi_abc"

	eventSucceeded := stripe.Event{
		ID:   "evt_123",
		Type: "payment_intent.succeeded",
		Data: &stripe.EventData{
			Object: map[string]any{
				"id": stripePaymentIntentID,
			},
		},
	}
	eventFailed := stripe.Event{
		ID:   "evt_456",
		Type: "payment_intent.payment_failed",
		Data: &stripe.EventData{
			Object: map[string]interface{}{
				"id": stripePaymentIntentID,
			},
		},
	}
	eventRefunded := stripe.Event{
		ID:   "evt_789",
		Type: "charge.refunded",
		Data: &stripe.EventData{
			Object: map[string]any{
				"id":             "ch_xyz",
				"payment_intent": stripePaymentIntentID,
			},
		},
	}
	eventOther := stripe.Event{
		ID:   "evt_000",
		Type: "customer.created",
		Data: &stripe.EventData{
			Object: map[string]interface{}{"id": "cus_123"},
		},
	}
	eventMissingID := stripe.Event{
		ID:   "evt_bad",
		Type: "payment_intent.succeeded",
		Data: &stripe.EventData{
			Object: map[string]interface{}{
				"amount": 1000,
			},
		},
	}

	t.Run("Success - payment_intent.succeeded", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		mockStripeClient.On("VerifyWebhookSignature", payload, signature).Return(eventSucceeded, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusSucceeded).Return(nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payload, signature)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, eventSucceeded.ID, event.ID)
		assert.Equal(t, eventSucceeded.Type, event.Type)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Success - payment_intent.payment_failed", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		payloadFailed := []byte(`{"id": "evt_456", "type": "payment_intent.payment_failed", "data": {"object": {"id": "pi_abc"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadFailed, signature).Return(eventFailed, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusFailed).Return(nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadFailed, signature)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, eventFailed.ID, event.ID)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Success - charge.refunded", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		payloadRefunded := []byte(`{"id": "evt_789", "type": "charge.refunded", "data": {"object": {"id": "ch_xyz", "payment_intent": "pi_abc"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadRefunded, signature).Return(eventRefunded, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusRefunded).Return(nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadRefunded, signature)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, eventRefunded.ID, event.ID)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Success - Unhandled Event Type", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		payloadOther := []byte(`{"id": "evt_000", "type": "customer.created", "data": {"object": {"id": "cus_123"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadOther, signature).Return(eventOther, nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadOther, signature)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, eventOther.ID, event.ID)

		mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - VerifyWebhookSignature Fails", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		verifyErr := errors.New("invalid signature")
		mockStripeClient.On("VerifyWebhookSignature", payload, signature).Return(stripe.Event{}, verifyErr).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payload, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, stripe.Event{}, event)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.ErrorIs(t, err, verifyErr)

		mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - Missing Payment Intent ID (Succeeded)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		payloadMissingID := []byte(`{"id": "evt_bad", "type": "payment_intent.succeeded", "data": {"object": {"amount": 1000}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadMissingID, signature).Return(eventMissingID, nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadMissingID, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventMissingID.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeInternal, appErr.Code)
		assert.Contains(t, err.Error(), "Payment intent ID not found")

		mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - UpdatePaymentStatus Fails (Succeeded)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		dbErr := errors.New("db update failed")

		mockStripeClient.On("VerifyWebhookSignature", payload, signature).Return(eventSucceeded, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusSucceeded).Return(dbErr).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payload, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventSucceeded.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - Missing Payment Intent ID (Failed)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		eventMissingIDFailed := stripe.Event{
			ID:   "evt_bad_fail",
			Type: "payment_intent.payment_failed",
			Data: &stripe.EventData{Object: map[string]interface{}{"reason": "card_declined"}},
		}
		payloadMissingIDFailed := []byte(`{"id": "evt_bad_fail", "type": "payment_intent.payment_failed", "data": {"object": {"reason": "card_declined"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadMissingIDFailed, signature).Return(eventMissingIDFailed, nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadMissingIDFailed, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventMissingIDFailed.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeInternal, appErr.Code)
		assert.Contains(t, err.Error(), "Payment intent ID not found")

		mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - UpdatePaymentStatus Fails (Failed)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		dbErr := errors.New("db update failed")
		payloadFailed := []byte(`{"id": "evt_456", "type": "payment_intent.payment_failed", "data": {"object": {"id": "pi_abc"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadFailed, signature).Return(eventFailed, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusFailed).Return(dbErr).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadFailed, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventFailed.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - Missing Payment Intent ID (Refunded)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		eventMissingIDRefunded := stripe.Event{
			ID:   "evt_bad_refund",
			Type: "charge.refunded",
			Data: &stripe.EventData{Object: map[string]any{"id": "ch_xyz"}},
		}
		payloadMissingIDRefunded := []byte(`{"id": "evt_bad_refund", "type": "charge.refunded", "data": {"object": {"id": "ch_xyz"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadMissingIDRefunded, signature).Return(eventMissingIDRefunded, nil).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadMissingIDRefunded, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventMissingIDRefunded.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeThirdPartyError, appErr.Code)
		assert.Contains(t, err.Error(), "Missing payment intent ID")

		mockRepo.AssertNotCalled(t, "UpdatePaymentStatus")
		mockStripeClient.AssertExpectations(t)
	})

	t.Run("Failure - UpdatePaymentStatus Fails (Refunded)", func(t *testing.T) {
		// Arrange
		mockRepo := repoMocks.NewMockPaymentRepository(t)
		mockStripeClient := stripeMocks.NewMockClient(t)
		paymentService := service.NewPaymentService(mockRepo, mockStripeClient)

		dbErr := errors.New("db update failed")
		payloadRefunded := []byte(`{"id": "evt_789", "type": "charge.refunded", "data": {"object": {"id": "ch_xyz", "payment_intent": "pi_abc"}}}`)
		mockStripeClient.On("VerifyWebhookSignature", payloadRefunded, signature).Return(eventRefunded, nil).Once()
		mockRepo.On("UpdatePaymentStatus", ctx, stripePaymentIntentID, models.PaymentStatusRefunded).Return(dbErr).Once()

		// Act
		event, err := paymentService.ProcessWebhook(ctx, payloadRefunded, signature)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, eventRefunded.ID, event.ID)

		appErr, ok := appErrors.IsAppError(err)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.ErrorIs(t, err, dbErr)

		mockRepo.AssertExpectations(t)
		mockStripeClient.AssertExpectations(t)
	})
}
