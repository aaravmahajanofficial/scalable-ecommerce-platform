package service

import (
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repository"
	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo *repository.OrderRepository
}

func NewOrderService(orderRepo *repository.OrderRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

func (s *OrderService) CreateOrder(req *models.CreateOrderRequest) (*models.Order, error) {

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

	err := s.orderRepo.CreateOrder(order)

	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil

}

func (s *OrderService) GetOrderById(id uuid.UUID) (*models.Order, error) {

	return s.orderRepo.GetOrderById(id)

}

func (s *OrderService) ListOrdersByCustomer(customerId uuid.UUID, page int, size int) ([]models.Order, int, error) {

	if page < 1 {
		page = 1
	}

	if size < 1 || size > 10 {
		size = 10
	}

	return s.orderRepo.ListOrdersByCustomer(customerId, page, size)

}

func (s *OrderService) UpdateOrderStatus(id uuid.UUID, status models.OrderStatus) error {

	// check if order exists or not
	_, err := s.orderRepo.GetOrderById(id)

	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	return s.orderRepo.UpdateOrderStatus(id, status)

}
