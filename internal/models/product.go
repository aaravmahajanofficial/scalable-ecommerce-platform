package models

import "time"

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Product struct {
	ID            int64     `json:"id"`
	CategoryID    int64     `json:"category_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	StockQuantity int64     `json:"stock_quantity"`
	SKU           string    `json:"sku"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Category      *Category `json:"category,omitempty"`
}

type CreateProductRequest struct {
	CategoryID    int64   `json:"category_id" validate:"required"`
	Name          string  `json:"name,omitempty" validate:"omitempty,min=3,max=200"`
	Description   string  `json:"description,omitempty"`
	Price         float64 `json:"price,omitempty" validate:"omitempty, gt=0"`
	StockQuantity int     `json:"stock_quantity" validate:"required,gte=0"`
	SKU           string  `json:"sku" validate:"required,min=3,max=50"`
}

type UpdateProductRequest struct {
	CategoryID    *int64   `json:"category_id,omitempty"`
	Name          *string  `json:"name,omitempty" validate:"omitempty,min=3,max=200"`
	Description   *string  `json:"description,omitempty"`
	Price         *float64 `json:"price,omitempty" validate:"omitempty,gt=0"`
	StockQuantity *int     `json:"stock_quantity,omitempty" validate:"omitempty,gte=0"`
	Status        *string  `json:"status,omitempty" validate:"omitempty,oneof=active inactive discontinued"`
}
