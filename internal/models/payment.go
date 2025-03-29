package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID            uuid.UUID     `json:"id"`
	CustomerID    uuid.UUID     `json:"customer_id"`
	Amount        int64         `json:"amount"`
	Currency      string        `json:"currency"`
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
	Amount        int64     `json:"amount" validate:"required,gt=0"`
	Currency      string    `json:"currency"`
	CustomerID    uuid.UUID `json:"customer_id"`
	Description   string    `json:"description"`
	PaymentMethod string    `json:"payment_method" validate:"required"`
}

type PaymentRequest struct {
	CustomerID    uuid.UUID `json:"customer_id" validate:"required"`
	Amount        int64     `json:"amount" validate:"required,gt=0"`
	Currency      string    `json:"currency" validate:"required,le=3"`
	Description   string    `json:"description" validate:"required"`
	PaymentMethod string    `json:"payment_method" validate:"required"`
	CardNumber    string    `json:"card_number" validate:"required_if=PaymentMethod card,omitempty,credit_card"`
	CardExpMonth  int       `json:"card_exp_month" validate:"required_if=PaymentMethod card,omitempty,min=1,max=12"`
	CardExpYear   int       `json:"card_exp_year" validate:"required_if=PaymentMethod card,omitempty,min=2025"`
	CardCVC       string    `json:"card_cvc" validate:"required_if=PaymentMethod card,omitempty,len=3"`
}

type PaymentResponse struct {
	Payment       *Payment `json:"payment"`
	ClientSecret  string   `json:"client_secret,omitempty"`
	PaymentStatus string   `json:"payment_status"`
	Message       string   `json:"message,omitempty"`
}
