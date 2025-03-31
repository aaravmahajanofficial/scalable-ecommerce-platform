package models

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	EmailNotification NotificationType = "email"
	SMSNotification   NotificationType = "sms"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusCancelled NotificationStatus = "cancelled"
)

type Notification struct {
	ID        uuid.UUID          `json:"id"`
	Type      NotificationType   `json:"type"`
	Recipient string             `json:"recipient"`
	Subject   string             `json:"subject,omitempty"`
	Content   string             `json:"content"`
	Status    NotificationStatus `json:"status"`
	Error     string             `json:"error,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
}

type CreateNotificationRequest struct {
	Type      NotificationType `json:"type" validate:"required,oneof=email sms"`
	Recipient string           `json:"recipient" validate:"required"`
	Subject   string           `json:"subject,omitempty" validate:"required_if=Type email"`
	Content   string           `json:"content" validate:"required"`
}
