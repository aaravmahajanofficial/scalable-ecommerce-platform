package models

import (
	"time"

	"github.com/google/uuid"
)

type CartItem struct {
	ProductID  uuid.UUID `json:"product_id"`
	Quantity   int       `json:"quantity"`
	UnitPrice  float64   `json:"unit_price"`
	TotalPrice float64   `json:"total_price"`
}

type Cart struct {
	ID        uuid.UUID           `json:"id"`
	UserID    uuid.UUID           `json:"user_id"`
	Items     map[string]CartItem `json:"items"`
	Total     float64             `json:"total"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type AddItemRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity"   validate:"required,min=1"`
	UnitPrice float64   `json:"unit_price" validate:"required,min=0"`
}

type UpdateQuantityRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity"   validate:"required,min=0"`
}
