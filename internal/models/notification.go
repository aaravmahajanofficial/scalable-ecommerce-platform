package models

import (
	"encoding/json"
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
	StatusPending NotificationStatus = "pending"
	StatusSent    NotificationStatus = "sent"
	StatusFailed  NotificationStatus = "failed"
)

type Notification struct {
	ID           uuid.UUID          `json:"id"`
	Type         NotificationType   `json:"type"`
	Recipient    string             `json:"recipient"`
	Subject      string             `json:"subject,omitempty"`
	Content      string             `json:"content"`
	Status       NotificationStatus `json:"status"`
	ErrorMessage string             `json:"error_message,omitempty"`
	Metadata     json.RawMessage    `json:"metadata,omitempty"      swaggertype:"object"` // highly dynamic
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	SentAt       *time.Time         `json:"sent_at,omitempty"`
}

type EmailNotificationRequest struct {
	To          string            `json:"to"                     validate:"required,email"`
	Subject     string            `json:"subject"                validate:"required"`
	Content     string            `json:"content"                validate:"required"`
	HTMLContent string            `json:"html_content,omitempty"`
	CC          []string          `json:"cc,omitempty"           validate:"omitempty,dive,email"`
	BCC         []string          `json:"bcc,omitempty"          validate:"omitempty,dive,email"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type NotificationResponse struct {
	ID        uuid.UUID          `json:"id"`
	Type      NotificationType   `json:"type"`
	Status    NotificationStatus `json:"status"`
	Recipient string             `json:"recipient,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}

type NotificationListResponse struct {
	Notifications []*Notification `json:"notifications"`
	Total         int             `json:"total"`
	Page          int             `json:"page"`
	PageSize      int             `json:"page_size"`
}
