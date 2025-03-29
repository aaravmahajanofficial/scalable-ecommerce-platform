package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID            uuid.UUID     `json:"id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	Description   string        `json:"description"`
	Status        PaymentStatus `json:"payment_status"`
	PaymentMethod string        `json:"payment_method"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type PaymentIntent struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type CreatePaymentRequest struct {
	OrderID       uuid.UUID `json:"order_id" validate:"required"`
	Amount        float64   `json:"amount" validate:"required,gt=0"`
	Currency      string    `json:"currency"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Description   string    `json:"description"`
	PaymentMethod string    `json:"payment_method" validate:"required"`
}

type PaymentResponse struct {
	PaymentIntent *PaymentIntent `json:"payment_intent"`
	ClientSecret  string         `json:"client_secret,omitempty"`
}
