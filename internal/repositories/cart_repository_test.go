package repository_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
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

func setupCartRepoTest(t *testing.T) (repository.CartRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err, "Failed to create sqlmock")

	t.Cleanup(func() {
		db.Close()
	})

	repo := repository.NewCartRepo(db)
	require.NotNil(t, repo, "NewCartRepo should return a non-nil repository")

	return repo, mock
}

func TestNewCartRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewCartRepo(db)
	assert.NotNil(t, repo, "NewCartRepo should return a non-nil repository")
}

func TestCartRepository(t *testing.T) {
	repo, mock := setupCartRepoTest(t)
	ctx := t.Context()

	t.Run("Create Cart", func(t *testing.T) {
		userID := uuid.New()
		cartID := uuid.New()
		now := time.Now()
		cart := &models.Cart{
			ID:     cartID,
			UserID: userID,
			Items:  make(map[string]models.CartItem),
			Total:  0,
		}
		// Expected JSON for empty items map
		expectedItemsJSON, err := json.Marshal(cart.Items)
		require.NoError(t, err, "Failed to marshal empty items map for test setup")

		expectedSQL := regexp.QuoteMeta(`
        INSERT INTO carts (id, user_id, items, created_at, updated_at)
        VALUES($1, $2, $3, NOW(), NOW())
        RETURNING id, created_at, updated_at
    `)

		t.Run("Success", func(t *testing.T) {
			// Arrange
			mock.ExpectQuery(expectedSQL).
				WithArgs(cart.ID, cart.UserID, expectedItemsJSON).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(cartID, now, now))

			// Act
			err := repo.CreateCart(ctx, cart)

			// Assert
			require.NoError(t, err, "CreateCart should not return an error on success")
			assert.Equal(t, cartID, cart.ID, "Cart ID should remain the same")
			assert.WithinDuration(t, now, cart.CreatedAt, time.Second, "Cart CreatedAt should be updated")
			assert.WithinDuration(t, now, cart.UpdatedAt, time.Second, "Cart UpdatedAt should be updated")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Marshal Error", func(t *testing.T) {
			// Arrange
			invalidCart := &models.Cart{
				ID:     uuid.New(),
				UserID: uuid.New(),
				Items: map[string]models.CartItem{
					"invalid":       {ProductID: uuid.New(), Quantity: 1, UnitPrice: 10.0, TotalPrice: 10.0},
					"unmarshalable": {ProductID: uuid.New(), Quantity: 1, UnitPrice: math.Inf(1), TotalPrice: 10.0},
				},
			}

			// Act
			err := repo.CreateCart(ctx, invalidCart)

			// Assert
			require.Error(t, err, "CreateCart should return an error on marshal failure")
			assert.ErrorContains(t, err, "failed to marshal cart items", "Error message should indicate marshal failure")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Database Error", func(t *testing.T) {
			// Arrange
			dbError := errors.New("database insertion error")
			mock.ExpectQuery(expectedSQL).
				WithArgs(cart.ID, cart.UserID, expectedItemsJSON).
				WillReturnError(dbError)

			// Act
			err := repo.CreateCart(ctx, cart)

			// Assert
			require.Error(t, err, "CreateCart should return an error on DB failure")
			assert.Equal(t, dbError, err, "Returned error should match the expected database error")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})

	t.Run("GetCartByCustomerID", func(t *testing.T) {
		customerID := uuid.New()
		cartID := uuid.New()
		productID := uuid.New()
		now := time.Now()
		expectedItems := map[string]models.CartItem{
			productID.String(): {ProductID: productID, Quantity: 2, UnitPrice: 10.50, TotalPrice: 21.00},
		}
		expectedItemsJSON, err := json.Marshal(expectedItems)
		require.NoError(t, err, "Failed to marshal items for test setup")

		expectedSQL := regexp.QuoteMeta(`
        SELECT id, user_id, items, created_at, updated_at
        FROM carts
        WHERE user_id = $1
    `)

		t.Run("Success", func(t *testing.T) {
			// Arrange
			rows := sqlmock.NewRows([]string{"id", "user_id", "items", "created_at", "updated_at"}).
				AddRow(cartID, customerID, expectedItemsJSON, now, now)
			mock.ExpectQuery(expectedSQL).
				WithArgs(customerID).
				WillReturnRows(rows)

			// Act
			cart, err := repo.GetCartByCustomerID(ctx, customerID)

			// Assert
			require.NoError(t, err, "GetCartByCustomerID should not return an error when cart is found")
			require.NotNil(t, cart, "Returned cart should not be nil")
			assert.Equal(t, cartID, cart.ID)
			assert.Equal(t, customerID, cart.UserID)
			assert.Equal(t, expectedItems, cart.Items)
			assert.WithinDuration(t, now, cart.CreatedAt, time.Second)
			assert.WithinDuration(t, now, cart.UpdatedAt, time.Second)
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Not Found", func(t *testing.T) {
			// Arrange
			mock.ExpectQuery(expectedSQL).
				WithArgs(customerID).
				WillReturnError(sql.ErrNoRows)

			// Act
			cart, err := repo.GetCartByCustomerID(ctx, customerID)

			// Assert
			require.Error(t, err, "GetCartByCustomerID should return an error when cart is not found")
			assert.ErrorIs(t, err, sql.ErrNoRows, "Error should be sql.ErrNoRows")
			assert.Nil(t, cart, "Returned cart should be nil")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Database Error", func(t *testing.T) {
			// Arrange
			dbError := errors.New("database query error")
			mock.ExpectQuery(expectedSQL).
				WithArgs(customerID).
				WillReturnError(dbError)

			// Act
			cart, err := repo.GetCartByCustomerID(ctx, customerID)

			// Assert
			require.Error(t, err, "GetCartByCustomerID should return an error on DB failure")
			assert.Equal(t, dbError, err, "Returned error should match the expected database error")
			assert.Nil(t, cart, "Returned cart should be nil")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Unmarshal Error", func(t *testing.T) {
			// Arrange
			invalidJSON := []byte(`{"invalid"`)
			rows := sqlmock.NewRows([]string{"id", "user_id", "items", "created_at", "updated_at"}).
				AddRow(cartID, customerID, invalidJSON, now, now)
			mock.ExpectQuery(expectedSQL).
				WithArgs(customerID).
				WillReturnRows(rows)

			// Act
			cart, err := repo.GetCartByCustomerID(ctx, customerID)

			// Assert
			require.Error(t, err, "GetCartByCustomerID should return an error on unmarshal failure")
			assert.ErrorContains(t, err, "failed to unmarshal cart items", "Error message should indicate unmarshal failure")

			var syntaxError *json.SyntaxError

			assert.ErrorAs(t, err, &syntaxError, "Error should be a json.SyntaxError")
			assert.Nil(t, cart, "Returned cart should be nil")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})

	t.Run("UpdateCart", func(t *testing.T) {
		cartID := uuid.New()
		userID := uuid.New()
		productID := uuid.New()
		updatedItems := map[string]models.CartItem{
			productID.String(): {ProductID: productID, Quantity: 3, UnitPrice: 10.0, TotalPrice: 30.0},
		}
		cartToUpdate := &models.Cart{
			ID:     cartID,
			UserID: userID,
			Items:  updatedItems,
			Total:  30.0,
		}
		expectedItemsJSON, err := json.Marshal(updatedItems)
		require.NoError(t, err, "Failed to marshal updated items for test setup")

		expectedSQL := regexp.QuoteMeta(`
        UPDATE carts
        SET items = $1, total = $2, updated_at = $3
        WHERE id = $4
    `)

		t.Run("Success", func(t *testing.T) {
			// Arrange
			mock.ExpectExec(expectedSQL).
				WithArgs(expectedItemsJSON, cartToUpdate.Total, sqlmock.AnyArg(), cartToUpdate.ID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			// Act
			err := repo.UpdateCart(ctx, cartToUpdate)

			// Assert
			require.NoError(t, err, "UpdateCart should not return an error on success")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Marshal Error", func(t *testing.T) {
			// Arrange
			invalidCart := &models.Cart{
				ID:     cartID,
				UserID: userID,
				Items: map[string]models.CartItem{
					"unmarshalable": {ProductID: uuid.New(), Quantity: 1, UnitPrice: math.Inf(1), TotalPrice: 10.0},
				},
				Total: 10.0,
			}

			// Act
			err := repo.UpdateCart(ctx, invalidCart)

			// Assert
			require.Error(t, err, "UpdateCart should return an error on marshal failure")
			assert.ErrorContains(t, err, "failed to marshal cart items", "Error message should indicate marshal failure")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Database Error", func(t *testing.T) {
			// Arrange
			dbError := errors.New("database update error")
			mock.ExpectExec(expectedSQL).
				WithArgs(expectedItemsJSON, cartToUpdate.Total, sqlmock.AnyArg(), cartToUpdate.ID).
				WillReturnError(dbError)

			// Act
			err := repo.UpdateCart(ctx, cartToUpdate)

			// Assert
			require.Error(t, err, "UpdateCart should return an error on DB failure")
			assert.ErrorIs(t, err, dbError, "Returned error should wrap the expected database error")
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Database Error No Rows Affected", func(t *testing.T) {
			// Arrange
			mock.ExpectExec(expectedSQL).
				WithArgs(expectedItemsJSON, cartToUpdate.Total, sqlmock.AnyArg(), cartToUpdate.ID).
				WillReturnResult(sqlmock.NewResult(0, 0))

			// Act
			err := repo.UpdateCart(ctx, cartToUpdate)

			// Assert
			require.Error(t, err, "UpdateCart should return an error if no rows were affected")
			assert.ErrorIs(t, err, sql.ErrNoRows)
			require.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})
}
