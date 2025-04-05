package service

import (
	"context"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo   *repository.OrderRepository
	cartRepo    *repository.CartRepository
	productRepo *repository.ProductRepository
}

func NewOrderService(orderRepo *repository.OrderRepository, cartRepo *repository.CartRepository, productRepo *repository.ProductRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo, cartRepo: cartRepo, productRepo: productRepo}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {

	// Check if the cart exists or not
	cart, err := s.cartRepo.GetCartByCustomerID(ctx, req.CustomerID)
	if err != nil {
		return nil, errors.NotFoundError("Cart not found").WithError(err)
	}

	if len(cart.Items) == 0 {
		return nil, errors.BadRequestError("Cannot create order with empty cart")
	}

	// now check the availability of the product
	for _, item := range cart.Items {
		product, err := s.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, errors.NotFoundError("Product not found: " + item.ProductID.String()).WithError(err)
		}
		if product.StockQuantity < item.Quantity {
			return nil, errors.BadRequestError("Insufficient stock for product: " + item.ProductID.String())
		}

	}

	// calculate the order total
	var grossTotal float64

	for _, item := range req.Items {
		grossTotal += float64(item.Quantity) * item.UnitPrice
	}

	// assemble the order struct
	order := &models.Order{
		ID:              uuid.New(),
		CustomerID:      req.CustomerID,
		Status:          models.OrderStatusPending,
		TotalAmount:     grossTotal,
		PaymentStatus:   models.PaymentStatusPending,
		ShippingAddress: &req.ShippingAddress,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// now add the items

	var items []models.OrderItem

	for _, item := range req.Items {

		orderItem := models.OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			CreatedAt: time.Now(),
		}

		items = append(items, orderItem)

	}
	order.Items = items

	err = s.orderRepo.CreateOrder(ctx, order)
	if err != nil {
		return nil, errors.DatabaseError("Failed to create order").WithError(err)
	}

	for _, item := range cart.Items {
		product, _ := s.productRepo.GetProductByID(ctx, item.ProductID)
		product.StockQuantity -= item.Quantity

		err := s.productRepo.UpdateProduct(ctx, product)
		if err != nil {
			return nil, errors.DatabaseError("Failed to update inventory").WithError(err)
		}
	}

	return order, nil
}

func (s *OrderService) GetOrderById(ctx context.Context, id uuid.UUID) (*models.Order, error) {

	order, err := s.orderRepo.GetOrderById(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Order not found").WithError(err)
	}

	return order, nil
}

func (s *OrderService) ListOrdersByCustomer(ctx context.Context, customerId uuid.UUID, page int, size int) ([]models.Order, error) {

	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	orders, _, err := s.orderRepo.ListOrdersByCustomer(ctx, customerId, page, size)
	if err != nil {
		return nil, errors.DatabaseError("Failed to fetch orders").WithError(err)
	}

	return orders, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status models.OrderStatus) (*models.Order, error) {

	// check if order exists or not
	_, err := s.orderRepo.GetOrderById(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Order not found").WithError(err)
	}

	order, err := s.orderRepo.UpdateOrderStatus(ctx, id, status)
	if err != nil {
		return nil, errors.DatabaseError("Failed to update order status").WithError(err)
	}

	return order, nil
}
