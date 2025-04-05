package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendGrid"
	"github.com/google/uuid"
)

type NotificationService interface {
	SendEmail(ctx context.Context, req *models.EmailNotificationRequest) (*models.NotificationResponse, error)
	GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, int, error)
}

type notificationService struct {
	repo         *repository.NotificationRepository
	userRepo     *repository.UserRepository
	emailService sendGrid.EmailService
}

func NewNotificationService(repo *repository.NotificationRepository, userRepo *repository.UserRepository, emailService sendGrid.EmailService) NotificationService {
	return &notificationService{repo: repo, emailService: emailService}
}

// SendEmail implements NotificationService.
func (s *notificationService) SendEmail(ctx context.Context, req *models.EmailNotificationRequest) (*models.NotificationResponse, error) {

	_, err := s.userRepo.GetUserByEmail(ctx, req.To)
	if err != nil {
		return nil, errors.NotFoundError("User not found").WithError(err)
	}

	var metadataJSON json.RawMessage

	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, errors.InternalError("Failed to marshal metadata").WithError(err)
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
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		return nil, errors.DatabaseError("Failed to create notification").WithError(err)
	}

	err = s.emailService.Send(ctx, req)
	if err != nil {

		notification.Status = models.StatusFailed
		notification.ErrorMessage = err.Error()

		_ = s.repo.UpdateNotificationStatus(ctx, notification.ID, models.StatusFailed, notification.ErrorMessage)

		return nil, errors.ThirdPartyError("Failed to send notification").WithError(err)

	}

	// Update the notification status if sent successfully
	notification.Status = models.StatusSent

	if err := s.repo.UpdateNotificationStatus(ctx, notification.ID, models.StatusSent, ""); err != nil {
		return nil, errors.DatabaseError("Failed to update notification status").WithError(err)
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
func (s *notificationService) GetNotification(ctx context.Context, id uuid.UUID) (*models.Notification, error) {

	notification, err := s.repo.GetNotificationById(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Notification not found").WithError(err)
	}

	return notification, err
}

// ListNotifications implements NotificationService.
func (s *notificationService) ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, int, error) {

	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	notifications, total, err := s.repo.ListNotifications(ctx, page, size)
	if err != nil {
		return nil, 0, errors.DatabaseError("Failed to fetch notifications").WithError(err)
	}

	return notifications, total, nil
}
