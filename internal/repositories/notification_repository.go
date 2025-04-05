package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/google/uuid"
)

type NotificationRepository struct {
	DB *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, notification *models.Notification) error {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO notifications (id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err := r.DB.ExecContext(dbCtx, query, notification.ID, notification.Type, notification.Recipient, notification.Subject, notification.Content, notification.Status, notification.ErrorMessage, notification.Metadata)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil

}

func (r *NotificationRepository) GetNotificationById(ctx context.Context, id uuid.UUID) (*models.Notification, error) {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	result := &models.Notification{}

	var metadata []byte

	err := r.DB.QueryRowContext(dbCtx, query, id).Scan(&result.ID, &result.Type, &result.Recipient, &result.Subject, &result.Content, &result.Status, &result.ErrorMessage, &metadata, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return &models.Notification{}, fmt.Errorf("failed to create notification: %w", err)
	}

	result.Metadata = json.RawMessage(metadata)

	return result, nil
}

func (r *NotificationRepository) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus, errorMsg string) error {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		UPDATE notifications SET status = $1, error_message = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := r.DB.ExecContext(dbCtx, query, status, errorMsg, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update the notification status: %w", err)
	}

	updatedRows, err := result.RowsAffected()

	if err != nil {
		return fmt.Errorf("failed to get updated rows: %w", err)
	}

	if updatedRows == 0 {

		return fmt.Errorf("notification not found: %s", id)

	}

	return nil

}

func (r *NotificationRepository) ListNotifications(ctx context.Context, page int, size int) ([]*models.Notification, int, error) {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	var total int
	countQuery := `SELECT COUNT(*) FROM products`
	err := r.DB.QueryRowContext(dbCtx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offSet := (page - 1) * size

	query := `
		SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.DB.QueryContext(dbCtx, query, size, offSet)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}

	defer rows.Close()

	notifications := []*models.Notification{}

	for rows.Next() {

		var notification models.Notification
		var metadata []byte

		err := rows.Scan(&notification.ID, &notification.Type, &notification.Recipient, &notification.Subject, &notification.Content, &notification.Status, &metadata, &notification.ErrorMessage, &notification.CreatedAt, &notification.UpdatedAt)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notifications: %w", err)
		}

		notification.Metadata = json.RawMessage(metadata)

		notifications = append(notifications, &notification)

	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over the rows: %w", err)
	}

	return notifications, total, nil

}
