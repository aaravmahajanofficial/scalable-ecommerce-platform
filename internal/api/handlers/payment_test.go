package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services/mocks"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/testutils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
)

func TestCreatePayment(t *testing.T) {
	mockPaymentService := new(mocks.PaymentService)
	paymentHandler := handlers.NewPaymentHandler(mockPaymentService)
	testUserID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		// Arrange
		reqBody := models.PaymentRequest{
			CustomerID:    testUserID.String(),
			Amount:        1000,
			Currency:      "usd",
			Description:   "Test Payment",
			PaymentMethod: "card",
			Token:         "Test_Payment123",
		}
		expectedResp := &models.PaymentResponse{
			ClientSecret: "pi_123_secret_456",
			Payment: &models.Payment{
				ID:         uuid.New().String(),
				Amount:     reqBody.Amount,
				Currency:   reqBody.Currency,
				CustomerID: reqBody.CustomerID,
				Status:     "pending",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}

		mockPaymentService.On("CreatePayment", mock.Anything, mock.Anything).Return(expectedResp, nil).Once()

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/payments", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		// Marshal the 'data' field within the map back to bytes
		paymentBytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respPayment *models.PaymentResponse
		err = json.Unmarshal(paymentBytes, &respPayment)
		assert.NoError(t, err)

		assert.Equal(t, expectedResp.ClientSecret, respPayment.ClientSecret)
		assert.Equal(t, expectedResp.Payment.CustomerID, respPayment.Payment.CustomerID)

		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		reqBody := models.PaymentRequest{
			CustomerID:    testUserID.String(),
			Amount:        1000,
			Currency:      "usd",
			Description:   "Test Payment",
			PaymentMethod: "card",
			Token:         "Test_Payment123",
		}

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/payments", bytes.NewReader(reqBodyBytes), nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeUnauthorized)
		mockPaymentService.AssertNotCalled(t, "CreatePayment")
	})

	t.Run("Failure - Invalid Request Body", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/payments", bytes.NewReader([]byte("{invalid json")), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeBadRequest)
		mockPaymentService.AssertNotCalled(t, "CreatePayment")
	})

	t.Run("Failure - Validation Error", func(t *testing.T) {
		// Arrange
		reqBody := models.PaymentRequest{
			CustomerID:    testUserID.String(),
			Amount:        0,
			Currency:      "usd",
			Description:   "Test Payment",
			PaymentMethod: "card",
			Token:         "Test_Payment123",
		}

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/payments", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeValidation)
		mockPaymentService.AssertNotCalled(t, "CreatePayment")
	})

	t.Run("Failure - Forbidden", func(t *testing.T) {
		// Arrange
		differentUserID := uuid.New()

		reqBody := models.PaymentRequest{
			CustomerID:    differentUserID.String(),
			Amount:        1000,
			Currency:      "usd",
			Description:   "Test Payment",
			PaymentMethod: "card",
			Token:         "Test_Payment123",
		}

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/payments", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeForbidden)
		mockPaymentService.AssertNotCalled(t, "CreatePayment")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		reqBody := models.PaymentRequest{
			CustomerID:    testUserID.String(),
			Amount:        1000,
			Currency:      "usd",
			Description:   "Test Payment",
			PaymentMethod: "card",
			Token:         "Test_Payment123",
		}

		mockPaymentService.On("CreatePayment", mock.Anything, mock.AnythingOfType("*models.PaymentRequest")).Return(nil, appErrors.InternalError("payment provider down")).Once()

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/payments", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.CreatePayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockPaymentService.AssertExpectations(t)
	})
}

