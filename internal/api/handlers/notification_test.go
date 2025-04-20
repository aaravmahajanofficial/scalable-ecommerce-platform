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
)

func TestSendEmail(t *testing.T) {
	// Arrange
	mockNotificationService := new(mocks.NotificationService)
	notificationHandler := handlers.NewNotificationHandler(mockNotificationService)
	testUserID := uuid.New()

	t.Run("Success - Send Email", func(t *testing.T) {
		// Arrange
		reqBody := models.EmailNotificationRequest{
			To:      "test@example.com",
			Subject: "Test Subject",
			Content: "Test Body",
		}

		// Mock Call
		expectedNotification := &models.NotificationResponse{
			ID:        uuid.New(),
			Recipient: testUserID.String(),
			Type:      models.NotificationTypeEmail,
			Status:    models.StatusPending,
			CreatedAt: time.Now(),
		}
		mockNotificationService.On("SendEmail", mock.Anything, &reqBody).Return(expectedNotification, nil).Once()

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/notifications/email", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.SendEmail()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Marshall the Data from map[string]interface{} to bytes
		databytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respNotification models.Notification
		err = json.Unmarshal(databytes, &respNotification)
		assert.NoError(t, err)
		assert.Equal(t, expectedNotification.ID, respNotification.ID)
		assert.Equal(t, expectedNotification.Recipient, respNotification.Recipient)
		assert.Equal(t, expectedNotification.Status, respNotification.Status)

		mockNotificationService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized (No Claims)", func(t *testing.T) {
		// Arrange
		reqBody := models.EmailNotificationRequest{
			To:      "test@example.com",
			Subject: "Test Subject",
			Content: "Test Body",
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/notifications/email", bytes.NewReader(reqBodyBytes), nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.SendEmail()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		mockNotificationService.AssertNotCalled(t, "SendEmail")
	})

	t.Run("Failure - Invalid Input (Bad JSON)", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/notifications/email", bytes.NewReader([]byte("{invalid json")), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.SendEmail()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockNotificationService.AssertNotCalled(t, "SendEmail")
	})

	t.Run("Failure - Invalid Input (Validation Error)", func(t *testing.T) {
		// Arrange
		reqBody := models.EmailNotificationRequest{
			To:      "test@example.com",
			Content: "Test Body",
		}

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/notifications/email", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.SendEmail()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockNotificationService.AssertNotCalled(t, "SendEmail")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		reqBody := models.EmailNotificationRequest{
			To:      "test@example.com",
			Subject: "Test Subject",
			Content: "Test Body",
		}

		// Mock Call
		mockNotificationService.On("SendEmail", mock.Anything, &reqBody).Return(nil, appErrors.InternalError("Failed to send email")).Once()

		// Create request body
		reqBodyBytes, _ := json.Marshal(reqBody)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/notifications/email", bytes.NewReader(reqBodyBytes), testUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.SendEmail()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		// Optionally check error response body matches serviceErr
		mockNotificationService.AssertExpectations(t)
	})
}

func TestListNotifications(t *testing.T) {
	// Arrange
	mockNotificationService := new(mocks.NotificationService)
	notificationHandler := handlers.NewNotificationHandler(mockNotificationService)
	testUserID := uuid.New()

	t.Run("Success - List Notifications with Pagination", func(t *testing.T) {
		// Arrange
		expectedNotifications := []*models.Notification{
			{ID: uuid.New(), Recipient: testUserID.String(), Type: models.NotificationTypeEmail, Subject: "Notification 1", Status: models.StatusSent},
			{ID: uuid.New(), Recipient: testUserID.String(), Type: models.NotificationTypeEmail, Subject: "Notification 2", Status: models.StatusFailed},
		}
		page := 2
		pageSize := 20

		expectedTotal := 15

		// Mock Call
		mockNotificationService.On("ListNotifications", mock.Anything, page, pageSize).Return(expectedNotifications, expectedTotal, nil).Once()

		target := fmt.Sprintf("/notifications?page=%d&pageSize=%d", page, pageSize)
		req := testutils.CreateTestRequestWithContext(http.MethodGet, target, nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.ListNotifications()
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
		notificationBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		// Unmarshal the order data
		var respNotifications []*models.Notification
		err = json.Unmarshal(notificationBytes, &respNotifications)
		assert.NoError(t, err)

		// Assert the order data
		assert.Len(t, respNotifications, len(expectedNotifications))
		assert.Equal(t, expectedNotifications[0].ID, respNotifications[0].ID)

		mockNotificationService.AssertExpectations(t)
	})

	t.Run("Success - Default Pagination", func(t *testing.T) {
		// Arrange
		expectedNotifications := []*models.Notification{
			{ID: uuid.New(), Recipient: testUserID.String(), Type: models.NotificationTypeEmail, Subject: "Notification A", Status: models.StatusSent},
		}
		expectedPage := 1
		expectedPageSize := 10
		expectedTotal := 5

		// Mock Call
		mockNotificationService.On("ListNotifications", mock.Anything, expectedPage, expectedPageSize).Return(expectedNotifications, expectedTotal, nil).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/notifications", nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.ListNotifications()
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
		notificationBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		var respNotifications []*models.Notification
		err = json.Unmarshal(notificationBytes, &respNotifications)
		assert.NoError(t, err)

		// Assert the order data
		assert.Len(t, respNotifications, len(expectedNotifications))
		assert.Equal(t, expectedNotifications[0].ID, respNotifications[0].ID)
		assert.Equal(t, expectedNotifications[0].Recipient, respNotifications[0].Recipient)

		mockNotificationService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized (No Claims)", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, "/notifications", nil, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.ListNotifications()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockNotificationService.AssertNotCalled(t, "ListNotifications")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		defaultPage := 1
		defaultPageSize := 10

		// Mock Call
		mockNotificationService.On("ListNotifications", mock.Anything, defaultPage, defaultPageSize).Return(nil, 0, appErrors.DatabaseError("DB Failed")).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/notifications", nil, testUserID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := notificationHandler.ListNotifications()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockNotificationService.AssertExpectations(t)
	})
}
