package service_test

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"testing"
// 	"time"

// 	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
// 	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
// 	"github.com/google/uuid"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func setupNotificationServiceTest(t *testing.T) (service.NotificationService, *mocks.NotificationRepository, *mocks.UserRepository) {
// 	mockNotificationRepository := mocks.NewNotificationRepository(t)
// 	mockUserRepository := mocks.NewUserRepository(t)
// 	notificationService := service.NewNotificationService(mockNotificationRepository, mockUserRepository)
// 	return notificationService, mockNotificationRepository, mockUserRepository
// }

// func TestNotificationService_SendEmail(t *testing.T) {
// 	mockRepo, mockUserRepo, mockEmailService := setupNotificationServiceTest(t)
// 	ctx := context.Background()

// 	testUser := &models.User{ID: uuid.New(), Email: "test@example.com"}
// 	testReq := &models.EmailNotificationRequest{
// 		To:       "test@example.com",
// 		Subject:  "Test Subject",
// 		Content:  "Test Content",
// 		Metadata: map[string]string{"key": "value"},
// 	}
// 	metadataBytes, _ := json.Marshal(testReq.Metadata)

// 	t.Run("Success", func(t *testing.T) {
// 		mockUserRepo.On("GetUserByEmail", ctx, testReq.To).Return(testUser, nil).Once()
// 		mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once().Run(func(args mock.Arguments) {
// 			// Check if notification is created correctly before sending
// 			notification := args.Get(1).(*models.Notification)
// 			assert.Equal(t, models.NotificationTypeEmail, notification.Type)
// 			assert.Equal(t, testReq.To, notification.Recipient)
// 			assert.Equal(t, testReq.Subject, notification.Subject)
// 			assert.Equal(t, testReq.Content, notification.Content)
// 			assert.Equal(t, models.StatusPending, notification.Status)
// 			assert.Equal(t, json.RawMessage(metadataBytes), notification.Metadata)
// 		})
// 		mockEmailService.On("Send", ctx, testReq).Return(nil).Once()
// 		mockRepo.On("UpdateNotificationStatus", ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(nil).Once()

// 		resp, err := svc.SendEmail(ctx, testReq)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, resp)
// 		assert.Equal(t, models.NotificationTypeEmail, resp.Type)
// 		assert.Equal(t, models.StatusSent, resp.Status)
// 		assert.Equal(t, testReq.To, resp.Recipient)
// 		assert.NotNil(t, resp.ID)
// 		assert.NotZero(t, resp.CreatedAt)

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertExpectations(t)
// 		mockEmailService.AssertExpectations(t)
// 	})

// 	t.Run("Success - No Metadata", func(t *testing.T) {
// 		reqNoMeta := &models.EmailNotificationRequest{
// 			To:      "test@example.com",
// 			Subject: "Test Subject",
// 			Content: "Test Content",
// 		}
// 		mockUserRepo.On("GetUserByEmail", ctx, reqNoMeta.To).Return(testUser, nil).Once()
// 		mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once().Run(func(args mock.Arguments) {
// 			notification := args.Get(1).(*models.Notification)
// 			assert.Nil(t, notification.Metadata) // Ensure metadata is nil
// 		})
// 		mockEmailService.On("Send", ctx, reqNoMeta).Return(nil).Once()
// 		mockRepo.On("UpdateNotificationStatus", ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(nil).Once()

// 		resp, err := svc.SendEmail(ctx, reqNoMeta)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, resp)
// 		assert.Equal(t, models.StatusSent, resp.Status)

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertExpectations(t)
// 		mockEmailService.AssertExpectations(t)
// 	})

// 	t.Run("Error - User Not Found", func(t *testing.T) {
// 		notFoundErr := errors.New("user not found")
// 		mockUserRepo.On("GetUserByEmail", ctx, testReq.To).Return(nil, notFoundErr).Once()

// 		resp, err := svc.SendEmail(ctx, testReq)

// 		assert.Error(t, err)
// 		assert.Nil(t, resp)
// 		targetErr := customErrors.NotFoundError("User not found")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), notFoundErr)

// 		mockUserRepo.AssertExpectations(t)
// 		// Ensure other mocks were not called
// 		mockRepo.AssertNotCalled(t, "CreateNotification", mock.Anything, mock.Anything)
// 		mockEmailService.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
// 		mockRepo.AssertNotCalled(t, "UpdateNotificationStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
// 	})