func TestGetPayment(t *testing.T) {
	mockPaymentService := new(mocks.PaymentService)
	paymentHandler := handlers.NewPaymentHandler(mockPaymentService)
	testUserID := uuid.New()
	paymentID := uuid.New().String()

	t.Run("Success", func(t *testing.T) {
		// Arrange
		expectedPayment := &models.Payment{
			ID:         paymentID,
			Amount:     1500,
			Currency:   "eur",
			CustomerID: testUserID.String(),
			Status:     "succeeded",
			CreatedAt:  time.Now().Add(-time.Hour),
			UpdatedAt:  time.Now(),
		}

		mockPaymentService.On("GetPaymentByID", mock.Anything, paymentID).Return(expectedPayment, nil).Once()

		pathParams := map[string]string{
			"id": paymentID,
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/payments/%s", paymentID), nil, testUserID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.GetPayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		// Marshal the 'data' field within the map back to bytes
		paymentBytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respPayment models.Payment
		err = json.Unmarshal(paymentBytes, &respPayment)
		assert.NoError(t, err)

		assert.Equal(t, expectedPayment.ID, respPayment.ID)
		assert.Equal(t, expectedPayment.CustomerID, respPayment.CustomerID)

		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, fmt.Sprintf("/payments/%s", paymentID), nil, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.GetPayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockPaymentService.AssertNotCalled(t, "GetPaymentByID")
	})

	t.Run("Failure - Missing Payment ID", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/payments/", nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.GetPayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeBadRequest)
		mockPaymentService.AssertNotCalled(t, "GetPaymentByID")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		mockPaymentService.On("GetPaymentByID", mock.Anything, paymentID).Return(nil, appErrors.NotFoundError("payment not found")).Once()

		pathParams := map[string]string{
			"id": paymentID,
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/payments/%s", paymentID), nil, testUserID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.GetPayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeNotFound)
		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Failure - Service Error (Internal)", func(t *testing.T) {
		// Arrange
		mockPaymentService.On("GetPaymentByID", mock.Anything, paymentID).Return(nil, appErrors.DatabaseError("database error")).Once()

		pathParams := map[string]string{
			"id": paymentID,
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/payments/%s", paymentID), nil, testUserID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.GetPayment()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockPaymentService.AssertExpectations(t)
	})
}

func TestListPayments(t *testing.T) {
	mockPaymentService := new(mocks.PaymentService)
	paymentHandler := handlers.NewPaymentHandler(mockPaymentService)
	testUserID := uuid.New()

	t.Run("Success - Default Pagination", func(t *testing.T) {
		// Arrange
		expectedPayments := []*models.Payment{
			{ID: uuid.New().String(), CustomerID: testUserID.String(), Amount: 100, Status: "succeeded"},
			{ID: uuid.New().String(), CustomerID: testUserID.String(), Amount: 200, Status: "pending"},
		}
		expectedPage := 1
		expectedPageSize := 10
		expectedTotal := 5

		// Mock Call
		mockPaymentService.On("ListPaymentsByCustomer", mock.Anything, testUserID.String(), expectedPage, expectedPageSize).Return(expectedPayments, expectedTotal, nil).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/payments", nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.ListPayments()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		dataMap, ok := resp.Data.(map[string]any)
		assert.True(t, ok, "resp.Data should be a map[string]any")

		assert.EqualValues(t, expectedPage, dataMap["page"])
		assert.EqualValues(t, expectedPageSize, dataMap["pageSize"])
		assert.EqualValues(t, expectedTotal, dataMap["total"])

		// Marshal the 'data' field within the map back to bytes
		paymentBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		var respPayments []models.Payment
		err = json.Unmarshal(paymentBytes, &respPayments)
		assert.NoError(t, err)

		assert.Len(t, respPayments, len(expectedPayments))
		assert.Equal(t, expectedPayments[0].ID, respPayments[0].ID)
		assert.Equal(t, expectedPayments[1].Amount, respPayments[1].Amount)

		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Success - Custom Pagination", func(t *testing.T) {
		// Arrange
		expectedPayments := []*models.Payment{
			{ID: uuid.New().String(), CustomerID: testUserID.String(), Amount: 300, Status: "succeeded"},
		}
		expectedTotal := 15
		page := 2
		pageSize := 5

		mockPaymentService.On("ListPaymentsByCustomer", mock.Anything, testUserID.String(), page, pageSize).Return(expectedPayments, expectedTotal, nil).Once()

		target := fmt.Sprintf("/payments?page=%d&pageSize=%d", page, pageSize)
		req := testutils.CreateTestRequestWithContext(http.MethodGet, target, nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.ListPayments()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal the base API response
		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		dataMap, ok := resp.Data.(map[string]any)
		assert.True(t, ok, "resp.Data should be a map[string]any")
		assert.EqualValues(t, page, dataMap["page"])
		assert.EqualValues(t, pageSize, dataMap["pageSize"])
		assert.EqualValues(t, expectedTotal, dataMap["total"])

		// Marshal the 'data' field within the map back to bytes
		paymentBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		var respPayments []*models.Payment
		err = json.Unmarshal(paymentBytes, &respPayments)
		assert.NoError(t, err)

		assert.Len(t, respPayments, len(expectedPayments))
		assert.Equal(t, expectedPayments[0].ID, respPayments[0].ID)

		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Success - Invalid Pagination Defaults", func(t *testing.T) {
		// Arrange
		testCases := []struct {
			name       string
			query      string
			expectPage int
			expectSize int
		}{
			{"Invalid page", "/payments?page=abc&pageSize=5", 1, 5},
			{"Page < 1", "/payments?page=0&pageSize=5", 1, 5},
			{"Invalid pageSize", "/payments?page=2&pageSize=xyz", 2, 10},
			{"PageSize < 1", "/payments?page=2&pageSize=0", 2, 10},
			{"PageSize > 100", "/payments?page=2&pageSize=101", 2, 10},
			{"Both invalid", "/payments?page=-1&pageSize=abc", 1, 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Arrange
				expectedPayments := []*models.Payment{}
				expectedTotal := 0

				mockPaymentService.On("ListPaymentsByCustomer", mock.Anything, testUserID.String(), tc.expectPage, tc.expectSize).
					Return(expectedPayments, expectedTotal, nil).Once()

				req := testutils.CreateTestRequestWithContext(http.MethodGet, tc.query, nil, testUserID, nil)
				rr := httptest.NewRecorder()

				// Act
				handler := paymentHandler.ListPayments()
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusOK, rr.Code)

				// Unmarshal the base API response
				var resp *response.APIResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.True(t, resp.Success)
				assert.NotEmpty(t, resp.Data)

				dataMap, ok := resp.Data.(map[string]any)
				assert.True(t, ok, "resp.Data should be a map[string]any")
				assert.EqualValues(t, tc.expectPage, dataMap["page"])
				assert.EqualValues(t, tc.expectSize, dataMap["pageSize"])
				assert.EqualValues(t, expectedTotal, dataMap["total"])

				mockPaymentService.AssertExpectations(t)
			})
		}
	})

	t.Run("Failure - Unauthorized (No Claims)", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, "/payments", nil, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.ListPayments()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeUnauthorized)
		mockPaymentService.AssertNotCalled(t, "ListPaymentsByCustomer")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		defaultPage := 1
		defaultPageSize := 10
		mockPaymentService.On("ListPaymentsByCustomer", mock.Anything, testUserID.String(), defaultPage, defaultPageSize).Return(nil, 0, appErrors.DatabaseError("database error")).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/payments", nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.ListPayments()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockPaymentService.AssertExpectations(t)
	})
}

