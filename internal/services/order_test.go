package service_test

// import (
// 	"context"
// 	"errors"
// 	"testing"
// 	"time"

// 	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
// 	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
// 	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
// 	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
// 	"github.com/google/uuid"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// func setupOrderServiceTest(t *testing.T) (service.OrderService, *mocks.OrderRepository, *mocks.CartRepository, *mocks.ProductRepository) {
// 	mockOrderRepo := mocks.NewOrderRepository(t)
// 	mockCartRepo := mocks.NewCartRepository(t)
// 	mockProductRepo := mocks.NewProductRepository(t)
// 	orderService := service.NewOrderService(mockOrderRepo, mockCartRepo, mockProductRepo)
// 	return orderService, mockOrderRepo, mockCartRepo, mockProductRepo
// }

// func TestCreateOrder_Success(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, mockCartRepo, mockProductRepo := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	productID1 := uuid.New()
// 	productID2 := uuid.New()

// 	mockCart := &models.Cart{
// 		UserID: customerID,
// 		Items: map[string]models.CartItem{
// 			productID1.String(): {ProductID: productID1, Quantity: 2},
// 			productID2.String(): {ProductID: productID2, Quantity: 1},
// 		},
// 	}

// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil).Once()

// 	// Mock Call Product Repository
// 	mockProduct1 := &models.Product{ID: productID1, StockQuantity: 10, Price: 50.0}
// 	mockProduct2 := &models.Product{ID: productID2, StockQuantity: 5, Price: 100.0}
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Once()
// 	mockProductRepo.On("GetProductByID", ctx, productID2).Return(mockProduct2, nil).Once()

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("CreateOrder", ctx, mock.AnythingOfType("*models.Order")).Return(nil).Run(func(args mock.Arguments) {
// 		orderArg := args.Get(1).(*models.Order)
// 		assert.Equal(t, customerID, orderArg.CustomerID)
// 		assert.Equal(t, models.OrderStatusPending, orderArg.Status)
// 		assert.Equal(t, models.PaymentStatusPending, orderArg.PaymentStatus)
// 		assert.Len(t, orderArg.Items, 2)
// 		assert.Equal(t, 200.0, orderArg.TotalAmount)
// 	}).Once()

// 	// Mock Call Product Repository
// 	// Need to mock GetProductByID again for the updating quantity
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Once()
// 	mockProductRepo.On("GetProductByID", ctx, productID2).Return(mockProduct2, nil).Once()
// 	mockProductRepo.On("UpdateProduct", ctx, mock.MatchedBy(func(p *models.Product) bool { return p.ID == productID1 && p.StockQuantity == 8 })).Return(nil).Once() // 10 - 2 = 8
// 	mockProductRepo.On("UpdateProduct", ctx, mock.MatchedBy(func(p *models.Product) bool { return p.ID == productID2 && p.StockQuantity == 4 })).Return(nil).Once() // 5 - 1 = 4

// 	req := &models.CreateOrderRequest{
// 		CustomerID: customerID,
// 		Items: []models.OrderItem{
// 			{ProductID: productID1, Quantity: 2, UnitPrice: 50.0},
// 			{ProductID: productID2, Quantity: 1, UnitPrice: 100.0},
// 		},
// 		ShippingAddress: models.Address{
// 			Street: "123 Main St", City: "Anytown", PostalCode: "12345", Country: "USA",
// 		},
// 	}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.NoError(t, err)
// 	assert.NotNil(t, order)
// 	assert.Equal(t, customerID, order.CustomerID)
// 	assert.Equal(t, models.OrderStatusPending, order.Status)
// 	assert.Equal(t, 200.0, order.TotalAmount)
// 	assert.Len(t, order.Items, 2)

// 	mockCartRepo.AssertExpectations(t)
// 	mockProductRepo.AssertExpectations(t)
// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestCreateOrder_CartNotFound(t *testing.T) {
// 	// Arrange
// 	orderService, _, mockCartRepo, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()

// 	// Mock Call Cart Repository
// 	mockErr := errors.New("mock cart repo error")
// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, mockErr)

// 	req := &models.CreateOrderRequest{CustomerID: customerID}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Cart not found")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockCartRepo.AssertExpectations(t)
// }

// func TestCreateOrder_EmptyCart(t *testing.T) {
// 	// Arrange
// 	orderService, _, mockCartRepo, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()

// 	// Mock Call Cart Repository
// 	mockCart := &models.Cart{UserID: customerID, Items: map[string]models.CartItem{}}
// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)

// 	req := &models.CreateOrderRequest{CustomerID: customerID}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Cannot create order with empty cart")

// 	mockCartRepo.AssertExpectations(t)
// }

// func TestCreateOrder_ProductNotFound(t *testing.T) {
// 	// Arrange
// 	orderService, _, mockCartRepo, mockProductRepo := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	productID1 := uuid.New() // Product that exists
// 	productID2 := uuid.New() // Product that doesn't exist

// 	// Mock Call Cart Repository
// 	mockCart := &models.Cart{
// 		UserID: customerID,
// 		Items: map[string]models.CartItem{
// 			productID1.String(): {ProductID: productID1, Quantity: 2},
// 			productID2.String(): {ProductID: productID2, Quantity: 1},
// 		},
// 	}
// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)

// 	// Mock Call Product Repository
// 	mockProduct1 := &models.Product{ID: productID1, StockQuantity: 10}
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Once()

// 	mockErr := errors.New("mock product repo error")
// 	mockProductRepo.On("GetProductByID", ctx, productID2).Return(nil, mockErr).Once()

// 	req := &models.CreateOrderRequest{CustomerID: customerID}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Product not found: "+productID2.String())
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockCartRepo.AssertExpectations(t)
// 	mockProductRepo.AssertExpectations(t)
// }

// func TestCreateOrder_InsufficientStock(t *testing.T) {
// 	// Arrange
// 	orderService, _, mockCartRepo, mockProductRepo := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	productID1 := uuid.New()

// 	mockCart := &models.Cart{
// 		UserID: customerID,
// 		Items: map[string]models.CartItem{
// 			productID1.String(): {ProductID: productID1, Quantity: 5},
// 		},
// 	}
// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)

// 	// Mock Call Product Repository
// 	mockProduct1 := &models.Product{ID: productID1, StockQuantity: 3} // Only 3 in stock
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Once()

// 	req := &models.CreateOrderRequest{CustomerID: customerID}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Insufficient stock for product: "+productID1.String())

// 	mockCartRepo.AssertExpectations(t)
// 	mockProductRepo.AssertExpectations(t)
// }

// func TestCreateOrder_CreateOrderRepoError(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, mockCartRepo, mockProductRepo := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	productID1 := uuid.New()

// 	// Mock Call Cart Repo
// 	mockCart := &models.Cart{
// 		UserID: customerID,
// 		Items: map[string]models.CartItem{
// 			productID1.String(): {ProductID: productID1, Quantity: 1},
// 		},
// 	}

// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)

// 	// Mock Call Product Repo
// 	mockProduct1 := &models.Product{ID: productID1, StockQuantity: 10, Price: 25.0}
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Once()

// 	// Mock Call Order Repo
// 	mockErr := errors.New("mock create order error")
// 	mockOrderRepo.On("CreateOrder", ctx, mock.AnythingOfType("*models.Order")).Return(mockErr).Once()

// 	req := &models.CreateOrderRequest{
// 		CustomerID:      customerID,
// 		Items:           []models.OrderItem{{ProductID: productID1, Quantity: 1, UnitPrice: 25.0}},
// 		ShippingAddress: models.Address{},
// 	}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Failed to create order")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockCartRepo.AssertExpectations(t)
// 	mockProductRepo.AssertExpectations(t)
// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestCreateOrder_UpdateInventoryRepoError(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, mockCartRepo, mockProductRepo := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	productID1 := uuid.New()

// 	// Mock Call Cart Repo
// 	mockCart := &models.Cart{
// 		UserID: customerID,
// 		Items: map[string]models.CartItem{
// 			productID1.String(): {ProductID: productID1, Quantity: 1},
// 		},
// 	}
// 	mockCartRepo.On("GetCartByCustomerID", ctx, customerID).Return(mockCart, nil)

// 	// Mock Call Product Repo
// 	mockProduct1 := &models.Product{ID: productID1, StockQuantity: 10, Price: 25.0}
// 	mockProductRepo.On("GetProductByID", ctx, productID1).Return(mockProduct1, nil).Twice() // Called once for check, once for update loop

// 	// Mock Call Order Repo
// 	mockOrderRepo.On("CreateOrder", ctx, mock.AnythingOfType("*models.Order")).Return(nil).Once()

// 	// Mock Call Product Repo
// 	mockErr := errors.New("mock update product error")
// 	mockProductRepo.On("UpdateProduct", ctx, mock.AnythingOfType("*models.Product")).Return(mockErr).Once()

// 	req := &models.CreateOrderRequest{
// 		CustomerID:      customerID,
// 		Items:           []models.OrderItem{{ProductID: productID1, Quantity: 1, UnitPrice: 25.0}},
// 		ShippingAddress: models.Address{},
// 	}

// 	// Act
// 	order, err := orderService.CreateOrder(ctx, req)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Failed to update inventory")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockCartRepo.AssertExpectations(t)
// 	mockProductRepo.AssertExpectations(t)
// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestGetOrderById_Success(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	orderID := uuid.New()
// 	expectedOrder := &models.Order{ID: orderID, CustomerID: uuid.New(), Status: models.OrderStatusDelivered}

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("GetOrderById", ctx, orderID).Return(expectedOrder, nil).Once()

// 	// Act
// 	order, err := orderService.GetOrderById(ctx, orderID)

// 	// Assert
// 	assert.NoError(t, err)
// 	assert.NotNil(t, order)
// 	assert.Equal(t, expectedOrder, order)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestGetOrderById_NotFound(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	orderID := uuid.New()

// 	// Mock Call Order Repository
// 	mockErr := errors.New("mock repo error: not found")
// 	mockOrderRepo.On("GetOrderById", ctx, orderID).Return(nil, mockErr).Once()

// 	// Act
// 	order, err := orderService.GetOrderById(ctx, orderID)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Order not found")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestListOrdersByCustomer_Success(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	page, size := 1, 5
// 	expectedOrders := []models.Order{
// 		{ID: uuid.New(), CustomerID: customerID},
// 		{ID: uuid.New(), CustomerID: customerID},
// 	}
// 	expectedTotal := 10 // Simulate more total orders than returned in this page

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("ListOrdersByCustomer", ctx, customerID, page, size).Return(expectedOrders, expectedTotal, nil).Once()

// 	// Act
// 	orders, total, err := orderService.ListOrdersByCustomer(ctx, customerID, page, size)

// 	// Assert
// 	assert.NoError(t, err)
// 	assert.Equal(t, expectedOrders, orders)
// 	assert.Equal(t, expectedTotal, total)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestListOrdersByCustomer_PaginationDefaults(t *testing.T) {
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	defaultPage, defaultSize := 1, 10

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("ListOrdersByCustomer", ctx, customerID, defaultPage, defaultSize).Return([]models.Order{}, 0, nil).Once()

// 	// Act
// 	orders, total, err := orderService.ListOrdersByCustomer(ctx, customerID, 0, 15) // page < 1, size > 10

// 	// Assert
// 	assert.NoError(t, err)
// 	assert.Empty(t, orders)
// 	assert.Equal(t, 0, total)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestListOrdersByCustomer_RepoError(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	customerID := uuid.New()
// 	page, size := 1, 10

// 	// Mock Call Order Repository
// 	mockErr := errors.New("mock repo list error")
// 	mockOrderRepo.On("ListOrdersByCustomer", ctx, customerID, page, size).Return(nil, 0, mockErr).Once()

// 	// Act
// 	orders, total, err := orderService.ListOrdersByCustomer(ctx, customerID, page, size)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, orders)
// 	assert.Equal(t, 0, total)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Failed to fetch orders")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestUpdateOrderStatus_Success(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	orderID := uuid.New()
// 	newStatus := models.OrderStatusShipping
// 	originalOrder := &models.Order{ID: orderID, Status: models.OrderStatusPending, UpdatedAt: time.Now().Add(-time.Hour)}
// 	updatedOrder := &models.Order{ID: orderID, Status: newStatus, UpdatedAt: time.Now()}

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("GetOrderById", ctx, orderID).Return(originalOrder, nil).Once()

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).Return(updatedOrder, nil).Once()

// 	// Act
// 	order, err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)

// 	// Assert
// 	assert.NoError(t, err)
// 	assert.NotNil(t, order)
// 	assert.Equal(t, updatedOrder, order) // Check if the returned order is the one from the update call
// 	assert.Equal(t, newStatus, order.Status)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestUpdateOrderStatus_OrderNotFound(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	orderID := uuid.New()
// 	newStatus := models.OrderStatusShipping

// 	// Mock Call Order Repository
// 	mockErr := errors.New("mock repo get error: not found")
// 	mockOrderRepo.On("GetOrderById", ctx, orderID).Return(nil, mockErr).Once()

// 	// Act
// 	order, err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Order not found")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockOrderRepo.AssertExpectations(t)
// }

// func TestUpdateOrderStatus_UpdateRepoError(t *testing.T) {
// 	// Arrange
// 	orderService, mockOrderRepo, _, _ := setupOrderServiceTest(t)
// 	ctx := context.Background()
// 	orderID := uuid.New()
// 	newStatus := models.OrderStatusDelivered
// 	originalOrder := &models.Order{ID: orderID, Status: models.OrderStatusShipping}

// 	// Mock Call Order Repository
// 	mockOrderRepo.On("GetOrderById", ctx, orderID).Return(originalOrder, nil).Once()

// 	// Mock Call Order Repository
// 	mockErr := errors.New("mock repo update error")
// 	mockOrderRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).Return(nil, mockErr).Once()

// 	// Act
// 	order, err := orderService.UpdateOrderStatus(ctx, orderID, newStatus)

// 	// Assert
// 	assert.Error(t, err)
// 	assert.Nil(t, order)
// 	appErr, ok := err.(*appErrors.AppError)
// 	assert.True(t, ok)
// 	assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
// 	assert.Contains(t, appErr.Error(), "Failed to update order status")
// 	assert.ErrorIs(t, appErr.Unwrap(), mockErr)

// 	mockOrderRepo.AssertExpectations(t)
// }
