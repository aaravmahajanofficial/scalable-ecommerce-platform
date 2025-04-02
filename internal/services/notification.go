package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendGrid"
	"github.com/google/uuid"
)

type NotificationService interface {
	SendEmail(ctx context.Context, req *models.EmailNotificationRequest) (*models.NotificationResponse, error)
	GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, error)
}

type notificationService struct {
	repo         *repository.NotificationRepository
	emailService sendGrid.EmailService
}

func NewNotificationService(repo *repository.NotificationRepository, emailService sendGrid.EmailService) NotificationService {
	return &notificationService{repo: repo, emailService: emailService}
}

// SendEmail implements NotificationService.
func (n *notificationService) SendEmail(ctx context.Context, req *models.EmailNotificationRequest) (*models.NotificationResponse, error) {

	var metadataJSON json.RawMessage

	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		metadataJSON = metadataBytes

	}

	notification := &models.Notification{
		ID:        uuid.New(),
		Type:      models.NotificationTypeEmail,
		Recipient: req.To,
		Subject:   req.Subject,
		Content:   req.Content,
		Status:    models.StatusPending,
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to the database
	if err := n.repo.CreateNotification(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification record: %w", err)
	}

	err := n.emailService.Send(ctx, req)

	if err != nil {

		notification.Status = models.StatusFailed
		notification.ErrorMessage = err.Error()

		_ = n.repo.UpdateNotificationStatus(ctx, notification.ID, models.StatusFailed, notification.ErrorMessage)

		return nil, fmt.Errorf("failed to send email: %w", err)

	}

	// Update the notification status if sent successfully
	notification.Status = models.StatusSent

	if err := n.repo.UpdateNotificationStatus(ctx, notification.ID, models.StatusSent, ""); err != nil {
		return nil, fmt.Errorf("notification sent successfully but failed to update notification status: %w", err)
	}

	return &models.NotificationResponse{
		ID:        notification.ID,
		Type:      notification.Type,
		Status:    notification.Status,
		Recipient: notification.Recipient,
		CreatedAt: notification.CreatedAt,
	}, nil

}

// GetNotification implements NotificationService.
func (n *notificationService) GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error) {

	return n.repo.GetNotificationById(ctx, id)

}

// ListNotifications implements NotificationService.
func (n *notificationService) ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, error) {

	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	notifications, err := n.repo.ListNotifications(ctx, page, size)

	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}

	return notifications, nil

}
