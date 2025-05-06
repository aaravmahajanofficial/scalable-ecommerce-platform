package repository_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupOrderRepoTest(t *testing.T) (repository.OrderRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err, "Failed to create sqlmock")

	t.Cleanup(func() {
		db.Close()
	})

	repo := repository.NewOrderRepository(db)
	require.NotNil(t, repo, "NewOrderRepository should return a non-nil repository")

	return repo, mock
}

func TestNewOrderRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewOrderRepository(db)
	assert.NotNil(t, repo, "NewOrderRepository should return a non-nil repository")
}

func TestCreateOrder(t *testing.T) {
	// Arrange
	repo, mock := setupOrderRepoTest(t)
	ctx := t.Context()

	orderID := uuid.New()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()
	itemID1 := uuid.New()
	itemID2 := uuid.New()
	now := time.Now()

	testOrder := &models.Order{
		ID:              orderID,
		CustomerID:      customerID,
		Status:          models.OrderStatusPending,
		TotalAmount:     250.00,
		PaymentStatus:   models.PaymentStatusPending,
		PaymentIntentID: "pi_123",
		ShippingAddress: &models.Address{
			Street:     "123 Test St",
			City:       "Testville",
			State:      "TS",
			PostalCode: "12345",
			Country:    "US",
		},
		Items: []models.OrderItem{
			{ID: itemID1, OrderID: orderID, ProductID: productID1, Quantity: 2, UnitPrice: 50.00, CreatedAt: now},
			{ID: itemID2, OrderID: orderID, ProductID: productID2, Quantity: 1, UnitPrice: 150.00, CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	shippingAddrJSON, err := json.Marshal(testOrder.ShippingAddress)
	require.NoError(t, err, "Failed to marshal shipping address for test setup")

	expectedOrderInsertSQL := regexp.QuoteMeta(`
        INSERT INTO orders (id, customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
    `)
	expectedItemInsertSQL := regexp.QuoteMeta(`
            INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, created_at)
            VALUES ($1, $2, $3, $4, $5, NOW())
        `)

	t.Run("Success - Create Order", func(t *testing.T) {
		// Expect the order insertion
		mock.ExpectExec(expectedOrderInsertSQL).
			WithArgs(testOrder.ID, testOrder.CustomerID, testOrder.Status, testOrder.TotalAmount, testOrder.PaymentStatus, testOrder.PaymentIntentID, shippingAddrJSON).
			WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate 1 row inserted

		// Expect the first item insertion
		mock.ExpectExec(expectedItemInsertSQL).
			WithArgs(testOrder.Items[0].ID, testOrder.ID, testOrder.Items[0].ProductID, testOrder.Items[0].Quantity, testOrder.Items[0].UnitPrice).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect the second item insertion
		mock.ExpectExec(expectedItemInsertSQL).
			WithArgs(testOrder.Items[1].ID, testOrder.ID, testOrder.Items[1].ProductID, testOrder.Items[1].Quantity, testOrder.Items[1].UnitPrice).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Act
		err := repo.CreateOrder(ctx, testOrder)

		// Assert
		assert.NoError(t, err, "CreateOrder should succeed")
	})

	t.Run("Failure - Order Insert Error", func(t *testing.T) {
		dbErr := errors.New("DB error on order insert")
		// Expect the order insertion to fail
		mock.ExpectExec(expectedOrderInsertSQL).
			WithArgs(testOrder.ID, testOrder.CustomerID, testOrder.Status, testOrder.TotalAmount, testOrder.PaymentStatus, testOrder.PaymentIntentID, shippingAddrJSON).
			WillReturnError(dbErr)

		// Act
		err := repo.CreateOrder(ctx, testOrder)

		// Assert
		require.Error(t, err, "CreateOrder should fail when order insert fails")
		assert.ErrorContains(t, err, "failed to insert order", "Error message should indicate order insert failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
	})

	t.Run("Failure - Item Insert Error", func(t *testing.T) {
		dbErr := errors.New("DB error on item insert")
		// Expect the order insertion to succeed
		mock.ExpectExec(expectedOrderInsertSQL).
			WithArgs(testOrder.ID, testOrder.CustomerID, testOrder.Status, testOrder.TotalAmount, testOrder.PaymentStatus, testOrder.PaymentIntentID, shippingAddrJSON).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect the first item insertion to fail
		mock.ExpectExec(expectedItemInsertSQL).
			WithArgs(testOrder.Items[0].ID, testOrder.ID, testOrder.Items[0].ProductID, testOrder.Items[0].Quantity, testOrder.Items[0].UnitPrice).
			WillReturnError(dbErr)

		// Act
		err := repo.CreateOrder(ctx, testOrder)

		// Assert
		require.Error(t, err, "CreateOrder should fail when item insert fails")
		assert.ErrorContains(t, err, "failed to insert an order item", "Error message should indicate item insert failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
	})
}

func TestGetOrderByID(t *testing.T) {
	repo, mock := setupOrderRepoTest(t)
	ctx := t.Context()

	orderID := uuid.New()
	customerID := uuid.New()
	productID1 := uuid.New()
	itemID1 := uuid.New()
	now := time.Now()

	expectedAddress := &models.Address{
		Street: "456 Get St", City: "Gettown", State: "GT", PostalCode: "67890", Country: "CA",
	}
	expectedAddrJSON, err := json.Marshal(expectedAddress)
	require.NoError(t, err, "Failed to marshal address for test")

	expectedOrder := &models.Order{
		ID:              orderID,
		CustomerID:      customerID,
		Status:          models.OrderStatusConfirmed,
		TotalAmount:     100.00,
		PaymentStatus:   models.PaymentStatusSucceeded,
		PaymentIntentID: "pi_123",
		ShippingAddress: expectedAddress,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now,
		Items: []models.OrderItem{
			{ID: itemID1, OrderID: orderID, ProductID: productID1, Quantity: 1, UnitPrice: 100.00, CreatedAt: now.Add(-time.Hour)},
		},
	}

	expectedOrderQuerySQL := regexp.QuoteMeta(`
        SELECT customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
        FROM orders
        WHERE id = $1
    `)
	expectedItemsQuerySQL := regexp.QuoteMeta(`
        SELECT id, product_id, quantity, unit_price, created_at
        FROM order_items
        WHERE order_id = $1
    `)

	t.Run("Success - Get Order By ID", func(t *testing.T) {
		// Mock order query
		orderRows := sqlmock.NewRows([]string{"customer_id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrder.CustomerID, expectedOrder.Status, expectedOrder.TotalAmount, expectedOrder.PaymentStatus, expectedOrder.PaymentIntentID, expectedAddrJSON, expectedOrder.CreatedAt, expectedOrder.UpdatedAt)
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnRows(orderRows)

		// Mock items query
		itemRows := sqlmock.NewRows([]string{"id", "product_id", "quantity", "unit_price", "created_at"}).
			AddRow(expectedOrder.Items[0].ID, expectedOrder.Items[0].ProductID, expectedOrder.Items[0].Quantity, expectedOrder.Items[0].UnitPrice, expectedOrder.Items[0].CreatedAt)
		mock.ExpectQuery(expectedItemsQuerySQL).WithArgs(orderID).WillReturnRows(itemRows)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		assert.NoError(t, err, "GetOrderByID should succeed")
		require.NotNil(t, order, "Order should not be nil on success")
		assert.Equal(t, expectedOrder.ID, order.ID)
		assert.Equal(t, expectedOrder.CustomerID, order.CustomerID)
		assert.Equal(t, expectedOrder.Status, order.Status)
		assert.Equal(t, expectedOrder.TotalAmount, order.TotalAmount)
		assert.Equal(t, expectedOrder.PaymentStatus, order.PaymentStatus)
		assert.Equal(t, expectedOrder.PaymentIntentID, order.PaymentIntentID)
		assert.Equal(t, expectedOrder.ShippingAddress, order.ShippingAddress)
		assert.WithinDuration(t, expectedOrder.CreatedAt, order.CreatedAt, time.Second)
		assert.WithinDuration(t, expectedOrder.UpdatedAt, order.UpdatedAt, time.Second)
		assert.Equal(t, expectedOrder.Items, order.Items)
	})

	t.Run("Failure - Order Not Found", func(t *testing.T) {
		// Mock order query returning no rows
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnError(sql.ErrNoRows)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		require.Error(t, err, "GetOrderByID should fail when order not found")
		assert.ErrorIs(t, err, sql.ErrNoRows, "Error should wrap sql.ErrNoRows")
		assert.Nil(t, order, "Returned order should be nil")
	})

	t.Run("Failure - Order Scan Error", func(t *testing.T) {
		// Mock order query with incorrect columns to cause scan error
		orderRows := sqlmock.NewRows([]string{"customer_id", "status"}).AddRow(customerID, "only_two_columns")
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnRows(orderRows)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		require.Error(t, err, "GetOrderByID should fail on order scan error")
		assert.ErrorContains(t, err, "failed to get the order", "Error message should indicate failure")
		assert.ErrorContains(t, err, "Scan", "Error should be related to scanning")
		assert.Nil(t, order, "Returned order should be nil")
	})

	t.Run("Failure - Address Unmarshal Error", func(t *testing.T) {
		// Mock order query with invalid JSON for address
		invalidJSON := []byte(`{"street": "Invalid`)
		orderRows := sqlmock.NewRows([]string{"customer_id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrder.CustomerID, expectedOrder.Status, expectedOrder.TotalAmount, expectedOrder.PaymentStatus, expectedOrder.PaymentIntentID, invalidJSON, expectedOrder.CreatedAt, expectedOrder.UpdatedAt)
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnRows(orderRows)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		require.Error(t, err, "GetOrderByID should fail on address unmarshal error")
		assert.ErrorContains(t, err, "failed to unmarshal shipping address", "Error message should indicate unmarshal failure")
		assert.Nil(t, order, "Returned order should be nil")
	})

	t.Run("Failure - Items Query Error", func(t *testing.T) {
		dbErr := errors.New("DB error querying items")
		// Mock order query (success)
		orderRows := sqlmock.NewRows([]string{"customer_id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrder.CustomerID, expectedOrder.Status, expectedOrder.TotalAmount, expectedOrder.PaymentStatus, expectedOrder.PaymentIntentID, expectedAddrJSON, expectedOrder.CreatedAt, expectedOrder.UpdatedAt)
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnRows(orderRows)

		// Mock items query (failure)
		mock.ExpectQuery(expectedItemsQuerySQL).WithArgs(orderID).WillReturnError(dbErr)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		require.Error(t, err, "GetOrderByID should fail when items query fails")
		assert.ErrorContains(t, err, "failed to get the order items", "Error message should indicate item query failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Nil(t, order, "Returned order should be nil")
	})

	t.Run("Failure - Item Scan Error", func(t *testing.T) {
		// Mock order query (success)
		orderRows := sqlmock.NewRows([]string{"customer_id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrder.CustomerID, expectedOrder.Status, expectedOrder.TotalAmount, expectedOrder.PaymentStatus, expectedOrder.PaymentIntentID, expectedAddrJSON, expectedOrder.CreatedAt, expectedOrder.UpdatedAt)
		mock.ExpectQuery(expectedOrderQuerySQL).WithArgs(orderID).WillReturnRows(orderRows)

		// Mock items query with incorrect columns
		itemRows := sqlmock.NewRows([]string{"id", "product_id"}).AddRow(itemID1, "only_two_item_columns")
		mock.ExpectQuery(expectedItemsQuerySQL).WithArgs(orderID).WillReturnRows(itemRows)

		// Act
		order, err := repo.GetOrderByID(ctx, orderID)

		// Assert
		require.Error(t, err, "GetOrderByID should fail on item scan error")
		assert.ErrorContains(t, err, "failed to scan order item", "Error message should indicate item scan failure")
		assert.ErrorContains(t, err, "Scan", "Error should be related to scanning")
		assert.Nil(t, order, "Returned order should be nil")
	})
}

func TestListOrdersByCustomer(t *testing.T) {
	repo, mock := setupOrderRepoTest(t)
	ctx := t.Context()

	customerID := uuid.New()
	orderID1, orderID2 := uuid.New(), uuid.New()
	itemID1, itemID2 := uuid.New(), uuid.New()
	page, size := 1, 10
	offset := (page - 1) * size
	now := time.Now()

	addr1 := &models.Address{Street: "List St 1", City: "Listville", State: "LS", PostalCode: "11111", Country: "US"}
	addr2 := &models.Address{Street: "List St 2", City: "Listville", State: "LS", PostalCode: "22222", Country: "US"}
	addr1JSON, _ := json.Marshal(addr1)
	addr2JSON, _ := json.Marshal(addr2)

	expectedOrders := []models.Order{
		{
			ID: orderID1, CustomerID: customerID, Status: models.OrderStatusDelivered, TotalAmount: 50.0, PaymentStatus: models.PaymentStatusSucceeded, ShippingAddress: addr1, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour),
			Items: []models.OrderItem{{ID: itemID1, OrderID: orderID1, ProductID: uuid.New(), Quantity: 1, UnitPrice: 50.0, CreatedAt: now.Add(-2 * time.Hour)}},
		},
		{
			ID: orderID2, CustomerID: customerID, Status: models.OrderStatusShipping, TotalAmount: 100.0, PaymentStatus: models.PaymentStatusSucceeded, ShippingAddress: addr2, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour),
			Items: []models.OrderItem{{ID: itemID2, OrderID: orderID2, ProductID: uuid.New(), Quantity: 2, UnitPrice: 50.0, CreatedAt: now.Add(-3 * time.Hour)}},
		},
	}
	totalOrders := len(expectedOrders)

	expectedCountSQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM orders WHERE customer_id = $1`)
	expectedListOrdersSQL := regexp.QuoteMeta(`
        SELECT id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
        FROM orders
        WHERE customer_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `)
	expectedListItemsSQL := regexp.QuoteMeta(`
        SELECT id, product_id, quantity, unit_price, created_at
        FROM order_items
        WHERE order_id = $1
    `)

	t.Run("Success - Multiple Orders", func(t *testing.T) {
		// Mock count query
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalOrders))

		// Mock list orders query
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrders[0].ID, expectedOrders[0].Status, expectedOrders[0].TotalAmount, expectedOrders[0].PaymentStatus, expectedOrders[0].PaymentIntentID, addr1JSON, expectedOrders[0].CreatedAt, expectedOrders[0].UpdatedAt).
			AddRow(expectedOrders[1].ID, expectedOrders[1].Status, expectedOrders[1].TotalAmount, expectedOrders[1].PaymentStatus, expectedOrders[1].PaymentIntentID, addr2JSON, expectedOrders[1].CreatedAt, expectedOrders[1].UpdatedAt)
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Mock items query for order 1
		itemRows1 := sqlmock.NewRows([]string{"id", "product_id", "quantity", "unit_price", "created_at"}).
			AddRow(expectedOrders[0].Items[0].ID, expectedOrders[0].Items[0].ProductID, expectedOrders[0].Items[0].Quantity, expectedOrders[0].Items[0].UnitPrice, expectedOrders[0].Items[0].CreatedAt)
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(expectedOrders[0].ID).WillReturnRows(itemRows1)

		// Mock items query for order 2
		itemRows2 := sqlmock.NewRows([]string{"id", "product_id", "quantity", "unit_price", "created_at"}).
			AddRow(expectedOrders[1].Items[0].ID, expectedOrders[1].Items[0].ProductID, expectedOrders[1].Items[0].Quantity, expectedOrders[1].Items[0].UnitPrice, expectedOrders[1].Items[0].CreatedAt)
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(expectedOrders[1].ID).WillReturnRows(itemRows2)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		assert.NoError(t, err, "ListOrdersByCustomer should succeed")
		assert.Equal(t, totalOrders, total, "Total count should match")
		assert.Equal(t, expectedOrders, orders, "Returned orders should match expected")
	})

	t.Run("Success - No Orders", func(t *testing.T) {
		// Mock count query
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// Mock list orders query (returns no rows)
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"})
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// No item queries expected

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		assert.NoError(t, err, "ListOrdersByCustomer should succeed even with no orders")
		assert.Equal(t, 0, total, "Total count should be 0")
		assert.Empty(t, orders, "Returned orders slice should be empty")
	})

	t.Run("Failure - Count Query Error", func(t *testing.T) {
		dbErr := errors.New("count query failed")
		// Mock count query failure
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnError(dbErr)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on count query error")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - List Orders Query Error", func(t *testing.T) {
		dbErr := errors.New("list orders query failed")
		// Mock count query (success)
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalOrders))

		// Mock list orders query (failure)
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnError(dbErr)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on list orders query error")
		assert.ErrorContains(t, err, "failed to list orders", "Error message should indicate failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - Order Scan Error", func(t *testing.T) {
		// Mock count query (success)
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalOrders))

		// Mock list orders query with bad data
		orderRows := sqlmock.NewRows([]string{"id", "status"}).AddRow(orderID1, "only_two_columns")
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on order scan error")
		assert.ErrorContains(t, err, "failed to scan order row", "Error message should indicate scan failure")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - Address Unmarshal Error", func(t *testing.T) {
		// Mock count query (success)
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock list orders query with invalid JSON address
		invalidJSON := []byte(`{"invalid`)
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrders[0].ID, expectedOrders[0].Status, expectedOrders[0].TotalAmount, expectedOrders[0].PaymentStatus, expectedOrders[0].PaymentIntentID, invalidJSON, expectedOrders[0].CreatedAt, expectedOrders[0].UpdatedAt)
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on address unmarshal error")
		assert.ErrorContains(t, err, "failed to unmarshal shipping address", "Error message should indicate unmarshal failure")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - Item Query Error", func(t *testing.T) {
		dbErr := errors.New("item query failed")
		// Mock count query (success)
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock list orders query (success)
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrders[0].ID, expectedOrders[0].Status, expectedOrders[0].TotalAmount, expectedOrders[0].PaymentStatus, expectedOrders[0].PaymentIntentID, addr1JSON, expectedOrders[0].CreatedAt, expectedOrders[0].UpdatedAt)
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Mock items query for order 1 (failure)
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(expectedOrders[0].ID).WillReturnError(dbErr)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on item query error")
		assert.ErrorContains(t, err, "failed to get the orders", "Error message should indicate item query failure") // Note: Error message could be more specific
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - Item Scan Error", func(t *testing.T) {
		// Mock count query (success)
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock list orders query (success)
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrders[0].ID, expectedOrders[0].Status, expectedOrders[0].TotalAmount, expectedOrders[0].PaymentStatus, expectedOrders[0].PaymentIntentID, addr1JSON, expectedOrders[0].CreatedAt, expectedOrders[0].UpdatedAt)
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Mock items query for order 1 (scan error)
		itemRows1 := sqlmock.NewRows([]string{"id", "product_id"}).AddRow(itemID1, "bad_data")
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(expectedOrders[0].ID).WillReturnRows(itemRows1)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail on item scan error")
		assert.ErrorContains(t, err, "failed to scan order items", "Error message should indicate item scan failure")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})

	t.Run("Failure - Rows Error After Loop", func(t *testing.T) {
		rowsErr := errors.New("rows iteration error")
		// Mock count query
		mock.ExpectQuery(expectedCountSQL).WithArgs(customerID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Mock list orders query, simulate error after reading rows
		orderRows := sqlmock.NewRows([]string{"id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(expectedOrders[0].ID, expectedOrders[0].Status, expectedOrders[0].TotalAmount, expectedOrders[0].PaymentStatus, expectedOrders[0].PaymentIntentID, addr1JSON, expectedOrders[0].CreatedAt, expectedOrders[0].UpdatedAt).
			CloseError(rowsErr) // Simulate error on rows.Err() or rows.Close()
		mock.ExpectQuery(expectedListOrdersSQL).WithArgs(customerID, size, offset).WillReturnRows(orderRows)

		// Mock items query for order 1 (will likely run before CloseError is checked)
		itemRows1 := sqlmock.NewRows([]string{"id", "product_id", "quantity", "unit_price", "created_at"}).
			AddRow(expectedOrders[0].Items[0].ID, expectedOrders[0].Items[0].ProductID, expectedOrders[0].Items[0].Quantity, expectedOrders[0].Items[0].UnitPrice, expectedOrders[0].Items[0].CreatedAt)
		mock.ExpectQuery(expectedListItemsSQL).WithArgs(expectedOrders[0].ID).WillReturnRows(itemRows1)

		// Act
		orders, total, err := repo.ListOrdersByCustomer(ctx, customerID, page, size)

		// Assert
		require.Error(t, err, "ListOrdersByCustomer should fail if rows.Err() occurs")
		assert.ErrorIs(t, err, rowsErr, "Error should be the rows iteration error")
		assert.Nil(t, orders, "Orders slice should be nil")
		assert.Zero(t, total, "Total should be zero")
	})
}

func TestUpdateOrderStatus(t *testing.T) {
	repo, mock := setupOrderRepoTest(t)
	ctx := t.Context()

	orderID := uuid.New()
	newStatus := models.OrderStatusShipping
	now := time.Now() // For mocking fetched order timestamps

	expectedSQL := regexp.QuoteMeta(`UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`)
	// Assume the implementation fetches the order after update
	expectedFetchSQL := regexp.QuoteMeta(`
        SELECT customer_id, status, total_amount, payment_status, payment_intent_id, shipping_address, created_at, updated_at
        FROM orders
        WHERE id = $1
    `)

	t.Run("Success - Order Status Update", func(t *testing.T) {
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 0 for LastInsertId (not relevant), 1 for RowsAffected

		expectedAddress := &models.Address{Street: "Fetched St", City: "Fetchedville"}
		expectedAddrJSON, _ := json.Marshal(expectedAddress)
		fetchedRows := sqlmock.NewRows([]string{"customer_id", "status", "total_amount", "payment_status", "payment_intent_id", "shipping_address", "created_at", "updated_at"}).
			AddRow(uuid.New(), newStatus, 100.0, models.PaymentStatusPending, "pi_fetch", expectedAddrJSON, now.Add(-time.Hour), now)
		mock.ExpectQuery(expectedFetchSQL).WithArgs(orderID).WillReturnRows(fetchedRows)

		expectedItemsQuerySQL := regexp.QuoteMeta(`SELECT id, product_id, quantity, unit_price, created_at FROM order_items WHERE order_id = $1`)
		mock.ExpectQuery(expectedItemsQuerySQL).WithArgs(orderID).WillReturnRows(sqlmock.NewRows([]string{"id", "product_id", "quantity", "unit_price", "created_at"})) // Assuming no items for simplicity or mock them

		// Act
		order, err := repo.UpdateOrderStatus(ctx, orderID, newStatus)

		// Assert
		assert.NoError(t, err, "UpdateOrderStatus should succeed")
		require.NotNil(t, order, "Order should not be nil on success")
		assert.Equal(t, orderID, order.ID)
		assert.Equal(t, newStatus, order.Status)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		dbErr := errors.New("update failed")
		// Expect the update execution to fail
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), orderID).
			WillReturnError(dbErr)

		// Act
		_, err := repo.UpdateOrderStatus(ctx, orderID, newStatus)

		// Assert
		require.Error(t, err, "UpdateOrderStatus should fail on DB error")
		assert.ErrorContains(t, err, "failed to execute update order status query", "Error message should indicate failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
	})

	t.Run("Failure - Order Not Found", func(t *testing.T) {
		// Expect the update execution, returning 0 rows affected
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		// Act
		_, err := repo.UpdateOrderStatus(ctx, orderID, newStatus)

		// Assert
		require.Error(t, err, "UpdateOrderStatus should fail when order not found")
		assert.ErrorIs(t, err, sql.ErrNoRows, "Error should be sql.ErrNoRows when order not found")
	})

	t.Run("Failure - Rows Affected Error", func(t *testing.T) {
		rowsAffectedErr := errors.New("error getting rows affected")
		// Expect the update execution, return a result that errors on RowsAffected()
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewErrorResult(rowsAffectedErr)) // Simulate error during RowsAffected() call

		// Act
		_, err := repo.UpdateOrderStatus(ctx, orderID, newStatus)

		// Assert
		require.Error(t, err, "UpdateOrderStatus should fail if RowsAffected errors")
		assert.ErrorContains(t, err, "failed checking rows affected for order status update", "Error message should indicate failure")
		assert.ErrorIs(t, err, rowsAffectedErr, "Error should wrap the RowsAffected error")
	})
}

func TestUpdatePaymentStatus(t *testing.T) {
	repo, mock := setupOrderRepoTest(t)
	ctx := t.Context()

	orderID := uuid.New()
	newStatus := models.PaymentStatusSucceeded
	paymentIntentID := "pi_updated_123"

	// Corrected SQL with comma before updated_at
	expectedSQL := regexp.QuoteMeta(`
        UPDATE orders set payment_status = $1, payment_intent_id = $2, updated_at = $3 WHERE id = $4
    `)

	t.Run("Success - Update Payment Status", func(t *testing.T) {
		// Expect the update execution, returning 1 row affected
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, paymentIntentID, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

		// Act
		err := repo.UpdatePaymentStatus(ctx, orderID, newStatus, paymentIntentID)

		// Assert
		assert.NoError(t, err, "UpdatePaymentStatus should succeed")
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		dbErr := errors.New("update payment status failed")
		// Expect the update execution to fail
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, paymentIntentID, sqlmock.AnyArg(), orderID).
			WillReturnError(dbErr)

		// Act
		err := repo.UpdatePaymentStatus(ctx, orderID, newStatus, paymentIntentID)

		// Assert
		require.Error(t, err, "UpdatePaymentStatus should fail on DB error")
		assert.ErrorContains(t, err, "failed to execute update payment status query", "Error message should indicate failure")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
	})

	t.Run("Failure - Order Not Found", func(t *testing.T) {
		// Expect the update execution, returning 0 rows affected
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, paymentIntentID, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

		// Act
		err := repo.UpdatePaymentStatus(ctx, orderID, newStatus, paymentIntentID)

		// Assert
		require.Error(t, err, "UpdatePaymentStatus should fail when order not found")
		assert.ErrorIs(t, err, sql.ErrNoRows, "Error should be sql.ErrNoRows when order not found")
	})

	t.Run("Failure - Rows Affected Error", func(t *testing.T) {
		rowsAffectedErr := errors.New("error getting rows affected")
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, paymentIntentID, sqlmock.AnyArg(), orderID).
			WillReturnResult(sqlmock.NewErrorResult(rowsAffectedErr))

		// Act
		err := repo.UpdatePaymentStatus(ctx, orderID, newStatus, paymentIntentID)

		// Assert
		require.Error(t, err, "UpdatePaymentStatus should fail if RowsAffected errors")
		assert.ErrorContains(t, err, "failed checking rows affected for payment status update", "Error message should indicate failure")
		assert.ErrorIs(t, err, rowsAffectedErr, "Error should wrap the RowsAffected error")
	})
}