func TestHandleStripeWebhook(t *testing.T) {
	mockPaymentService := new(mocks.PaymentService)
	paymentHandler := handlers.NewPaymentHandler(mockPaymentService)

	t.Run("Success", func(t *testing.T) {
		// Arrange
		payload := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded"}`)
		signature := "t=123,v1=abc,v0=def"
		expectedEvent := stripe.Event{
			ID:   "evt_123",
			Type: "payment_intent.succeeded",
		}

		mockPaymentService.On("ProcessWebhook", mock.Anything, payload, signature).Return(expectedEvent, nil).Once()

		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/payments/webhook", bytes.NewReader(payload), nil)
		req.Header.Set("Stripe-Signature", signature)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.HandleStripeWebhook()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Failure - Missing Signature", func(t *testing.T) {
		// Arrange
		payload := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded"}`)
		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/payments/webhook", bytes.NewReader(payload), nil)
		// No Stripe-Signature header
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.HandleStripeWebhook()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeBadRequest)
		mockPaymentService.AssertNotCalled(t, "ProcessWebhook")
	})

	t.Run("Failure - Service Error (Signature Verification)", func(t *testing.T) {
		// Arrange
		payload := []byte(`{"id": "evt_123", "type": "payment_intent.succeeded"}`)
		signature := "t=123,v1=invalid,v0=def"

		mockPaymentService.On("ProcessWebhook", mock.Anything, payload, signature).Return(stripe.Event{}, appErrors.UnauthorizedError("invalid webhook signature")).Once()

		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/payments/webhook", bytes.NewReader(payload), nil)
		req.Header.Set("Stripe-Signature", signature)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.HandleStripeWebhook()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeUnauthorized)
		mockPaymentService.AssertExpectations(t)
	})

	t.Run("Failure - Service Error (Processing)", func(t *testing.T) {
		// Arrange
		payload := []byte(`{"id": "evt_123", "type": "payment_intent.failed"}`)
		signature := "t=123,v1=abc,v0=def"
		expectedEvent := stripe.Event{
			ID:   "evt_123",
			Type: "payment_intent.failed",
		}

		mockPaymentService.On("ProcessWebhook", mock.Anything, payload, signature).Return(expectedEvent, appErrors.InternalError("failed to update order status")).Once()

		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/payments/webhook", bytes.NewReader(payload), nil)
		req.Header.Set("Stripe-Signature", signature)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := paymentHandler.HandleStripeWebhook()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockPaymentService.AssertExpectations(t)
	})
}
