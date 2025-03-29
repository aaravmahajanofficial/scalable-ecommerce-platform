package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
)

type PaymentRepository struct {
	DB *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

// ID            uuid.UUID     `json:"id"`
// Amount        float64       `json:"amount"`
// Currency      string        `json:"currency"`
// CustomerID    uuid.UUID     `json:"customer_id"`
// Description   string        `json:"description"`
// Status        PaymentStatus `json:"payment_status"`
// PaymentMethod string        `json:"payment_method"`
// CreatedAt     time.Time     `json:"created_at"`
// UpdatedAt     time.Time     `json:"updated_at"`

func (p *PaymentRepository) CreatePayment(ctx context.Context, payment *models.Payment) error {

	query := `
		INSERT INTO payments (amount, currency, customer_id, description, status, payment_method, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`

	_, err := p.DB.ExecContext(ctx, query, &payment.Amount, &payment.Currency, &payment.CustomerID, &payment.Description, &payment.Status, &payment.PaymentMethod)

	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	return nil
}

func (p *PaymentRepository) GetPayment(ctx context.Context, id uuid.UUID) (*models.Payment, error) {

	payment := &models.Payment{}

	query := `
		SELECT id, amount, currency, customer_id, description, status, payment_method, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	err := p.DB.QueryRowContext(ctx, query, id).Scan(&payment.Amount, &payment.CustomerID, &payment.Description, &payment.Status, &payment.PaymentMethod, &payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get the payment: %w", err)
	}

	return payment, nil

}
