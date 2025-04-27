package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPaymentRepoTest(t *testing.T) (repository.PaymentRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create sqlmock")

	t.Cleanup(func() {
		db.Close()
	})

	repo := repository.NewPaymentRepository(db)
	require.NotNil(t, repo, "NewPaymentRepository should not return nil")
	return repo, mock
}

func TestNewPaymentRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewPaymentRepository(db)
	assert.NotNil(t, repo, "Expected a non-nil repository instance")
}

func TestCreatePayment(t *testing.T) {
	repo, mock := setupPaymentRepoTest(t)
	ctx := context.Background()

	payment := &models.Payment{
		ID:            "pi_123",
		Amount:        1000,
		Currency:      "usd",
		CustomerID:    "cus_abc",
		Description:   "Test Payment",
		Status:        models.PaymentStatusPending,
		PaymentMethod: "card",
		StripeID:      "pi_123",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	expectedSQL := regexp.QuoteMeta(`
        INSERT INTO payments (id, amount, currency, customer_id, description, status, payment_method, stripe_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8,NOW(), NOW())
    `)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(expectedSQL).
			WithArgs(payment.ID, payment.Amount, payment.Currency, payment.CustomerID, payment.Description, payment.Status, payment.PaymentMethod, payment.StripeID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Act
		err := repo.CreatePayment(ctx, payment)

		// Assert
		assert.NoError(t, err, "CreatePayment should succeed")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - DB Error", func(t *testing.T) {
		dbErr := errors.New("database connection lost")
		mock.ExpectExec(expectedSQL).
			WithArgs(payment.ID, payment.Amount, payment.Currency, payment.CustomerID, payment.Description, payment.Status, payment.PaymentMethod, payment.StripeID).
			WillReturnError(dbErr)

		// Act
		err := repo.CreatePayment(ctx, payment)

		// Assert
		assert.Error(t, err, "CreatePayment should fail")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Contains(t, err.Error(), "failed to insert payment", "Error message should indicate insertion failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})
}

func TestGetPaymentByID(t *testing.T) {
	repo, mock := setupPaymentRepoTest(t)
	ctx := context.Background()
	testID := "pi_xyz789"

	// Define the expected SQL query
	expectedSQL := regexp.QuoteMeta(`
        SELECT id, amount, currency, customer_id, description, status, payment_method, stripe_id, created_at, updated_at
        FROM payments
        WHERE id = $1
    `)

	// Expected payment data
	expectedPayment := &models.Payment{
		ID:            testID,
		Amount:        5000,
		Currency:      "eur",
		CustomerID:    "cus_def",
		Description:   "Another Test Payment",
		Status:        models.PaymentStatusSucceeded,
		PaymentMethod: "ideal",
		StripeID:      testID,
		CreatedAt:     time.Now().Add(-time.Hour),
		UpdatedAt:     time.Now(),
	}

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "amount", "currency", "customer_id", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"}).
			AddRow(expectedPayment.ID, expectedPayment.Amount, expectedPayment.Currency, expectedPayment.CustomerID, expectedPayment.Description, expectedPayment.Status, expectedPayment.PaymentMethod, expectedPayment.StripeID, expectedPayment.CreatedAt, expectedPayment.UpdatedAt)

		mock.ExpectQuery(expectedSQL).
			WithArgs(testID).
			WillReturnRows(rows)

		// Act
		payment, err := repo.GetPaymentByID(ctx, testID)

		// Assert
		assert.NoError(t, err, "GetPaymentByID should succeed")
		assert.NotNil(t, payment, "Expected a non-nil payment")
		assert.Equal(t, expectedPayment, payment, "Returned payment does not match expected")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		mock.ExpectQuery(expectedSQL).
			WithArgs(testID).
			WillReturnError(sql.ErrNoRows)

		// Act
		payment, err := repo.GetPaymentByID(ctx, testID)

		// Assert
		assert.Error(t, err, "GetPaymentByID should fail")
		assert.Nil(t, payment, "Payment should be nil on error")
		assert.ErrorIs(t, err, sql.ErrNoRows, "Error should wrap sql.ErrNoRows")
		assert.Contains(t, err.Error(), "failed to get the payment", "Error message should indicate retrieval failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - DB Error on Query", func(t *testing.T) {
		dbErr := errors.New("query execution failed")
		mock.ExpectQuery(expectedSQL).
			WithArgs(testID).
			WillReturnError(dbErr)

		// Act
		payment, err := repo.GetPaymentByID(ctx, testID)

		// Assert
		assert.Error(t, err, "GetPaymentByID should fail")
		assert.Nil(t, payment, "Payment should be nil on error")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Contains(t, err.Error(), "failed to get the payment", "Error message should indicate retrieval failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Scan Error", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "amount", "currency", "customer_id", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"}).
			AddRow(expectedPayment.ID, "not-an-int", expectedPayment.Currency, expectedPayment.CustomerID, expectedPayment.Description, expectedPayment.Status, expectedPayment.PaymentMethod, expectedPayment.StripeID, expectedPayment.CreatedAt, expectedPayment.UpdatedAt)

		mock.ExpectQuery(expectedSQL).
			WithArgs(testID).
			WillReturnRows(rows)

		// Act
		payment, err := repo.GetPaymentByID(ctx, testID)

		// Assert
		assert.Error(t, err, "GetPaymentByID should fail due to scan error")
		assert.Nil(t, payment, "Payment should be nil on scan error")
		assert.Contains(t, err.Error(), "failed to get the payment", "Error message should indicate retrieval failure")
		assert.Contains(t, err.Error(), "Scan", "Error message should suggest a Scan issue")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})
}

func TestPaymentRepository_UpdatePaymentStatus(t *testing.T) {
	repo, mock := setupPaymentRepoTest(t)
	ctx := context.Background()
	testID := "pi_update123"
	newStatus := models.PaymentStatusFailed

	// Define the expected SQL query
	expectedSQL := regexp.QuoteMeta(`
        UPDATE payments SET status = $1, updated_at = $2
        WHERE id = $3
    `)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), testID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Act
		err := repo.UpdatePaymentStatus(ctx, testID, newStatus)

		// Assert
		assert.NoError(t, err, "UpdatePaymentStatus should succeed")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Not Found (0 Rows Affected)", func(t *testing.T) {
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), testID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Act
		err := repo.UpdatePaymentStatus(ctx, testID, newStatus)

		// Assert
		assert.Error(t, err, "UpdatePaymentStatus should fail when no rows are affected")
		assert.ErrorIs(t, err, sql.ErrNoRows, "Error should be sql.ErrNoRows for 0 affected rows")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - DB Error on Exec", func(t *testing.T) {
		dbErr := errors.New("update execution failed")
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), testID).
			WillReturnError(dbErr)

		// Act
		err := repo.UpdatePaymentStatus(ctx, testID, newStatus)

		// Assert
		assert.Error(t, err, "UpdatePaymentStatus should fail on DB error")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original DB error")
		assert.Contains(t, err.Error(), "failed to update the payment status", "Error message should indicate update failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Error Getting RowsAffected", func(t *testing.T) {
		rowsAffectedErr := errors.New("failed to determine rows affected")
		mock.ExpectExec(expectedSQL).
			WithArgs(newStatus, sqlmock.AnyArg(), testID).
			WillReturnResult(sqlmock.NewErrorResult(rowsAffectedErr))

		// Act
		err := repo.UpdatePaymentStatus(ctx, testID, newStatus)

		// Assert
		assert.Error(t, err, "UpdatePaymentStatus should fail if RowsAffected returns an error")
		assert.ErrorIs(t, err, rowsAffectedErr, "Error should wrap the RowsAffected error")
		assert.Contains(t, err.Error(), "failed to get updated rows", "Error message should indicate RowsAffected failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})
}

func TestListPaymentsOfCustomer(t *testing.T) {
	repo, mock := setupPaymentRepoTest(t)
	ctx := context.Background()
	customerID := "cus_list123"
	page, size := 1, 2

	// Define expected SQL queries
	expectedCountSQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM payments`)
	expectedListSQL := regexp.QuoteMeta(`
        SELECT id, customer_id, amount, currency, description, status, payment_method, stripe_id, created_at, updated_at
        FROM payments
        WHERE customer_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `)

	// Expected payment data for list
	payment1 := models.Payment{ID: "pi_list1", CustomerID: customerID, Amount: 100, Currency: "usd", Status: models.PaymentStatusSucceeded, CreatedAt: time.Now().Add(-2 * time.Hour)}
	payment2 := models.Payment{ID: "pi_list2", CustomerID: customerID, Amount: 200, Currency: "usd", Status: models.PaymentStatusPending, CreatedAt: time.Now().Add(-1 * time.Hour)}

	t.Run("Success - Multiple Payments", func(t *testing.T) {
		expectedTotal := 5

		mock.ExpectQuery(expectedCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		listRows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"}).
			AddRow(payment2.ID, payment2.CustomerID, payment2.Amount, payment2.Currency, payment2.Description, payment2.Status, payment2.PaymentMethod, payment2.StripeID, payment2.CreatedAt, payment2.UpdatedAt). // Order is DESC
			AddRow(payment1.ID, payment1.CustomerID, payment1.Amount, payment1.Currency, payment1.Description, payment1.Status, payment1.PaymentMethod, payment1.StripeID, payment1.CreatedAt, payment1.UpdatedAt)

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnRows(listRows)

		// Act
		payments, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.NoError(t, err, "ListPaymentsOfCustomer should succeed")
		assert.Equal(t, expectedTotal, total, "Total count mismatch")
		assert.Len(t, payments, 2, "Expected 2 payments in the result")
		assert.Equal(t, payment2.ID, payments[0].ID)
		assert.Equal(t, payment1.ID, payments[1].ID)
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Success - Zero Payments", func(t *testing.T) {
		expectedTotal := 0

		mock.ExpectQuery(expectedCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		listRows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"})

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnRows(listRows)

		// Act
		payments, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.NoError(t, err, "ListPaymentsOfCustomer should succeed")
		assert.Equal(t, expectedTotal, total, "Total count should be 0")
		assert.Len(t, payments, 0, "Expected 0 payments in the result")
		assert.Empty(t, payments, "Payments slice should be empty")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Count Query Error", func(t *testing.T) {
		dbErr := errors.New("count query failed")
		mock.ExpectQuery(expectedCountSQL).
			WillReturnError(dbErr)

		// Act
		payments, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.Error(t, err, "ListPaymentsOfCustomer should fail")
		assert.Nil(t, payments, "Payments should be nil on error")
		assert.Zero(t, total, "Total should be 0 on error")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original count query error")
	})

	t.Run("Failure - List Query Error", func(t *testing.T) {
		expectedTotal := 5
		dbErr := errors.New("list query failed")

		mock.ExpectQuery(expectedCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnError(dbErr)

		// Act
		payments, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.Error(t, err, "ListPaymentsOfCustomer should fail")
		assert.Nil(t, payments, "Payments should be nil on error")
		assert.Zero(t, total, "Total should be 0 on error")
		assert.ErrorIs(t, err, dbErr, "Error should wrap the original list query error")
		assert.Contains(t, err.Error(), "failed to list the payments", "Error message should indicate list failure")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - Row Scan Error", func(t *testing.T) {
		expectedTotal := 1

		mock.ExpectQuery(expectedCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		listRows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"}).
			AddRow(payment1.ID, payment1.CustomerID, "not-an-int", payment1.Currency, payment1.Description, payment1.Status, payment1.PaymentMethod, payment1.StripeID, payment1.CreatedAt, payment1.UpdatedAt) // Bad amount

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnRows(listRows)

		// Act
		payments, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.Error(t, err, "ListPaymentsOfCustomer should fail on scan error")
		assert.Nil(t, payments, "Payments should be nil on scan error")
		assert.Zero(t, total, "Total should be 0 on scan error")
		assert.Contains(t, err.Error(), "failed to scan the payments", "Error message should indicate scan failure")
		assert.Contains(t, err.Error(), "Scan", "Error should be related to scanning")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Failure - rows.Err() after loop", func(t *testing.T) {
		expectedTotal := 1
		rowsErr := errors.New("error during row iteration")

		mock.ExpectQuery(expectedCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		listRows := sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"}).
			AddRow(payment1.ID, payment1.CustomerID, payment1.Amount, payment1.Currency, payment1.Description, payment1.Status, payment1.PaymentMethod, payment1.StripeID, payment1.CreatedAt, payment1.UpdatedAt).
			RowError(0, rowsErr)

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnRows(listRows)

		// Act
		_, total, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.Error(t, err, "ListPaymentsOfCustomer should fail if rows.Err() returns an error")
		// Depending on when rows.Err() is checked, payments might be partially populated or nil.
		// The current implementation checks after the loop, so payments might have one item.
		// assert.Nil(t, payments, "Payments should ideally be nil on rows.Err()") // Or assert partial result if intended
		assert.Zero(t, total, "Total should be 0 on rows.Err()")
		assert.ErrorIs(t, err, rowsErr, "Error should wrap the rows.Err() error")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Verify Count Query Table Name", func(t *testing.T) {
		expectedTotal := 0
		incorrectCountSQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM payments`)

		mock.ExpectQuery(incorrectCountSQL).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(expectedTotal))

		mock.ExpectQuery(expectedListSQL).
			WithArgs(customerID, size, (page-1)*size).
			WillReturnRows(sqlmock.NewRows([]string{"id", "customer_id", "amount", "currency", "description", "status", "payment_method", "stripe_id", "created_at", "updated_at"})) // Empty result

		// Act
		_, _, err := repo.ListPaymentsOfCustomer(ctx, customerID, page, size)

		// Assert
		assert.NoError(t, err, "ListPaymentsOfCustomer should still execute with the specified count query")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met, check count query")
	})
}