// 	// Note: Testing json.Marshal failure is tricky without specific invalid data types.
// 	// This case assumes Metadata could potentially cause a marshal error.
// 	t.Run("Error - Metadata Marshal Failure", func(t *testing.T) {
// 		// Use a channel, which cannot be marshaled to JSON, to force an error
// 		reqInvalidMeta := &models.EmailNotificationRequest{
// 			To:       "test@example.com",
// 			Subject:  "Test Subject",
// 			Content:  "Test Content",
// 			Metadata: map[string]interface{}{"invalid": make(chan int)},
// 		}
// 		mockUserRepo.On("GetUserByEmail", ctx, reqInvalidMeta.To).Return(testUser, nil).Once()

// 		resp, err := svc.SendEmail(ctx, reqInvalidMeta)

// 		assert.Error(t, err)
// 		assert.Nil(t, resp)
// 		targetErr := customErrors.InternalError("Failed to marshal metadata")
// 		assert.ErrorAs(t, err, &targetErr)
// 		_, ok := err.(customErrors.AppError).Unwrap().(*json.UnsupportedTypeError)
// 		assert.True(t, ok, "Expected underlying error to be json.UnsupportedTypeError")

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertNotCalled(t, "CreateNotification", mock.Anything, mock.Anything)
// 		mockEmailService.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
// 		mockRepo.AssertNotCalled(t, "UpdateNotificationStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
// 	})

// 	t.Run("Error - Create Notification Failure", func(t *testing.T) {
// 		dbErr := errors.New("database error")
// 		mockUserRepo.On("GetUserByEmail", ctx, testReq.To).Return(testUser, nil).Once()
// 		mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(dbErr).Once()

// 		resp, err := svc.SendEmail(ctx, testReq)

// 		assert.Error(t, err)
// 		assert.Nil(t, resp)
// 		targetErr := customErrors.DatabaseError("Failed to create notification")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), dbErr)

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertExpectations(t)
// 		mockEmailService.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
// 		mockRepo.AssertNotCalled(t, "UpdateNotificationStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
// 	})

// 	t.Run("Error - Email Send Failure", func(t *testing.T) {
// 		sendErr := errors.New("sendgrid error")
// 		mockUserRepo.On("GetUserByEmail", ctx, testReq.To).Return(testUser, nil).Once()
// 		mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once()
// 		mockEmailService.On("Send", ctx, testReq).Return(sendErr).Once()
// 		// Expect UpdateNotificationStatus to be called to mark as Failed
// 		mockRepo.On("UpdateNotificationStatus", ctx, mock.AnythingOfType("uuid.UUID"), models.StatusFailed, sendErr.Error()).Return(nil).Once()

// 		resp, err := svc.SendEmail(ctx, testReq)

// 		assert.Error(t, err)
// 		assert.Nil(t, resp)
// 		targetErr := customErrors.ThirdPartyError("Failed to send notification")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), sendErr)

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertExpectations(t)
// 		mockEmailService.AssertExpectations(t)
// 	})

// 	t.Run("Error - Update Status Failure (after successful send)", func(t *testing.T) {
// 		updateErr := errors.New("update status error")
// 		mockUserRepo.On("GetUserByEmail", ctx, testReq.To).Return(testUser, nil).Once()
// 		mockRepo.On("CreateNotification", ctx, mock.AnythingOfType("*models.Notification")).Return(nil).Once()
// 		mockEmailService.On("Send", ctx, testReq).Return(nil).Once()
// 		mockRepo.On("UpdateNotificationStatus", ctx, mock.AnythingOfType("uuid.UUID"), models.StatusSent, "").Return(updateErr).Once()

// 		resp, err := svc.SendEmail(ctx, testReq)

// 		assert.Error(t, err)
// 		assert.Nil(t, resp)
// 		targetErr := customErrors.DatabaseError("Failed to update notification status")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), updateErr)

// 		mockUserRepo.AssertExpectations(t)
// 		mockRepo.AssertExpectations(t)
// 		mockEmailService.AssertExpectations(t)
// 	})
// }

// func TestNotificationService_GetNotification(t *testing.T) {
// 	ctx := context.Background()
// 	mockRepo := new(MockNotificationRepository)
// 	// UserRepo and EmailService not needed for GetNotification
// 	mockUserRepo := new(MockUserRepository)
// 	mockEmailService := new(MockEmailService)
// 	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)

// 	testID := uuid.New()
// 	testNotification := &models.Notification{
// 		ID:        testID,
// 		Type:      models.NotificationTypeEmail,
// 		Recipient: "found@example.com",
// 		Status:    models.StatusSent,
// 		CreatedAt: time.Now(),
// 	}

