package models

import "time"

type CartItem struct {
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

type Cart struct {
	ID        string              `json:"id"`
	UserID    string              `json:"user_id"`
	Items     map[string]CartItem `json:"items"`
	Total     float64             `json:"total"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type AddItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `jsoin:"quantity" validate:"required,min=1"`
}

type UpdateQuantityRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `jsoin:"quantity" validate:"required,min=0"`
}
