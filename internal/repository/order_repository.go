package repository

import (
	"database/sql"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (o *OrderRepository) CreateOrder(order *models.Order) error {

	// Insert an order
	query := `
		INSERT INTO orders (id, customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, items, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err := o.DB.Exec(query, order.ID, order.CustomerID, order.Status, order.TotalAmount, order.PaymentStatus, order.PaymentIntentID, order.ShippingAddress, order.Items)

	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert an order
	for _, item := range order.Items {

		query := `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`

		_, err := o.DB.Exec(query, item.ID, order.ID, item.ProductID, item.Quantity, item.UnitPrice, item.CreatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert an order item: %w", err)
		}

	}

	return nil
}
