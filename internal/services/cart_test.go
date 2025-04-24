package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateCart(t *testing.T) {
	mockRepo := repository.NewMockCartRepository()
	cartService := service.NewMockCartService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockRepo.On("CreateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(nil).Once()

		// Act
		cart, err := cartService.CreateCart(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, cart)
		assert.Equal(t, userID, cart.UserID)
		assert.NotEqual(t, uuid.Nil, cart.ID)
		assert.Empty(t, cart.Items)
		assert.Equal(t, float64(0), cart.Total)
		assert.WithinDuration(t, time.Now(), cart.CreatedAt, time.Second) // Check if created recently
		assert.WithinDuration(t, time.Now(), cart.UpdatedAt, time.Second) // Check if updated recently
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		// Arrange
		dbError := errors.New("database connection failed")
		mockRepo.On("CreateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		// Act
		cart, err := cartService.CreateCart(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to create cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetCart(t *testing.T) {
	mockRepo := repository.NewMockCartRepository()
	cartService := service.NewMockCartService(mockRepo)
	ctx := context.Background()
	customerID := uuid.New()
	existingCart := &models.Cart{
		ID:        uuid.New(),
		UserID:    customerID,
		Items:     make(map[string]models.CartItem),
		Total:     0,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}

	t.Run("Success - Cart Found", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()

		// Act
		cart, err := cartService.GetCart(ctx, customerID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, cart)
		assert.Equal(t, existingCart.ID, cart.ID)
		assert.Equal(t, customerID, cart.UserID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Cart Not Found", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		// Act
		cart, err := cartService.GetCart(ctx, customerID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Other Database Error", func(t *testing.T) {
		// Arrange
		dbError := errors.New("unexpected database error")
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, dbError).Once()

		// Act
		cart, err := cartService.GetCart(ctx, customerID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeInternal, appErr.Code)
		assert.Equal(t, "Failed to retrieve cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}

func TestAddItem(t *testing.T) {
	mockRepo := repository.NewMockCartRepository()
	cartService := service.NewMockCartService(mockRepo)
	ctx := context.Background()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	existingCart := &models.Cart{
		ID:     uuid.New(),
		UserID: customerID,
		Items:  make(map[string]models.CartItem),
		Total:  0,
	}

	addItemReq := &models.AddItemRequest{
		ProductID: productID1,
		Quantity:  2,
		UnitPrice: 10.50,
	}

	t.Run("Success - Add New Item", func(t *testing.T) {
		// Arrange:
		// 1. Expect GetCartByCustomerID to return the existing empty cart
		// 2. Expect UpdateCart to be called with the updated cart and return nil error
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			item, exists := cart.Items[productID1.String()]
			return exists &&
				item.ProductID == productID1 &&
				item.Quantity == 2 &&
				item.UnitPrice == 10.50 &&
				item.TotalPrice == 21.00 &&
				cart.Total == 21.00 &&
				cart.UserID == customerID
		})).Return(nil).Once()

		// Act
		updatedCart, err := cartService.AddItem(ctx, customerID, addItemReq)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		assert.Len(t, updatedCart.Items, 1)
		item, exists := updatedCart.Items[productID1.String()]
		assert.True(t, exists)
		assert.Equal(t, productID1, item.ProductID)
		assert.Equal(t, 2, item.Quantity)
		assert.Equal(t, 10.50, item.UnitPrice)
		assert.Equal(t, 21.00, item.TotalPrice)
		assert.Equal(t, 21.00, updatedCart.Total)
		assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
		mockRepo.AssertExpectations(t)

		existingCart.Items = make(map[string]models.CartItem)
		existingCart.Total = 0
	})

	t.Run("Success - Add Another Item (Update Existing Cart)", func(t *testing.T) {
		// Arrange
		existingCart.Items[productID1.String()] = models.CartItem{ProductID: productID1, Quantity: 1, UnitPrice: 5.0, TotalPrice: 5.0}
		existingCart.Total = 5.0
		addItemReq2 := &models.AddItemRequest{ProductID: productID2, Quantity: 3, UnitPrice: 2.0}

		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			item1, exists1 := cart.Items[productID1.String()]
			item2, exists2 := cart.Items[productID2.String()]
			expectedTotal := 5.0 + (3 * 2.0) // Old item total + new item total
			return exists1 && exists2 &&
				item1.Quantity == 1 && item1.TotalPrice == 5.0 &&
				item2.ProductID == productID2 && item2.Quantity == 3 && item2.UnitPrice == 2.0 && item2.TotalPrice == 6.0 &&
				cart.Total == expectedTotal &&
				len(cart.Items) == 2
		})).Return(nil).Once()

		// Act
		updatedCart, err := cartService.AddItem(ctx, customerID, addItemReq2)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		assert.Len(t, updatedCart.Items, 2)
		assert.Equal(t, 11.00, updatedCart.Total) // 5.0 + 6.0
		mockRepo.AssertExpectations(t)

		existingCart.Items = make(map[string]models.CartItem)
		existingCart.Total = 0
	})

	t.Run("Failure - Cart Not Found", func(t *testing.T) {
		// Arrange
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		// Act
		cart, err := cartService.AddItem(ctx, customerID, addItemReq)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		var appErr *appErrors.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart")
	})

	t.Run("Failure - Database Error on Update", func(t *testing.T) {
		// Arrange:
		// 1. GetCart succeeds
		// 2. UpdateCart fails
		dbError := errors.New("failed to write to db")
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(existingCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		// Act
		cart, err := cartService.AddItem(ctx, customerID, addItemReq)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to update cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)

		existingCart.Items = make(map[string]models.CartItem)
		existingCart.Total = 0
	})
}

func TestCartService_UpdateQuantity(t *testing.T) {
	mockRepo := repository.NewMockCartRepository()
	cartService := service.NewMockCartService(mockRepo)
	ctx := context.Background()
	customerID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New() // Non-existent product ID

	initialItem := models.CartItem{
		ProductID:  productID1,
		Quantity:   2,
		UnitPrice:  10.0,
		TotalPrice: 20.0,
	}
	initialCart := &models.Cart{
		ID:     uuid.New(),
		UserID: customerID,
		Items:  map[string]models.CartItem{productID1.String(): initialItem},
		Total:  20.0,
	}

	resetState := func() {
		mockRepo.ExpectedCalls = nil
		mockRepo.Calls = nil
		initialCart.Items = map[string]models.CartItem{
			productID1.String(): {
				ProductID:  productID1,
				Quantity:   2,
				UnitPrice:  10.0,
				TotalPrice: 20.0,
			},
		}
		initialCart.Total = 20.0
	}

	t.Run("Success - Update Existing Item Quantity", func(t *testing.T) {
		resetState()
		// Arrange
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 5}
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			item, exists := cart.Items[productID1.String()]
			return exists &&
				item.Quantity == 5 &&
				item.TotalPrice == 50.0 && // 5 * 10.0
				cart.Total == 50.0
		})).Return(nil).Once()

		// Act
		updatedCart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		item, exists := updatedCart.Items[productID1.String()]
		assert.True(t, exists)
		assert.Equal(t, 5, item.Quantity)
		assert.Equal(t, 50.0, item.TotalPrice)
		assert.Equal(t, 50.0, updatedCart.Total)
		assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Success - Remove Item (Quantity 0)", func(t *testing.T) {
		resetState()
		// Arrange:
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 0}
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.MatchedBy(func(cart *models.Cart) bool {
			_, exists := cart.Items[productID1.String()]
			return !exists && // Item should be removed
				cart.Total == 0.0 && // Total should be recalculated
				len(cart.Items) == 0
		})).Return(nil).Once()

		// Act
		updatedCart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, updatedCart)
		assert.Empty(t, updatedCart.Items) // Cart should be empty
		assert.Equal(t, 0.0, updatedCart.Total)
		assert.WithinDuration(t, time.Now(), updatedCart.UpdatedAt, time.Second)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure - Cart Not Found on Get", func(t *testing.T) {
		resetState()
		// Arrange
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 3}
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(nil, sql.ErrNoRows).Once()

		// Act
		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)
		assert.Equal(t, "Cart not found", appErr.Message)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Item Not Found in Cart", func(t *testing.T) {
		resetState()
		// Arrange
		updateReq := &models.UpdateQuantityRequest{ProductID: productID2, Quantity: 1}
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once() // Get succeeds

		// Act
		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart) // Cart should not be returned on error
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeBadRequest, appErr.Code)
		assert.Equal(t, "Item not found in the cart", appErr.Message)
		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "UpdateCart", mock.Anything, mock.Anything)
	})

	t.Run("Failure - Database Error on Update", func(t *testing.T) {
		resetState()
		// Arrange
		updateReq := &models.UpdateQuantityRequest{ProductID: productID1, Quantity: 4}
		dbError := errors.New("db write constraint failed")
		mockRepo.On("GetCartByCustomerID", ctx, customerID).Return(initialCart, nil).Once()
		mockRepo.On("UpdateCart", ctx, mock.AnythingOfType("*models.Cart")).Return(dbError).Once()

		// Act
		cart, err := cartService.UpdateQuantity(ctx, customerID, updateReq)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, cart)
		appErr, ok := err.(*appErrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)
		assert.Equal(t, "Failed to update cart", appErr.Message)
		assert.ErrorIs(t, err, dbError)
		mockRepo.AssertExpectations(t)
	})
}