// 	t.Run("Success", func(t *testing.T) {
// 		mockRepo.On("GetNotificationById", ctx, testID).Return(testNotification, nil).Once()

// 		notification, err := svc.GetNotification(ctx, testID)

// 		assert.NoError(t, err)
// 		assert.NotNil(t, notification)
// 		assert.Equal(t, testNotification, notification)

// 		mockRepo.AssertExpectations(t)
// 	})

// 	t.Run("Error - Not Found", func(t *testing.T) {
// 		notFoundErr := errors.New("not found in db")
// 		mockRepo.On("GetNotificationById", ctx, testID).Return(nil, notFoundErr).Once()

// 		notification, err := svc.GetNotification(ctx, testID)

// 		assert.Error(t, err)
// 		assert.Nil(t, notification)
// 		targetErr := customErrors.NotFoundError("Notification not found")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), notFoundErr)

// 		mockRepo.AssertExpectations(t)
// 	})
// }

// func TestNotificationService_ListNotifications(t *testing.T) {
// 	ctx := context.Background()
// 	mockRepo := new(MockNotificationRepository)
// 	mockUserRepo := new(MockUserRepository)
// 	mockEmailService := new(MockEmailService)
// 	svc := service.NewNotificationService(mockRepo, mockUserRepo, mockEmailService)

// 	testNotifications := []*models.Notification{
// 		{ID: uuid.New(), Recipient: "test1@example.com"},
// 		{ID: uuid.New(), Recipient: "test2@example.com"},
// 	}
// 	totalCount := 15

// 	t.Run("Success - Valid Page and Size", func(t *testing.T) {
// 		page, size := 2, 5
// 		mockRepo.On("ListNotifications", ctx, page, size).Return(testNotifications, totalCount, nil).Once()

// 		notifications, total, err := svc.ListNotifications(ctx, page, size)

// 		assert.NoError(t, err)
// 		assert.Equal(t, testNotifications, notifications)
// 		assert.Equal(t, totalCount, total)

// 		mockRepo.AssertExpectations(t)
// 	})

// 	t.Run("Success - Page < 1", func(t *testing.T) {
// 		page, size := 0, 5
// 		expectedPage := 1 // Should default to 1
// 		mockRepo.On("ListNotifications", ctx, expectedPage, size).Return(testNotifications, totalCount, nil).Once()

// 		notifications, total, err := svc.ListNotifications(ctx, page, size)

// 		assert.NoError(t, err)
// 		assert.Equal(t, testNotifications, notifications)
// 		assert.Equal(t, totalCount, total)

// 		mockRepo.AssertExpectations(t)
// 	})

// 	t.Run("Success - Size < 1", func(t *testing.T) {
// 		page, size := 1, 0
// 		expectedSize := 10 // Should default to 10
// 		mockRepo.On("ListNotifications", ctx, page, expectedSize).Return(testNotifications, totalCount, nil).Once()

// 		notifications, total, err := svc.ListNotifications(ctx, page, size)

// 		assert.NoError(t, err)
// 		assert.Equal(t, testNotifications, notifications)
// 		assert.Equal(t, totalCount, total)

// 		mockRepo.AssertExpectations(t)
// 	})

// 	t.Run("Success - Size > 10", func(t *testing.T) {
// 		page, size := 1, 15
// 		expectedSize := 10 // Should default to 10
// 		mockRepo.On("ListNotifications", ctx, page, expectedSize).Return(testNotifications, totalCount, nil).Once()

// 		notifications, total, err := svc.ListNotifications(ctx, page, size)

// 		assert.NoError(t, err)
// 		assert.Equal(t, testNotifications, notifications)
// 		assert.Equal(t, totalCount, total)

// 		mockRepo.AssertExpectations(t)
// 	})

// 	t.Run("Error - Database Error", func(t *testing.T) {
// 		page, size := 1, 10
// 		dbErr := errors.New("failed to fetch")
// 		mockRepo.On("ListNotifications", ctx, page, size).Return(nil, 0, dbErr).Once()

// 		notifications, total, err := svc.ListNotifications(ctx, page, size)

// 		assert.Error(t, err)
// 		assert.Nil(t, notifications)
// 		assert.Equal(t, 0, total)
// 		targetErr := customErrors.DatabaseError("Failed to fetch notifications")
// 		assert.ErrorAs(t, err, &targetErr)
// 		assert.ErrorIs(t, err.(customErrors.AppError).Unwrap(), dbErr)

// 		mockRepo.AssertExpectations(t)
// 	})
// }
