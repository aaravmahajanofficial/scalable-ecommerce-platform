package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services/mocks"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/testutils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestCreateOrder tests the CreateOrder handler
func TestCreateOrder(t *testing.T) {
	mockOrderService := new(mocks.OrderService)
	orderHandler := handlers.NewOrderHandler(mockOrderService)
	userID := uuid.New()
	orderID := uuid.New()

	t.Run("Success - Order Created", func(t *testing.T) {
		// Arrange
		createReq := models.CreateOrderRequest{
			CustomerID: userID,
			ShippingAddress: models.Address{
				Street:     "123 Test Street",
				City:       "Test City",
				State:      "TS",
				PostalCode: "12345",
				Country:    "US",
			},
			Items: []models.OrderItem{
				{
					ProductID: uuid.New(),
					Quantity:  1,
					UnitPrice: 50.0,
				},
			},
		}
		expectedOrder := &models.Order{
			ID:         orderID,
			CustomerID: userID,
			Status:     models.OrderStatusPending,
			ShippingAddress: &models.Address{
				Street:     "123 Test Street",
				City:       "Test City",
				State:      "TS",
				PostalCode: "12345",
				Country:    "US",
			},
			Items: []models.OrderItem{
				{
					ProductID: createReq.Items[0].ProductID,
					Quantity:  1,
					UnitPrice: 50.0,
				},
			},
			TotalAmount: 50.0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Mock Call
		mockOrderService.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.CreateOrderRequest")).Return(expectedOrder, nil).Once()

		// Create request body
		bodyBytes, _ := json.Marshal(createReq)
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/orders", bytes.NewReader(bodyBytes), userID, pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.CreateOrder()
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

		var respOrder models.Order
		err = json.Unmarshal(databytes, &respOrder)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrder.ID, respOrder.ID)
		assert.Equal(t, expectedOrder.CustomerID, respOrder.CustomerID)
		assert.Equal(t, expectedOrder.Status, respOrder.Status)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		createReq := models.CreateOrderRequest{
			ShippingAddress: models.Address{
				Street:     "123 Test Street",
				City:       "Test City",
				State:      "TS",
				PostalCode: "12345",
				Country:    "US",
			},
			Items: []models.OrderItem{
				{
					ProductID: uuid.New(),
					Quantity:  1,
					UnitPrice: 50.0,
				},
			},
		}
		bodyBytes, _ := json.Marshal(createReq)
		req := testutils.CreateTestRequestWithoutContext(http.MethodPost, "/orders", bytes.NewReader(bodyBytes), nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.CreateOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrderService.AssertNotCalled(t, "CreateOrder")
	})

	t.Run("Failure - Invalid Input", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/orders", bytes.NewReader([]byte("{invalid json")), userID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.CreateOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockOrderService.AssertNotCalled(t, "CreateOrder")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		createReq := models.CreateOrderRequest{
			CustomerID: uuid.New(),
			ShippingAddress: models.Address{
				Street:     "123 Test Street",
				City:       "Test City",
				State:      "TS",
				PostalCode: "12345",
				Country:    "US",
			},
			Items: []models.OrderItem{
				{
					ProductID: uuid.New(),
					Quantity:  1,
					UnitPrice: 50.0,
				},
			},
		}
		// Mock Call
		mockOrderService.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.CreateOrderRequest")).Return(nil, appErrors.DatabaseError("DB Connection Failed")).Once()

		bodyBytes, _ := json.Marshal(createReq)
		req := testutils.CreateTestRequestWithContext(http.MethodPost, "/orders", bytes.NewReader(bodyBytes), userID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.CreateOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockOrderService.AssertExpectations(t)
	})
}

func TestGetOrder(t *testing.T) {
	mockOrderService := new(mocks.OrderService)
	orderHandler := handlers.NewOrderHandler(mockOrderService)
	userID := uuid.New()
	orderID := uuid.New()

	t.Run("Success - Get Order", func(t *testing.T) {
		// Arrange
		expectedOrder := &models.Order{
			ID:          orderID,
			CustomerID:  userID,
			Status:      models.OrderStatusPending,
			TotalAmount: 50.0,
		}

		// Mock Call
		mockOrderService.On("GetOrderById", mock.Anything, orderID).Return(expectedOrder, nil).Once()

		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/orders/%s", orderID), nil, userID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
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

		var respOrder models.Order
		err = json.Unmarshal(databytes, &respOrder)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrder.ID, respOrder.ID)
		assert.Equal(t, expectedOrder.CustomerID, respOrder.CustomerID)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, fmt.Sprintf("/orders/%s", orderID), nil, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrderService.AssertNotCalled(t, "GetOrderById")
	})

	t.Run("Failure - Invalid Order ID", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/orders/invalid-uuid", nil, userID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockOrderService.AssertNotCalled(t, "GetOrderById")
	})

	t.Run("Failure - Order Not Found", func(t *testing.T) {
		// Arrange
		notFoundErr := appErrors.NewAppError(appErrors.ErrCodeNotFound, "order not found", http.StatusNotFound)

		// Mock Call
		mockOrderService.On("GetOrderById", mock.Anything, orderID).Return(nil, notFoundErr).Once()
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/orders/%s", orderID), nil, userID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		mockOrderService.AssertExpectations(t)
	})

	t.Run("Forbidden - Wrong User", func(t *testing.T) {
		// Arrange
		otherUserID := uuid.New()
		orderFromOtherUser := &models.Order{
			ID:         orderID,
			CustomerID: otherUserID,
			Status:     models.OrderStatusPending,
		}

		// Mock Call
		mockOrderService.On("GetOrderById", mock.Anything, orderID).Return(orderFromOtherUser, nil).Once()
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/orders/%s", orderID), nil, userID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, rr.Code)
		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange

		// Mock Call
		mockOrderService.On("GetOrderById", mock.Anything, orderID).Return(nil, appErrors.DatabaseError("DB Connection Failed")).Once()
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodGet, fmt.Sprintf("/orders/%s", orderID), nil, userID, pathParams)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.GetOrder()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockOrderService.AssertExpectations(t)
	})
}

