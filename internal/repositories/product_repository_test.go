package repository_test

import (
	"database/sql"
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

func TestNewProductRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewProductRepo(db)
	assert.NotNil(t, repo, "NewProductRepo should return a non-nil repository")
}

func TestProductRepository(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewProductRepo(db)
	ctx := t.Context()

	t.Run("CreateProduct", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Arrange
			product := &models.Product{
				CategoryID:    uuid.New(),
				Name:          "Test Product",
				Description:   "Test Description",
				Price:         99.99,
				StockQuantity: 100,
				SKU:           "TESTSKU123",
				Status:        "active",
			}
			now := time.Now()
			newID := uuid.New()

			expectedSQL := regexp.QuoteMeta(`INSERT INTO products (category_id, name, description, price, stock_quantity, sku, status) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`)

			mock.ExpectQuery(expectedSQL).
				WithArgs(product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.SKU, product.Status).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(newID, now, now))

			// Act
			err := repo.CreateProduct(ctx, product)

			// Assert
			require.NoError(t, err, "CreateProduct should not return an error on success")
			assert.Equal(t, newID, product.ID, "Product ID should be updated")
			assert.WithinDuration(t, now, product.CreatedAt, time.Second, "Product CreatedAt should be updated")
			assert.WithinDuration(t, now, product.UpdatedAt, time.Second, "Product UpdatedAt should be updated")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("Error", func(t *testing.T) {
			// Arrange
			product := &models.Product{
				CategoryID:    uuid.New(),
				Name:          "Error Product",
				Description:   "Error Description",
				Price:         10.00,
				StockQuantity: 5,
				SKU:           "ERRORSKU",
				Status:        "active",
			}
			dbError := errors.New("database insertion error")

			expectedSQL := regexp.QuoteMeta(`INSERT INTO products (category_id, name, description, price, stock_quantity, sku, status) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`)

			mock.ExpectQuery(expectedSQL).
				WithArgs(product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.SKU, product.Status).
				WillReturnError(dbError)

			// Act
			err := repo.CreateProduct(ctx, product)

			// Assert
			require.Error(t, err, "CreateProduct should return an error on database failure")
			assert.ErrorIs(t, err, dbError, "Returned error should be the database error")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("GetProductByID", func(t *testing.T) {
		productID := uuid.New()
		categoryID := uuid.New()
		now := time.Now()

		expectedSQL := regexp.QuoteMeta(`
        SELECT p.id, p.category_id, p.name, p.description, p.price,
               p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
               c.id, c.name, c.description
        FROM products p
        LEFT JOIN categories c ON p.category_id = c.id
        WHERE p.id = $1`)

		t.Run("Success", func(t *testing.T) {
			// Arrange
			expectedProduct := &models.Product{
				ID:            productID,
				CategoryID:    categoryID,
				Name:          "Found Product",
				Description:   "Found Description",
				Price:         50.00,
				StockQuantity: 20,
				SKU:           "FOUNDSKU",
				Status:        "active",
				CreatedAt:     now.Add(-time.Hour),
				UpdatedAt:     now,
				Category: &models.Category{
					ID:          categoryID,
					Name:        "Found Category",
					Description: "Category Description",
				},
			}

			rows := sqlmock.NewRows([]string{
				"p.id", "p.category_id", "p.name", "p.description", "p.price",
				"p.stock_quantity", "p.sku", "p.status", "p.created_at", "p.updated_at",
				"c.id", "c.name", "c.description",
			}).AddRow(
				expectedProduct.ID, expectedProduct.CategoryID, expectedProduct.Name, expectedProduct.Description, expectedProduct.Price,
				expectedProduct.StockQuantity, expectedProduct.SKU, expectedProduct.Status, expectedProduct.CreatedAt, expectedProduct.UpdatedAt,
				expectedProduct.Category.ID, expectedProduct.Category.Name, expectedProduct.Category.Description,
			)

			mock.ExpectQuery(expectedSQL).
				WithArgs(productID).
				WillReturnRows(rows)

			// Act
			product, err := repo.GetProductByID(ctx, productID)

			// Assert
			require.NoError(t, err, "GetProductByID should not return an error when product is found")
			assert.Equal(t, expectedProduct, product, "Returned product should match the expected product")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("NotFound", func(t *testing.T) {
			// Arrange
			mock.ExpectQuery(expectedSQL).
				WithArgs(productID).
				WillReturnError(sql.ErrNoRows)

			// Act
			product, err := repo.GetProductByID(ctx, productID)

			// Assert
			require.Error(t, err, "GetProductByID should return an error when product is not found")
			assert.ErrorIs(t, err, sql.ErrNoRows, "Returned error should be sql.ErrNoRows")
			assert.Nil(t, product, "Returned product should be nil on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("ScanError", func(t *testing.T) {
			// Arrange
			// Return rows with incorrect column types to trigger a scan error
			rows := sqlmock.NewRows([]string{"p.id", "p.category_id"}).AddRow("not-a-uuid", "not-a-uuid")

			mock.ExpectQuery(expectedSQL).
				WithArgs(productID).
				WillReturnRows(rows) // Intentionally cause scan error

			// Act
			product, err := repo.GetProductByID(ctx, productID)

			// Assert
			require.Error(t, err, "GetProductByID should return an error on scan failure")
			assert.Contains(t, err.Error(), "Scan", "Error message should indicate a scan issue")
			assert.Nil(t, product, "Returned product should be nil on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("UpdateProduct", func(t *testing.T) {
		productID := uuid.New()
		categoryID := uuid.New()
		now := time.Now()

		expectedSQL := regexp.QuoteMeta(`
        UPDATE products SET category_id = $1, name = $2, description = $3, price = $4, stock_quantity = $5, status = $6, updated_at = NOW()
        WHERE id = $7
        RETURNING updated_at`)

		t.Run("Success", func(t *testing.T) {
			// Arrange
			productToUpdate := &models.Product{
				ID:            productID,
				CategoryID:    categoryID,
				Name:          "Updated Product",
				Description:   "Updated Description",
				Price:         150.00,
				StockQuantity: 15,
				Status:        "inactive",
			}
			updatedAt := now

			mock.ExpectQuery(expectedSQL).
				WithArgs(productToUpdate.CategoryID, productToUpdate.Name, productToUpdate.Description, productToUpdate.Price, productToUpdate.StockQuantity, productToUpdate.Status, productToUpdate.ID).
				WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(updatedAt))

			// Act
			err := repo.UpdateProduct(ctx, productToUpdate)

			// Assert
			require.NoError(t, err, "UpdateProduct should not return an error on success")
			assert.WithinDuration(t, updatedAt, productToUpdate.UpdatedAt, time.Second, "Product UpdatedAt should be updated")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("Error", func(t *testing.T) {
			// Arrange
			productToUpdate := &models.Product{
				ID:            productID,
				CategoryID:    categoryID,
				Name:          "Update Error Product",
				Description:   "Update Error Desc",
				Price:         10.00,
				StockQuantity: 1,
				Status:        "active",
			}
			dbError := errors.New("database update error")

			mock.ExpectQuery(expectedSQL).
				WithArgs(productToUpdate.CategoryID, productToUpdate.Name, productToUpdate.Description, productToUpdate.Price, productToUpdate.StockQuantity, productToUpdate.Status, productToUpdate.ID).
				WillReturnError(dbError)

			// Act
			err := repo.UpdateProduct(ctx, productToUpdate)

			// Assert
			require.Error(t, err, "UpdateProduct should return an error on database failure")
			assert.ErrorIs(t, err, dbError, "Returned error should be the database error")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("NotFound", func(t *testing.T) {
			// Arrange
			productToUpdate := &models.Product{
				ID:            uuid.New(), // Use a different ID that won't be found
				CategoryID:    categoryID,
				Name:          "Not Found Product",
				Description:   "Not Found Desc",
				Price:         5.00,
				StockQuantity: 2,
				Status:        "active",
			}

			mock.ExpectQuery(expectedSQL).
				WithArgs(productToUpdate.CategoryID, productToUpdate.Name, productToUpdate.Description, productToUpdate.Price, productToUpdate.StockQuantity, productToUpdate.Status, productToUpdate.ID).
				WillReturnError(sql.ErrNoRows) // Simulate row not found during update

			// Act
			err := repo.UpdateProduct(ctx, productToUpdate)

			// Assert
			require.Error(t, err, "UpdateProduct should return an error if the product to update is not found")
			assert.ErrorIs(t, err, sql.ErrNoRows, "Returned error should be sql.ErrNoRows")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})

	t.Run("ListProducts", func(t *testing.T) {
		page, size := 1, 2
		offset := (page - 1) * size
		now := time.Now()

		expectedCountSQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM products`)
		expectedListSQL := regexp.QuoteMeta(`
        SELECT p.id, p.category_id, p.name, p.description, p.price,
        p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
        c.id, c.name, c.description
        FROM products p
        LEFT JOIN categories c on p.category_id = c.id
        ORDER BY p.id
        LIMIT $1 OFFSET $2`)

		productCols := []string{
			"p.id", "p.category_id", "p.name", "p.description", "p.price",
			"p.stock_quantity", "p.sku", "p.status", "p.created_at", "p.updated_at",
			"c.id", "c.name", "c.description",
		}

		t.Run("Success_MultipleItems", func(t *testing.T) {
			// Arrange
			total := 5
			catID1, catID2 := uuid.New(), uuid.New()
			prodID1, prodID2 := uuid.New(), uuid.New()

			expectedProducts := []*models.Product{
				{
					ID: prodID1, CategoryID: catID1, Name: "Prod 1", Price: 10, StockQuantity: 1, SKU: "SKU1", Status: "active", CreatedAt: now, UpdatedAt: now,
					Category: &models.Category{ID: catID1, Name: "Cat 1"},
				},
				{
					ID: prodID2, CategoryID: catID2, Name: "Prod 2", Price: 20, StockQuantity: 2, SKU: "SKU2", Status: "active", CreatedAt: now, UpdatedAt: now,
					Category: &models.Category{ID: catID2, Name: "Cat 2"},
				},
			}

			mock.ExpectQuery(expectedCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
			rows := sqlmock.NewRows(productCols).
				AddRow(expectedProducts[0].ID, expectedProducts[0].CategoryID, expectedProducts[0].Name, expectedProducts[0].Description, expectedProducts[0].Price, expectedProducts[0].StockQuantity, expectedProducts[0].SKU, expectedProducts[0].Status, expectedProducts[0].CreatedAt, expectedProducts[0].UpdatedAt, expectedProducts[0].Category.ID, expectedProducts[0].Category.Name, expectedProducts[0].Category.Description).
				AddRow(expectedProducts[1].ID, expectedProducts[1].CategoryID, expectedProducts[1].Name, expectedProducts[1].Description, expectedProducts[1].Price, expectedProducts[1].StockQuantity, expectedProducts[1].SKU, expectedProducts[1].Status, expectedProducts[1].CreatedAt, expectedProducts[1].UpdatedAt, expectedProducts[1].Category.ID, expectedProducts[1].Category.Name, expectedProducts[1].Category.Description)
			mock.ExpectQuery(expectedListSQL).WithArgs(size, offset).WillReturnRows(rows)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.NoError(t, err, "ListProducts should not return an error on success")
			assert.Equal(t, total, count, "Returned total count should match expected")
			assert.Equal(t, expectedProducts, products, "Returned products should match expected")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("Success_NoItems", func(t *testing.T) {
			// Arrange
			total := 0
			mock.ExpectQuery(expectedCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
			rows := sqlmock.NewRows(productCols) // No rows added
			mock.ExpectQuery(expectedListSQL).WithArgs(size, offset).WillReturnRows(rows)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.NoError(t, err, "ListProducts should not return an error when no items exist")
			assert.Equal(t, total, count, "Returned total count should be 0")
			assert.Empty(t, products, "Returned products slice should be empty")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("CountError", func(t *testing.T) {
			// Arrange
			dbError := errors.New("count query failed")
			mock.ExpectQuery(expectedCountSQL).WillReturnError(dbError)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.Error(t, err, "ListProducts should return an error if count query fails")
			assert.ErrorIs(t, err, dbError, "Returned error should be the database error")
			assert.Nil(t, products, "Returned products should be nil on error")
			assert.Zero(t, count, "Returned count should be zero on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("ListQueryError", func(t *testing.T) {
			// Arrange
			total := 5
			dbError := errors.New("list query failed")

			mock.ExpectQuery(expectedCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
			mock.ExpectQuery(expectedListSQL).WithArgs(size, offset).WillReturnError(dbError)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.Error(t, err, "ListProducts should return an error if list query fails")
			assert.ErrorIs(t, err, dbError, "Returned error should be the database error")
			assert.Nil(t, products, "Returned products should be nil on error")
			assert.Zero(t, count, "Returned count should be zero on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("ScanError", func(t *testing.T) {
			// Arrange
			total := 1
			scanError := errors.New("scan error")

			mock.ExpectQuery(expectedCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
			// Return rows with incorrect column types to trigger a scan error
			rows := sqlmock.NewRows(productCols).AddRow("invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid", "invalid").RowError(0, scanError)
			mock.ExpectQuery(expectedListSQL).WithArgs(size, offset).WillReturnRows(rows)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.Error(t, err, "ListProducts should return an error on scan failure")
			assert.ErrorIs(t, err, scanError, "Returned error should be the scan error")
			assert.Nil(t, products, "Returned products should be nil on error")
			assert.Zero(t, count, "Returned count should be zero on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run("RowsError", func(t *testing.T) {
			// Arrange
			total := 1
			rowsError := errors.New("rows iteration error")

			mock.ExpectQuery(expectedCountSQL).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
			rows := sqlmock.NewRows(productCols).
				AddRow(uuid.New(), uuid.New(), "Prod 1", "", 10.0, 1, "SKU1", "active", time.Now(), time.Now(), uuid.New(), "Cat 1", "").
				CloseError(rowsError) // Simulate error during rows.Err() check after loop
			mock.ExpectQuery(expectedListSQL).WithArgs(size, offset).WillReturnRows(rows)

			// Act
			products, count, err := repo.ListProducts(ctx, page, size)

			// Assert
			require.Error(t, err, "ListProducts should return an error if rows.Err() returns an error")
			assert.ErrorIs(t, err, rowsError, "Returned error should be the rows iteration error")
			assert.Nil(t, products, "Returned products should be nil on error") // Or potentially partially filled depending on where error occurs
			assert.Zero(t, count, "Returned count should be zero on error")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
