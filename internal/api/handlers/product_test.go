package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// newTestRequest -> creates a request with context containing a logger
func newTestRequest(method, target string, body []byte) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewReader(body))

	logger := slog.Default()
	ctx := context.WithValue(req.Context(), middleware.LoggerKey, logger)
	return req.WithContext(ctx)
}

func TestCreateProduct(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success - Product Created", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateProductRequest{
			Name:          "Test Product",
			Description:   "Test Description",
			Price:         99.99,
			StockQuantity: 10,
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPost, "/products", reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")

		expectedProduct := &models.Product{
			ID:            uuid.New(),
			Name:          reqBody.Name,
			Description:   reqBody.Description,
			Price:         reqBody.Price,
			StockQuantity: reqBody.StockQuantity,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		mockProductService.On("CreateProduct", mock.Anything, &reqBody).Return(expectedProduct, nil).Once()

		// Act
		handler := productHandler.CreateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		var respProduct models.Product
		err := json.Unmarshal(rr.Body.Bytes(), &respProduct)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct.ID, respProduct.ID)
		assert.Equal(t, expectedProduct.Name, respProduct.Name)

		mockProductService.AssertExpectations(t)
	})

	t.Run("Invalid Input - Bad JSON", func(t *testing.T) {
		// Arrange
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPost, "/products", []byte("{invalid json"))
		req.Header.Set("Content-Type", "application/json")

		// Act
		handler := productHandler.CreateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockProductService.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
	})

	t.Run("Invalid Input - Validation Error", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateProductRequest{ // Name field not present
			Description:   "Test Description",
			Price:         0,
			StockQuantity: 10,
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPost, "/products", reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")

		// Act
		handler := productHandler.CreateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "validation failed")
		mockProductService.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateProductRequest{
			Name:          "Test Product",
			Description:   "Test Description",
			Price:         99.99,
			StockQuantity: 10,
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPost, "/products", reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")

		mockProductService.On("CreateProduct", mock.Anything, &reqBody).Return(nil, appErrors.DatabaseError("DB Connection Failed")).Once()

		// Act
		handler := productHandler.CreateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockProductService.AssertExpectations(t)
	})
}

func TestGetProduct(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products/"+productID.String(), nil)
		req.SetPathValue("id", productID.String()) // Set path parameter

		expectedProduct := &models.Product{
			ID:            productID,
			Name:          "Fetched Product",
			Description:   "Fetched Description",
			Price:         149.50,
			StockQuantity: 5,
			CreatedAt:     time.Now().Add(-time.Hour),
			UpdatedAt:     time.Now(),
		}

		mockProductService.On("GetProductByID", mock.Anything, productID).Return(expectedProduct, nil).Once()

		// Act
		handler := productHandler.GetProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var respProduct models.Product
		err := json.Unmarshal(rr.Body.Bytes(), &respProduct)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct.ID, respProduct.ID)
		assert.Equal(t, expectedProduct.Name, respProduct.Name)

		mockProductService.AssertExpectations(t)
	})

	t.Run("Invalid ID Format", func(t *testing.T) {
		// Arrange
		invalidID := "not-a-uuid"
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products/"+invalidID, nil)
		req.SetPathValue("id", invalidID)

		// Act
		handler := productHandler.GetProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid ID format")
		mockProductService.AssertNotCalled(t, "GetProductByID", mock.Anything, mock.Anything)
	})

	t.Run("Product Not Found", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products/"+productID.String(), nil)
		req.SetPathValue("id", productID.String())

		mockProductService.On("GetProductByID", mock.Anything, productID).Return(nil, appErrors.NotFoundError("Product Not Found")).Once()

		// Act
		handler := productHandler.GetProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeNotFound)
		mockProductService.AssertExpectations(t)
	})

	t.Run("Service Error", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products/"+productID.String(), nil)
		req.SetPathValue("id", productID.String())

		mockProductService.On("GetProductByID", mock.Anything, productID).Return(nil, appErrors.DatabaseError("Internal Server Error")).Once()

		// Act
		handler := productHandler.GetProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockProductService.AssertExpectations(t)
	})
}