func TestListOrders(t *testing.T) {
	mockOrderService := new(mocks.OrderService)
	orderHandler := handlers.NewOrderHandler(mockOrderService)
	userID := uuid.New()

	t.Run("Success - Default Pagination", func(t *testing.T) {
		// Arrange
		expectedOrders := []models.Order{
			{ID: uuid.New(), CustomerID: userID, Status: models.OrderStatusDelivered},
			{ID: uuid.New(), CustomerID: userID, Status: models.OrderStatusShipping},
		}
		expectedTotal := 5
		expectedPage := 1
		expectedPageSize := 10

		// Mock Call
		mockOrderService.On("ListOrdersByCustomer", mock.Anything, userID, expectedPage, expectedPageSize).Return(expectedOrders, expectedTotal, nil).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/orders", nil, userID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.ListOrders()
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
		ordersBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		var respOrders []models.Order
		err = json.Unmarshal(ordersBytes, &respOrders)
		assert.NoError(t, err)

		// Assert the order data
		assert.Len(t, respOrders, len(expectedOrders))
		assert.Equal(t, expectedOrders[0].ID, respOrders[0].ID)
		assert.Equal(t, expectedOrders[1].CustomerID, respOrders[1].CustomerID)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("Success - Custom Pagination", func(t *testing.T) {
		// Arrange
		page := 2
		pageSize := 20

		expectedOrders := []models.Order{
			{ID: uuid.New(), CustomerID: userID, Status: models.OrderStatusDelivered},
		}

		expectedTotal := 5

		// Mock Call
		mockOrderService.On("ListOrdersByCustomer", mock.Anything, userID, page, pageSize).Return(expectedOrders, expectedTotal, nil).Once()

		target := fmt.Sprintf("/orders?page=%d&pageSize=%d", page, pageSize)
		req := testutils.CreateTestRequestWithContext(http.MethodGet, target, nil, userID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.ListOrders()
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
		ordersBytes, err := json.Marshal(dataMap["data"])
		assert.NoError(t, err)

		// Unmarshal the order data
		var respOrders []models.Order
		err = json.Unmarshal(ordersBytes, &respOrders)
		assert.NoError(t, err)

		// Assert the order data
		assert.Len(t, respOrders, len(expectedOrders))
		assert.Equal(t, expectedOrders[0].ID, respOrders[0].ID)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("Success - Invalid Pagination Params (Uses Defaults)", func(t *testing.T) {

		testCases := []struct {
			name       string
			query      string
			expectPage int
			expectSize int
		}{
			{"Invalid page", "/orders?page=abc&pageSize=5", 1, 5},
			{"Page < 1", "/orders?page=0&pageSize=5", 1, 5},
			{"Invalid pageSize", "/orders?page=2&pageSize=xyz", 2, 10},
			{"PageSize < 1", "/orders?page=2&pageSize=0", 2, 10},
			{"PageSize > 100", "/orders?page=2&pageSize=101", 2, 10},
			{"Both invalid", "/orders?page=-1&pageSize=abc", 1, 10},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Arrange
				expectedOrders := []models.Order{}
				expectedTotal := 0

				mockOrderService.On("ListOrdersByCustomer", mock.Anything, userID, tc.expectPage, tc.expectSize).
					Return(expectedOrders, expectedTotal, nil).Once()

				req := testutils.CreateTestRequestWithContext(http.MethodGet, tc.query, nil, userID, nil)
				rr := httptest.NewRecorder()

				// Act
				handler := orderHandler.ListOrders()
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
				assert.EqualValues(t, tc.expectPage, dataMap["page"])
				assert.EqualValues(t, tc.expectSize, dataMap["pageSize"])
				assert.EqualValues(t, expectedTotal, dataMap["total"])

				mockOrderService.AssertExpectations(t)
			})
		}
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, "/orders", nil, nil) // No user context
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.ListOrders()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrderService.AssertNotCalled(t, "ListOrdersByCustomer")
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		defaultPage := 1
		defaultPageSize := 10

		// Mock Call
		mockOrderService.On("ListOrdersByCustomer", mock.Anything, userID, defaultPage, defaultPageSize).Return(nil, 0, appErrors.DatabaseError("DB Failed")).Once()

		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/orders", nil, userID, nil)
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.ListOrders()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockOrderService.AssertExpectations(t)
	})
}

func TestUpdateOrderStatus(t *testing.T) {
	mockOrderService := new(mocks.OrderService)
	orderHandler := handlers.NewOrderHandler(mockOrderService)
	adminUserID := uuid.New() // Assuming an admin/updater user ID
	orderID := uuid.New()
	customerID := uuid.New()

	t.Run("Success - Order Updated", func(t *testing.T) {
		// Arrange
		updateReq := models.UpdateOrderStatusRequest{
			Status: models.OrderStatusShipping,
		}
		expectedOrder := &models.Order{
			ID:         orderID,
			CustomerID: customerID,
			Status:     updateReq.Status,
			UpdatedAt:  time.Now(),
		}

		// Mock Call
		mockOrderService.On("UpdateOrderStatus", mock.Anything, orderID, updateReq.Status).Return(expectedOrder, nil).Once()

		bodyBytes, _ := json.Marshal(updateReq)
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodPatch, fmt.Sprintf("/orders/%s/status", orderID), bytes.NewReader(bodyBytes), adminUserID, pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
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

		var respOrder models.Order
		err = json.Unmarshal(databytes, &respOrder)
		assert.NoError(t, err)
		assert.Equal(t, expectedOrder.ID, respOrder.ID)
		assert.Equal(t, expectedOrder.Status, respOrder.Status)
		assert.WithinDuration(t, expectedOrder.UpdatedAt, respOrder.UpdatedAt, time.Second)

		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failure - Unauthorized", func(t *testing.T) {
		// Arrange
		updateReq := models.UpdateOrderStatusRequest{Status: models.OrderStatusShipping}
		bodyBytes, _ := json.Marshal(updateReq)
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithoutContext(http.MethodPatch, fmt.Sprintf("/orders/%s/status", orderID), bytes.NewReader(bodyBytes), pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockOrderService.AssertNotCalled(t, "UpdateOrderStatus")
	})

	t.Run("Failure - Invalid Order ID", func(t *testing.T) {
		// Arrange
		updateReq := models.UpdateOrderStatusRequest{Status: models.OrderStatusShipping}
		bodyBytes, _ := json.Marshal(updateReq)
		req := testutils.CreateTestRequestWithContext(http.MethodPatch, "/orders/invalid-uuid/status", bytes.NewReader(bodyBytes), adminUserID, nil)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockOrderService.AssertNotCalled(t, "UpdateOrderStatus")
	})

	t.Run("Invalid Input - Bad JSON", func(t *testing.T) {
		// Arrange
		invalidBody := `{"status": "invalid_status"}`
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodPatch, fmt.Sprintf("/orders/%s/status", orderID), bytes.NewReader([]byte(invalidBody)), adminUserID, pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockOrderService.AssertNotCalled(t, "UpdateOrderStatus")
	})

	t.Run("Order Not Found", func(t *testing.T) {
		// Arrange
		updateReq := models.UpdateOrderStatusRequest{Status: models.OrderStatusShipping}
		notFoundErr := appErrors.NewAppError(appErrors.ErrCodeNotFound, "order not found", http.StatusNotFound)

		// Mock Call
		mockOrderService.On("UpdateOrderStatus", mock.Anything, orderID, updateReq.Status).Return(nil, notFoundErr).Once()

		bodyBytes, _ := json.Marshal(updateReq)
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodPatch, fmt.Sprintf("/orders/%s/status", orderID), bytes.NewReader(bodyBytes), adminUserID, pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeNotFound)
		mockOrderService.AssertExpectations(t)
	})

	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		updateReq := models.UpdateOrderStatusRequest{Status: models.OrderStatusShipping}

		// Mock Call
		mockOrderService.On("UpdateOrderStatus", mock.Anything, orderID, updateReq.Status).Return(nil, appErrors.DatabaseError("DB Update Failed")).Once()

		bodyBytes, _ := json.Marshal(updateReq)
		pathParams := map[string]string{
			"id": orderID.String(),
		}
		req := testutils.CreateTestRequestWithContext(http.MethodPatch, fmt.Sprintf("/orders/%s/status", orderID), bytes.NewReader(bodyBytes), adminUserID, pathParams)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Act
		handler := orderHandler.UpdateOrderStatus()
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), appErrors.ErrCodeDatabaseError)
		mockOrderService.AssertExpectations(t)
	})
}
