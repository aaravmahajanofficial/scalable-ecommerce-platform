package models

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusFailed    NotificationStatus = "failed"
	StatusScheduled NotificationStatus = "scheduled"
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

type EmailNotificationRequest struct {
	Subject   string            `json:"subject" validate:"required"`
	Content   string            `json:"content" validate:"required"`
	Recipient string            `json:"recipient" validate:"required,email"`
	CC        []string          `json:"cc,omitempty" validate:"omitempty,dive,email"`
	BCC       []string          `json:"bcc,omitempty" validate:"omitempty,dive,email"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type NotificationResponse struct {
	ID        uuid.UUID          `json:"id"`
	Type      NotificationType   `json:"type"`
	Status    NotificationStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	SentAt    *time.Time         `json:"sent_at,omitempty"`
}
