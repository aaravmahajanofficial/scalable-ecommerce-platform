package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

type CartService struct {
	repo *repository.CartRepository
}

func NewCartService(repo *repository.CartRepository) *CartService {
	return &CartService{repo: repo}
}

func (s *CartService) CreateCart(ctx context.Context, userId string) (*models.Cart, error) {

	cart := &models.Cart{
		ID:        uuid.NewString(),
		UserID:    userId,
		Items:     make(map[string]models.CartItem),
		Total:     0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := s.repo.CreateCart(ctx, cart)

	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (s *CartService) GetCart(ctx context.Context, cartId string) (*models.Cart, error) {

	cart, err := s.repo.GetCart(ctx, cartId)

	if err != nil {
		return nil, err
	}

	return cart, err

}

func (s *CartService) AddItem(ctx context.Context, cartId string, req *models.AddItemRequest) (*models.Cart, error) {

	cart, err := s.repo.GetCart(ctx, cartId)

	if err != nil {
		return nil, err
	}

	item := models.CartItem{

		ProductID:  req.ProductID,
		Quantity:   req.Quantity,
		UnitPrice:  req.UnitPrice,
		TotalPrice: float64(req.Quantity) * req.UnitPrice,
	}

	cart.Items[req.ProductID] = item
	cart.UpdatedAt = time.Now()
	cart.Total = s.calculateTotal(cart.Items)

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil

}

func (s *CartService) UpdateQuantity(ctx context.Context, cartId string, req *models.UpdateQuantityRequest) (*models.Cart, error) {

	cart, err := s.repo.GetCart(ctx, cartId)

	if err != nil {
		return nil, err
	}

	item, exists := cart.Items[req.ProductID]

	if !exists {
		return nil, fmt.Errorf("item not found in the cart")
	}

	if req.Quantity == 0 {

		delete(cart.Items, req.ProductID)
	} else {

		item.Quantity = req.Quantity
		item.TotalPrice = item.UnitPrice * float64(item.Quantity)
		cart.Items[req.ProductID] = item

	}

	// update the cart

	cart.UpdatedAt = time.Now()
	cart.Total = s.calculateTotal(cart.Items)

	err = s.repo.UpdateCart(ctx, cart)

	if err != nil {
		return nil, err
	}

	return cart, nil

}

func (s *CartService) calculateTotal(items map[string]models.CartItem) float64 {

	var totalPrice float64

	for _, item := range items {
		totalPrice += item.TotalPrice
	}

	return totalPrice

}
