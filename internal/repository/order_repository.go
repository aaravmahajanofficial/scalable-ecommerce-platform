package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
)

type OrderRepository struct {
	DB *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{DB: db}
}

func (o *OrderRepository) CreateOrder(order *models.Order) error {

	shipping_address, err := json.Marshal(order.ShippingAddress)

	if err != nil {
		return fmt.Errorf("failed to marshal shipping address: %w", err)
	}

	// Insert an order
	query := `
		INSERT INTO orders (id, customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`

	_, err = o.DB.Exec(query, order.ID, order.CustomerID, order.Status, order.TotalAmount, order.PaymentStatus, order.PaymentIntentID, shipping_address)

	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert order items
	for _, item := range order.Items {

		query := `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
		`

		_, err := o.DB.Exec(query, item.ID, order.ID, item.ProductID, item.Quantity, item.UnitPrice)

		if err != nil {
			return fmt.Errorf("failed to insert an order item: %w", err)
		}

	}

	return nil
}

/*
Get the order
Get the order items
*/
func (o *OrderRepository) GetOrderById(id uuid.UUID) (*models.Order, error) {

	order := &models.Order{
		ID: id,
	}

	query := `
		SELECT customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	var jsonData []byte

	err := o.DB.QueryRow(query, id).Scan(&order.CustomerID, &order.Status, &order.TotalAmount, &order.PaymentStatus, &order.PaymentIntentID, &jsonData, &order.CreatedAt, &order.UpdatedAt)

	json.Unmarshal(jsonData, &order.ShippingAddress)

	if err != nil {
		return nil, fmt.Errorf("failed to get the order: %w", err)
	}

	// Get the order items
	query = `
		SELECT id, product_id, quantity, unit_price, created_at
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := o.DB.Query(query, id)

	if err != nil {
		return nil, fmt.Errorf("failed to get the order items: %w", err)
	}

	defer rows.Close()

	var items []models.OrderItem

	for rows.Next() {

		var item models.OrderItem

		err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.CreatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan order item: %w", err)
		}

		item.OrderID = order.ID

		items = append(items, item)

	}

	order.Items = items

	return order, nil

}

// List the orders of the customer, along with pagination
/*
	1. Get the orders of the customer
	2.

*/
func (o *OrderRepository) ListOrdersByCustomer(customerID uuid.UUID, page int, size int) ([]models.Order, int, error) {

	// Get the total number of orders
	var count int

	query := `
		SELECT COUNT(*) FROM orders WHERE customer_id = $1
	`

	err := o.DB.QueryRow(query, customerID).Scan(&count)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count the orders: %w", err)
	}

	// Offset
	offset := (page - 1) * size

	// Get orders with pagination
	query = `
		SELECT id, customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
		FROM orders
		WHERE customer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := o.DB.Query(query, customerID, size, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}

	defer rows.Close()

	var orders []models.Order

	for rows.Next() {

		var order models.Order

		order.CustomerID = customerID

		err := rows.Scan(&order.CustomerID, &order.Status, &order.TotalAmount, &order.PaymentStatus, &order.PaymentIntentID, &order.ShippingAddress, &order.CreatedAt, &order.UpdatedAt)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan orders: %w", err)
		}

		orders = append(orders, order)

	}

	// now for each order we have to fetch the respective order items
	query = `
		SELECT id, product_id, quantity, unit_price, created_at
		FROM order_items
		WHERE order_id = $1
	`

	for i := range orders {

		// Get the order items
		itemsRows, err := o.DB.Query(query, orders[i].ID)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to get the orders: %w", err)
		}

		defer itemsRows.Close()

		var items []models.OrderItem

		for itemsRows.Next() {

			var item models.OrderItem

			err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.CreatedAt)

			if err != nil {
				return nil, 0, fmt.Errorf("failed to scan order items: %w", err)
			}
			item.OrderID = orders[i].ID

			items = append(items, item)

		}

		orders[i].Items = items
	}

	return orders, count, nil
}

// Update Order status
func (o *OrderRepository) UpdateOrderStatus(id uuid.UUID, status models.OrderStatus) error {

	query := `
		UPDATE orders SET status = $1, updated_at = $2, WHERE id = $3
	`

	result, err := o.DB.Exec(query, status, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	updatedRows, err := result.RowsAffected()

	if updatedRows == 0 {

		return fmt.Errorf("order not found")

	} else if err != nil {
		return fmt.Errorf("failed to update the order: %w", err)
	}

	return nil

}

// Update the Payment Status and Payment Intent ID of an order
func (o *OrderRepository) UpdatePaymentStatus(id uuid.UUID, status models.PaymentStatus, paymentIntentID string) error {

	query := `
		UPDATE orders set payment_status = $1, payment_intent_id = $2 updated_at = $3 WHERE id = $4
	`

	result, err := o.DB.Exec(query, status, paymentIntentID, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	updatedRows, err := result.RowsAffected()

	if updatedRows == 0 {
		return fmt.Errorf("order not found")
	} else if err != nil {
		return fmt.Errorf("failed to update the order: %w", err)
	}

	return nil

}
