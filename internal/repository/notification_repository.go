package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
)

type NotificationRepository struct {
	DB *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{DB: db}
}

func (n *NotificationRepository) CreateNotification(ctx context.Context, notification models.Notification) (*models.Notification, error) {

	query := `
		INSERT INTO notifications (id, type, recipient, subject, content, status, error, created_at, updated_at, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, type, recipient, subject, content, status, error, created_at, updated_at, sent_at
	`

	row := n.DB.QueryRowContext(ctx, query, notification.ID, notification.Type, notification.Recipient, notification.Subject, notification.Content, notification.Status, notification.ErrorMessage, notification.CreatedAt, notification.UpdatedAt, notification.SentAt)

	result := &models.Notification{}

	// handle the nullable column
	var sentAt sql.NullTime
	var errorMsg sql.NullString

	err := row.Scan(&result.ID, &result.Type, &result.Recipient, &result.Subject, &result.Content, &result.Status, &errorMsg, &result.CreatedAt, &result.UpdatedAt, &sentAt)

	if err != nil {
		return &models.Notification{}, fmt.Errorf("failed to create notification: %w", err)
	}

	if sentAt.Valid {
		result.SentAt = &sentAt.Time
	}

	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
	}

	return result, nil

}

func (n *NotificationRepository) GetNotificationById(ctx context.Context, id uuid.UUID) (*models.Notification, error) {

	query := `
		SELECT id, type, recipient, subject, content, status, error, created_at, updated_at, sent_at
		FROM notifications
		WHERE id = $1
	`

	result := &models.Notification{}

	// handle the nullable column
	var sentAt sql.NullTime
	var errorMsg sql.NullString

	err := n.DB.QueryRowContext(ctx, query, id).Scan(&result.ID, &result.Type, &result.Recipient, &result.Subject, &result.Content, &result.Status, &errorMsg, &result.CreatedAt, &result.UpdatedAt, &sentAt)

	if err != nil {
		return &models.Notification{}, fmt.Errorf("failed to create notification: %w", err)
	}

	if sentAt.Valid {
		result.SentAt = &sentAt.Time
	}

	if errorMsg.Valid {
		result.ErrorMessage = errorMsg.String
	}

	return result, nil

}

func (n *NotificationRepository) UpdateNotificationStatus(ctx context.Context, id uuid.UUID, status models.NotificationStatus, errorMsg string) error {

	query := `
		UPDATE notifications SET status = $2, updated_at = $3, error = $4
	`

	// a slice containing values of any type
	args := []interface{}{id, status, time.Now(), errorMsg}

	if status == models.StatusSent {

		query += ` sent_at = $5 WHERE id = $1`
		args = append(args, time.Now())

	} else {

		query += ` WHERE id = $1`
	}

	result, err := n.DB.ExecContext(ctx, query, args)

	if err != nil {
		return fmt.Errorf("failed to update the notification status: %w", err)
	}

	updatedRows, err := result.RowsAffected()

	if err != nil {
		return fmt.Errorf("failed to get updated rows: %w", err)
	}

	if updatedRows == 0 {

		return fmt.Errorf("notification not found")

	}

	return nil

}

func (n *NotificationRepository) ListPending(ctx context.Context, page int, size int) ([]models.Notification, int, error) {

	offSet := (page - 1) * size

	query := `
		SELECT id, type, recipient, subject, content, status, error, created_at, updated_at, sent_at
		FROM notifications
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT = $2, OFFSET = $3
	`

	rows, err := n.DB.QueryContext(ctx, query, size, offSet)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to query pending notifications: %w", err)
	}

	defer rows.Close()

	notifications := []models.Notification{}

	if rows.Next() {

		var notification models.Notification
		var sentAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(&notification.ID, &notification.Type, &notification.Recipient, &notification.Subject, &notification.Content, &notification.Status, &errorMsg, &notification.CreatedAt, &notification.UpdatedAt, &sentAt)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan pending notifications: %w", err)
		}

		if sentAt.Valid {
			notification.SentAt = &sentAt.Time
		}

		if errorMsg.Valid {
			notification.ErrorMessage = errorMsg.String
		}

		notifications = append(notifications, notification)

	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating over the rows: %w", err)
	}

	return notifications, len(notifications), nil

}
