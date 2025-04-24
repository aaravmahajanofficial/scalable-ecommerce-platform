package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateProduct(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockProductRepository(t)
	productService := service.NewProductService(mockRepo)
	ctx := context.Background()

	req := &models.CreateProductRequest{
		CategoryID:    uuid.New(),
		Name:          "Test Product",
		Description:   "Test Description",
		Price:         99.99,
		StockQuantity: 10,
		SKU:           "TEST-SKU-001",
	}

	t.Run("Success - Create Product", func(t *testing.T) {
		// Arrange
		mockRepo.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.CategoryID == req.CategoryID &&
				p.Name == req.Name &&
				p.Description == req.Description &&
				p.Price == req.Price &&
				p.StockQuantity == req.StockQuantity &&
				p.SKU == req.SKU &&
				p.Status == "active"
		})).Return(nil).Once()

		// Act
		product, err := productService.CreateProduct(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.Equal(t, req.Name, product.Name)
		assert.Equal(t, req.Description, product.Description)
		assert.Equal(t, req.Price, product.Price)
		assert.Equal(t, req.StockQuantity, product.StockQuantity)
		assert.Equal(t, req.SKU, product.SKU)
		assert.Equal(t, "active", product.Status)
		assert.NotEqual(t, uuid.Nil, product.ID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		// Arrange
		mockRepo.On("CreateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(appErrors.DatabaseError("DB Connection Failed")).Once()

		// Act
		product, err := productService.CreateProduct(ctx, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, product)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestGetProductByID(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockProductRepository(t)
	productService := service.NewProductService(mockRepo)
	ctx := context.Background()
	testID := uuid.New()

	t.Run("Success - Get Product", func(t *testing.T) {
		// Arrange
		expectedProduct := &models.Product{
			ID:   testID,
			Name: "Found Product",
		}

		// Mock Call
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(expectedProduct, nil).Once()

		// Act
		product, err := productService.GetProductByID(ctx, testID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, product)
		assert.Equal(t, expectedProduct, product)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(nil, sql.ErrNoRows).Once()

		// Act
		product, err := productService.GetProductByID(ctx, testID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, product)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(nil, appErrors.DatabaseError("DB Query Failed")).Once()

		// Act
		product, err := productService.GetProductByID(ctx, testID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, product)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateProduct(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockProductRepository(t)
	productService := service.NewProductService(mockRepo)
	ctx := context.Background()
	testID := uuid.New()

	existingProduct := &models.Product{
		ID:            testID,
		CategoryID:    uuid.New(),
		Name:          "Old Name",
		Description:   "Old Description",
		Price:         50.0,
		StockQuantity: 20,
		SKU:           "OLD-SKU",
		Status:        "active",
	}

	newCategoryID := uuid.New()
	newName := "New Name"
	newDescription := "New Description"
	newPrice := 60.0
	newStockQuantity := 30
	newStatus := "inactive"

	req := &models.UpdateProductRequest{
		CategoryID:    &newCategoryID,
		Name:          &newName,
		Description:   &newDescription,
		Price:         &newPrice,
		StockQuantity: &newStockQuantity,
		Status:        &newStatus,
	}

	t.Run("Success - Update Product", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(existingProduct, nil).Once()
		mockRepo.On("UpdateProduct", mock.Anything, mock.MatchedBy(func(p *models.Product) bool {
			return p.ID == testID &&
				p.CategoryID == *req.CategoryID &&
				p.Name == *req.Name &&
				p.Description == *req.Description &&
				p.Price == *req.Price &&
				p.StockQuantity == *req.StockQuantity &&
				p.Status == *req.Status &&
				p.SKU == existingProduct.SKU
		})).Return(nil).Once()

		// Act
		updatedProduct, err := productService.UpdateProduct(ctx, testID, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedProduct)
		assert.Equal(t, testID, updatedProduct.ID)
		assert.Equal(t, newCategoryID, updatedProduct.CategoryID)
		assert.Equal(t, newName, updatedProduct.Name)
		assert.Equal(t, newDescription, updatedProduct.Description)
		assert.Equal(t, newPrice, updatedProduct.Price)
		assert.Equal(t, newStockQuantity, updatedProduct.StockQuantity)
		assert.Equal(t, newStatus, updatedProduct.Status)
		assert.Equal(t, existingProduct.SKU, updatedProduct.SKU)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Product Not Found", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(nil, sql.ErrNoRows).Once()

		// Act
		updatedProduct, err := productService.UpdateProduct(ctx, testID, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedProduct)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)

		mockRepo.AssertNotCalled(t, "UpdateProduct")
	})

	t.Run("Failure - Update Database Error", func(t *testing.T) {
		// Arrange
		foundProduct := *existingProduct
		mockRepo.On("GetProductByID", mock.Anything, testID).Return(&foundProduct, nil).Once()
		mockRepo.On("UpdateProduct", mock.Anything, mock.AnythingOfType("*models.Product")).Return(appErrors.DatabaseError("DB Update Failed")).Once()

		// Act
		updatedProduct, err := productService.UpdateProduct(ctx, testID, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, updatedProduct)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)

		mockRepo.AssertExpectations(t)
	})
}

func TestListProducts(t *testing.T) {
	// Arrange
	mockRepo := mocks.NewMockProductRepository(t)
	productService := service.NewProductService(mockRepo)
	ctx := context.Background()
	page := 1
	pageSize := 10

	t.Run("Success - List Products", func(t *testing.T) {
		// Arrange
		expectedProducts := []*models.Product{
			{ID: uuid.New(), Name: "Product A"},
			{ID: uuid.New(), Name: "Product B"},
		}
		expectedTotal := 50

		mockRepo.On("ListProducts", mock.Anything, page, pageSize).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		products, total, err := productService.ListProducts(ctx, page, pageSize)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, products)
		assert.Len(t, products, len(expectedProducts))
		assert.Equal(t, expectedTotal, total)
		assert.Equal(t, expectedProducts, products)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		// Arrange
		mockRepo.On("ListProducts", mock.Anything, page, pageSize).Return(nil, 0, appErrors.DatabaseError("DB Query Failed")).Once()

		// Act
		products, total, err := productService.ListProducts(ctx, page, pageSize)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, products)
		assert.Zero(t, total)

		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)

		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Products List Empty", func(t *testing.T) {
		// Arrange
		var expectedProducts []*models.Product
		expectedTotal := 0
		mockRepo.On("ListProducts", mock.Anything, page, pageSize).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		products, total, err := productService.ListProducts(ctx, page, pageSize)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, products)
		assert.Len(t, products, 0)
		assert.Equal(t, expectedTotal, total)
		mockRepo.AssertExpectations(t)
	})
}
