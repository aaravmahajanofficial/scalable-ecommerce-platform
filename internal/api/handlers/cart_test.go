package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// setupCartTest creates common test dependencies
func setupCartTest() (*mocks.CartService, *handlers.CartHandler) {
	mockCartService := new(mocks.CartService)
	cartHandler := handlers.NewCartHandler(mockCartService)
	return mockCartService, cartHandler
}

// createAuthenticatedRequest creates a request with authentication context
func createAuthenticatedRequest(method, url string, body []byte) (*http.Request, *models.Claims) {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Create user claims
	userID := uuid.New()
	claims := &models.Claims{
		UserID: userID,
		Email:  "test@example.com",
	}

	// Set up context with user claims and logger
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
	logger := slog.Default()
	ctx = context.WithValue(ctx, middleware.LoggerContextKey, logger)
	req = req.WithContext(ctx)

	return req, claims
}

func TestGetCart(t *testing.T) {
	t.Run("Success - Retrieve Cart", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()
		req, claims := createAuthenticatedRequest("GET", "/carts", nil)
		recorder := httptest.NewRecorder()

		// Create mock cart response
		mockCart := &models.Cart{
			ID:     uuid.New(),
			UserID: claims.UserID,
			Items:  []models.CartItem{},
		}

		// Setup mock expectations
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(mockCart, nil).Once()

		// Act
		handler := cartHandler.GetCart()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		// Verify response contains expected cart
		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotNil(t, response["data"])

		mockCartService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		_, cartHandler := setupCartTest()

		// Create request without auth context
		req := httptest.NewRequest("GET", "/carts", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add logger to context
		ctx := context.WithValue(req.Context(), middleware.LoggerContextKey, slog.Default())
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		// Act
		handler := cartHandler.GetCart()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Equal(t, "Authentication required", response["message"])
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()
		req, claims := createAuthenticatedRequest("GET", "/carts", nil)
		recorder := httptest.NewRecorder()

		// Setup mock expectations with error
		mockError := appErrors.InternalError("Database error")
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(nil, mockError).Once()

		// Act
		handler := cartHandler.GetCart()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))

		mockCartService.AssertExpectations(t)
	})
}