func TestUpdateProduct(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		reqBody := models.UpdateProductRequest{
			Name:          stringPtr("Updated Product"),
			Description:   stringPtr("Updated Description"),
			Price:         float64Ptr(109.99),
			StockQuantity: intPtr(15),
		}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+productID.String(), reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", productID.String())

		expectedProduct := &models.Product{
			ID:            productID,
			Name:          *reqBody.Name,
			Description:   *reqBody.Description,
			Price:         *reqBody.Price,
			StockQuantity: *reqBody.StockQuantity,
			CreatedAt:     time.Now().Add(-time.Hour),
			UpdatedAt:     time.Now(),
		}

		mockProductService.On("UpdateProduct", mock.Anything, productID, &reqBody).Return(expectedProduct, nil).Once()

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var respProduct models.Product
		err := json.Unmarshal(rr.Body.Bytes(), &respProduct)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct.ID, respProduct.ID)
		assert.Equal(t, expectedProduct.Name, respProduct.Name)
		assert.WithinDuration(t, expectedProduct.UpdatedAt, respProduct.UpdatedAt, time.Second)

		mockProductService.AssertExpectations(t)
	})

	t.Run("Invalid ID Format", func(t *testing.T) {
		// Arrange
		invalidID := "not-a-uuid"
		reqBody := models.UpdateProductRequest{Name: stringPtr("Update")}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+invalidID, reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", invalidID)

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid ID format")
		mockProductService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Invalid Input - Bad JSON", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+productID.String(), []byte("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", productID.String())

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockProductService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Invalid Input - Validation Error", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		// Price is negative, which should fail validation if defined in UpdateProductRequest model
		reqBody := models.UpdateProductRequest{Price: float64Ptr(-10.0)}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+productID.String(), reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", productID.String())

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeValidation)
		mockProductService.AssertNotCalled(t, "UpdateProduct", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("Product Not Found", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		reqBody := models.UpdateProductRequest{Name: stringPtr("Update")}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+productID.String(), reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", productID.String())

		mockProductService.On("UpdateProduct", mock.Anything, productID, &reqBody).Return(nil, appErrors.NotFoundError("Product Not Found")).Once()

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeNotFound)
		mockProductService.AssertExpectations(t)
	})

	t.Run("Service Error", func(t *testing.T) {
		// Arrange
		productID := uuid.New()
		reqBody := models.UpdateProductRequest{Name: stringPtr("Update")}
		reqBodyBytes, _ := json.Marshal(reqBody)

		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodPut, "/products/"+productID.String(), reqBodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", productID.String())

		mockProductService.On("UpdateProduct", mock.Anything, productID, &reqBody).Return(nil, appErrors.DatabaseError("DB Update Failed")).Once()

		// Act
		handler := productHandler.UpdateProduct()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockProductService.AssertExpectations(t)
	})
}

func TestListProducts(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success - Default Pagination", func(t *testing.T) {
		// Arrange
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products", nil)

		expectedProducts := []models.Product{
			{ID: uuid.New(), Name: "Product 1", Price: 10.0, StockQuantity: 100},
			{ID: uuid.New(), Name: "Product 2", Price: 20.0, StockQuantity: 50},
		}
		expectedTotal := 25

		// Expect default page=1, pageSize=10
		mockProductService.On("ListProducts", mock.Anything, 1, 10).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		handler := productHandler.ListProducts()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp models.PaginatedResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)

		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 10, resp.PageSize)
		assert.Equal(t, expectedTotal, resp.Total)
		assert.Len(t, resp.Data, len(expectedProducts))

		// Unmarshal the Data field into []models.Product
		var respProducts []models.Product
		dataBytes, _ := json.Marshal(resp.Data) // Marshal the interface{} back to bytes
		err = json.Unmarshal(dataBytes, &respProducts)
		assert.NoError(t, err)
		assert.Equal(t, expectedProducts[0].ID, respProducts[0].ID)
		assert.Equal(t, expectedProducts[1].Name, respProducts[1].Name)

		mockProductService.AssertExpectations(t)
	})

	t.Run("Success - Custom Pagination", func(t *testing.T) {
		// Arrange
		page := 2
		pageSize := 5
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, fmt.Sprintf("/products?page=%d&pageSize=%d", page, pageSize), nil)

		expectedProducts := []models.Product{
			{ID: uuid.New(), Name: "Product 3", Price: 30.0, StockQuantity: 30},
		}
		expectedTotal := 8

		mockProductService.On("ListProducts", mock.Anything, page, pageSize).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		handler := productHandler.ListProducts()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp models.PaginatedResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)

		assert.Equal(t, page, resp.Page)
		assert.Equal(t, pageSize, resp.PageSize)
		assert.Equal(t, expectedTotal, resp.Total)
		assert.Len(t, resp.Data, len(expectedProducts))

		mockProductService.AssertExpectations(t)
	})

	t.Run("Success - Invalid Pagination Defaults", func(t *testing.T) {
		// Arrange
		testCases := []struct {
			name       string
			query      string
			expectPage int
			expectSize int
		}{
			{"Invalid page", "/products?page=abc&pageSize=5", 1, 5},
			{"Page < 1", "/products?page=0&pageSize=5", 1, 5},
			{"Invalid pageSize", "/products?page=2&pageSize=xyz", 2, 10},
			{"PageSize < 1", "/products?page=2&pageSize=0", 2, 10},
			{"PageSize > 100", "/products?page=2&pageSize=101", 2, 10},
			{"Both invalid", "/products?page=-1&pageSize=abc", 1, 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rr := httptest.NewRecorder()
				req := newTestRequest(http.MethodGet, tc.query, nil)

				mockProductService.On("ListProducts", mock.Anything, tc.expectPage, tc.expectSize).Return([]models.Product{}, 0, nil).Once()

				// Act
				handler := productHandler.ListProducts()
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusOK, rr.Code)
				mockProductService.AssertExpectations(t)
			})
		}
	})

	t.Run("Service Error", func(t *testing.T) {
		// Arrange
		rr := httptest.NewRecorder()
		req := newTestRequest(http.MethodGet, "/products?page=1&pageSize=10", nil)

		mockProductService.On("ListProducts", mock.Anything, 1, 10).Return(nil, 0, appErrors.DatabaseError("DB Query Failed")).Once()

		// Act
		handler := productHandler.ListProducts()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeInternal)
		mockProductService.AssertExpectations(t)
	})
}

// Helper functions for pointer types used in UpdateProductRequest
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
