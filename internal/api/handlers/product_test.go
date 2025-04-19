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
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
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
			CategoryID:    uuid.New(),
			Name:          "Test Product",
			Description:   "Test Description",
			Price:         99.99,
			StockQuantity: 10,
			SKU:           "TEST-SKU-001",
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

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Marshall the Data from map[string]interface{} to bytes
		databytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respProduct models.Product
		err = json.Unmarshal(databytes, &respProduct)
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
		mockProductService.AssertNotCalled(t, "CreateProduct")
	})

	t.Run("Invalid Input - Validation Error", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateProductRequest{
			CategoryID:    uuid.New(),
			Name:          "",
			Description:   "Test Description",
			Price:         0,
			StockQuantity: 10,
			SKU:           "TEST-SKU-001",
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeValidation)
		mockProductService.AssertNotCalled(t, "CreateProduct")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		reqBody := models.CreateProductRequest{
			CategoryID:    uuid.New(),
			Name:          "Test Product",
			Description:   "Test Description",
			Price:         99.99,
			StockQuantity: 10,
			SKU:           "TEST-SKU-001",
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockProductService.AssertExpectations(t)
	})
}

func TestGetProduct(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success - Get Product", func(t *testing.T) {
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

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Marshall the Data from map[string]interface{} to bytes
		databytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respProduct models.Product
		err = json.Unmarshal(databytes, &respProduct)
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeBadRequest)
		mockProductService.AssertNotCalled(t, "GetProductByID")
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockProductService.AssertExpectations(t)
	})
}

func TestUpdateProduct(t *testing.T) {
	mockProductService := new(mocks.ProductService)
	productHandler := handlers.NewProductHandler(mockProductService)

	t.Run("Success - Update Product", func(t *testing.T) {
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

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)

		// Marshall the Data from map[string]interface{} to bytes
		databytes, err := json.Marshal(resp.Data)
		assert.NoError(t, err)

		var respProduct models.Product
		err = json.Unmarshal(databytes, &respProduct)
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeBadRequest)
		mockProductService.AssertNotCalled(t, "UpdateProduct")
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
		mockProductService.AssertNotCalled(t, "UpdateProduct")
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
		mockProductService.AssertNotCalled(t, "UpdateProduct")
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

	t.Run("Failure - Service Error", func(t *testing.T) {
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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
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

		expectedProducts := []*models.Product{
			{ID: uuid.New(), Name: "Product 1", Price: 10.0, StockQuantity: 100},
			{ID: uuid.New(), Name: "Product 2", Price: 20.0, StockQuantity: 50},
		}
		expectedTotal := 25
		expectedPage := 1
		expectedPageSize := 10

		// Expect default page=1, pageSize=10
		mockProductService.On("ListProducts", mock.Anything, 1, 10).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		handler := productHandler.ListProducts()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		dataMap, ok := resp.Data.(map[string]any)
		assert.True(t, ok, "resp.Data should be a map[string]any")

		assert.EqualValues(t, expectedPage, dataMap["page"])
		assert.EqualValues(t, expectedPageSize, dataMap["pageSize"])
		assert.EqualValues(t, expectedTotal, dataMap["total"])

		// Marshal the 'data' field within the map back to bytes
		productsBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		// Unmarshal the product data
		var respProducts []*models.Product
		err = json.Unmarshal(productsBytes, &respProducts)
		assert.NoError(t, err)

		// Assert the product data
		assert.Len(t, respProducts, len(expectedProducts))
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

		expectedProducts := []*models.Product{
			{ID: uuid.New(), Name: "Product 3", Price: 30.0, StockQuantity: 30},
		}
		expectedTotal := 8

		mockProductService.On("ListProducts", mock.Anything, page, pageSize).Return(expectedProducts, expectedTotal, nil).Once()

		// Act
		handler := productHandler.ListProducts()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		// Unmarshal the base API response
		var resp *response.APIResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Data)

		dataMap, ok := resp.Data.(map[string]any)
		assert.True(t, ok, "resp.Data should be a map[string]any")
		assert.EqualValues(t, page, dataMap["page"])
		assert.EqualValues(t, pageSize, dataMap["pageSize"])
		assert.EqualValues(t, expectedTotal, dataMap["total"])

		// Marshal the 'data' field within the map back to bytes
		productsBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		// Unmarshal the product data
		var respProducts []*models.Product
		err = json.Unmarshal(productsBytes, &respProducts)
		assert.NoError(t, err)

		// Assert the product data
		assert.Len(t, respProducts, len(expectedProducts))
		assert.Equal(t, expectedProducts[0].ID, respProducts[0].ID)

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
				mockProductService := new(mocks.ProductService)
				productHandler := handlers.NewProductHandler(mockProductService)
				rr := httptest.NewRecorder()
				req := newTestRequest(http.MethodGet, tc.query, nil)

				mockProductService.On("ListProducts", mock.Anything, tc.expectPage, tc.expectSize).Return([]*models.Product{}, 0, nil).Once()

				// Act
				handler := productHandler.ListProducts()
				handler.ServeHTTP(rr, req)

				// Assert
				assert.Equal(t, http.StatusOK, rr.Code)

				var resp *response.APIResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.True(t, resp.Success)

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
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
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
