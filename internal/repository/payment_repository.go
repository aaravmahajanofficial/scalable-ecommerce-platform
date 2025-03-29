package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type PaymentRepository struct {
	DB *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

func (p *PaymentRepository) CreatePayment(ctx context.Context, payment *models.Payment) error {

	query := `
		INSERT INTO payments (id, amount, currency, customer_id, description, status, payment_method, stripe_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8,NOW(), NOW())
	`

	_, err := p.DB.ExecContext(ctx, query, &payment.ID, &payment.Amount, &payment.Currency, &payment.CustomerID, &payment.Description, &payment.Status, &payment.PaymentMethod, &payment.StripeID)

	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	return nil
}

func (p *PaymentRepository) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {

	payment := &models.Payment{}

	query := `
		SELECT id, amount, currency, customer_id, description, status, payment_method, stripe_id, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	err := p.DB.QueryRowContext(ctx, query, id).Scan(&payment.ID, &payment.Amount, &payment.Currency, &payment.CustomerID, &payment.Description, &payment.Status, &payment.PaymentMethod, &payment.StripeID, &payment.CreatedAt, &payment.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get the payment: %w", err)
	}

	return payment, nil

}

func (p *PaymentRepository) UpdatePaymentStatus(ctx context.Context, id string, status models.PaymentStatus) error {

	query := `
		UPDATE payments SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := p.DB.ExecContext(ctx, query, status, time.Now(), id)

	return err

}

func (p *PaymentRepository) ListPaymentsOfCustomer(ctx context.Context, customerID string, page, size int) ([]*models.Payment, int, error) {

	// Offset
	offset := (page - 1) * size

	query := `
		SELECT id, customer_id, amount, currency, description, status, payment_method, stripe_id, created_at, updated_at
		FROM payments
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.DB.QueryContext(ctx, query, customerID, size, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list the payments: %w", err)
	}

	defer rows.Close()

	var payments []*models.Payment

	for rows.Next() {

		payment := &models.Payment{}

		err := rows.Scan(&payment.ID, &payment.CustomerID, &payment.Amount, &payment.Currency, &payment.Description, &payment.Status, &payment.PaymentMethod, &payment.StripeID, &payment.CreatedAt, &payment.UpdatedAt)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan the payments: %w", err)
		}

		payments = append(payments, payment)

	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return payments, len(payments), nil

}