func TestAddItem(t *testing.T) {
	t.Run("Success - Add Item To Cart", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with item data
		addItemRequest := models.AddItemRequest{
			ProductID: uuid.New(),
			Quantity:  2,
		}
		requestBody, _ := json.Marshal(addItemRequest)

		req, claims := createAuthenticatedRequest("POST", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Create mock responses
		mockCart := &models.Cart{
			ID:     uuid.New(),
			UserID: claims.UserID,
			Items: []models.CartItem{
				{
					ProductID: addItemRequest.ProductID,
					Quantity:  addItemRequest.Quantity,
				},
			},
		}

		// Setup mock expectations
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(mockCart, nil).Once()
		mockCartService.On("AddItem", mock.Anything, claims.UserID, mock.MatchedBy(func(req *models.AddItemRequest) bool {
			return req.ProductID == addItemRequest.ProductID && req.Quantity == addItemRequest.Quantity
		})).Return(mockCart, nil).Once()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockCartService.AssertExpectations(t)
	})

	t.Run("Success - Create Cart Then Add Item", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with item data
		addItemRequest := models.AddItemRequest{
			ProductID: uuid.New(),
			Quantity:  2,
		}
		requestBody, _ := json.Marshal(addItemRequest)

		req, claims := createAuthenticatedRequest("POST", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Create mock responses
		mockCart := &models.Cart{
			ID:     uuid.New(),
			UserID: claims.UserID,
			Items: []models.CartItem{
				{
					ProductID: addItemRequest.ProductID,
					Quantity:  addItemRequest.Quantity,
				},
			},
		}

		// Setup mock expectations - first cart not found
		notFoundErr := appErrors.NotFoundError("Cart not found")
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(nil, notFoundErr).Once()

		// Then cart created
		mockCartService.On("CreateCart", mock.Anything, claims.UserID).Return(mockCart, nil).Once()

		// Then item added
		mockCartService.On("AddItem", mock.Anything, claims.UserID, mock.MatchedBy(func(req *models.AddItemRequest) bool {
			return req.ProductID == addItemRequest.ProductID && req.Quantity == addItemRequest.Quantity
		})).Return(mockCart, nil).Once()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockCartService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		_, cartHandler := setupCartTest()

		// Create request without auth context
		addItemRequest := models.AddItemRequest{
			ProductID: uuid.New(),
			Quantity:  2,
		}
		requestBody, _ := json.Marshal(addItemRequest)

		req := httptest.NewRequest("POST", "/carts/items", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// Add logger to context
		ctx := context.WithValue(req.Context(), middleware.LoggerContextKey, slog.Default())
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("Failure - Invalid Request Body", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with invalid JSON
		invalidJSON := []byte(`{"productID": "not-a-uuid", "quantity": "not-a-number"}`)

		req, claims := createAuthenticatedRequest("POST", "/carts/items", invalidJSON)
		recorder := httptest.NewRecorder()

		// Setup mock expectations
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(&models.Cart{}, nil).Once()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, recorder.Code)

		mockCartService.AssertExpectations(t)
	})

	t.Run("Failure - Cart Creation Error", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with item data
		addItemRequest := models.AddItemRequest{
			ProductID: uuid.New(),
			Quantity:  2,
		}
		requestBody, _ := json.Marshal(addItemRequest)

		req, claims := createAuthenticatedRequest("POST", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Setup mock expectations
		notFoundErr := appErrors.NotFoundError("Cart not found")
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(nil, notFoundErr).Once()

		// Create cart fails
		createErr := appErrors.InternalError("Failed to create cart")
		mockCartService.On("CreateCart", mock.Anything, claims.UserID).Return(nil, createErr).Once()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		mockCartService.AssertExpectations(t)
	})

	t.Run("Failure - Service Error When Adding Item", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with item data
		addItemRequest := models.AddItemRequest{
			ProductID: uuid.New(),
			Quantity:  2,
		}
		requestBody, _ := json.Marshal(addItemRequest)

		req, claims := createAuthenticatedRequest("POST", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Setup mock expectations
		mockCartService.On("GetCart", mock.Anything, claims.UserID).Return(&models.Cart{}, nil).Once()

		// AddItem fails
		addErr := appErrors.InternalError("Failed to add item")
		mockCartService.On("AddItem", mock.Anything, claims.UserID, mock.Anything).Return(nil, addErr).Once()

		// Act
		handler := cartHandler.AddItem()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		mockCartService.AssertExpectations(t)
	})
}

func TestUpdateQuantity(t *testing.T) {
	t.Run("Success - Update Item Quantity", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with update data
		updateRequest := models.UpdateQuantityRequest{
			ProductID: uuid.New(),
			Quantity:  5,
		}
		requestBody, _ := json.Marshal(updateRequest)

		req, claims := createAuthenticatedRequest("PUT", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Create mock response
		mockCart := &models.Cart{
			ID:     uuid.New(),
			UserID: claims.UserID,
			Items: []models.CartItem{
				{
					ProductID: updateRequest.ProductID,
					Quantity:  updateRequest.Quantity,
				},
			},
		}

		// Setup mock expectations
		mockCartService.On("UpdateQuantity", mock.Anything, claims.UserID, mock.MatchedBy(func(req *models.UpdateQuantityRequest) bool {
			return req.ProductID == updateRequest.ProductID && req.Quantity == updateRequest.Quantity
		})).Return(mockCart, nil).Once()

		// Act
		handler := cartHandler.UpdateQuantity()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		mockCartService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		_, cartHandler := setupCartTest()

		// Create request without auth context
		updateRequest := models.UpdateQuantityRequest{
			ProductID: uuid.New(),
			Quantity:  5,
		}
		requestBody, _ := json.Marshal(updateRequest)

		req := httptest.NewRequest("PUT", "/carts/items", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		// Add logger to context
		ctx := context.WithValue(req.Context(), middleware.LoggerContextKey, slog.Default())
		req = req.WithContext(ctx)

		recorder := httptest.NewRecorder()

		// Act
		handler := cartHandler.UpdateQuantity()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("Failure - Invalid Request Body", func(t *testing.T) {
		// Arrange
		_, cartHandler := setupCartTest()

		// Create request with invalid JSON
		invalidJSON := []byte(`{"productID": "not-a-uuid", "quantity": "not-a-number"}`)

		req, _ := createAuthenticatedRequest("PUT", "/carts/items", invalidJSON)
		recorder := httptest.NewRecorder()

		// Act
		handler := cartHandler.UpdateQuantity()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		mockCartService, cartHandler := setupCartTest()

		// Create request with update data
		updateRequest := models.UpdateQuantityRequest{
			ProductID: uuid.New(),
			Quantity:  5,
		}
		requestBody, _ := json.Marshal(updateRequest)

		req, claims := createAuthenticatedRequest("PUT", "/carts/items", requestBody)
		recorder := httptest.NewRecorder()

		// Setup mock expectations with error
		updateErr := appErrors.NotFoundError("Item not found in cart")
		mockCartService.On("UpdateQuantity", mock.Anything, claims.UserID, mock.Anything).Return(nil, updateErr).Once()

		// Act
		handler := cartHandler.UpdateQuantity()
		handler(recorder, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, recorder.Code)

		mockCartService.AssertExpectations(t)
	})
}
