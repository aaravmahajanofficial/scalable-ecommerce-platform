package models

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

type PaymentStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipping  OrderStatus = "shipping"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"

	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusAuthorized PaymentStatus = "authorized"
	PaymentStatusPaid       PaymentStatus = "paid"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

type Address struct {
	Street     string `json:"street" validate:"required"`
	City       string `json:"city" validate:"required"`
	State      string `json:"state" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Country    string `json:"country" validate:"required,iso3166_1_alpha2"`
}

type OrderItem struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id" validate:"required"`
	Quantity  int       `json:"quantity" validate:"required,min=1"`
	UnitPrice float64   `json:"unit_price" validate:"required,gte=0"`
	CreatedAt time.Time `json:"created_at"`
}

type Order struct {
	ID              uuid.UUID     `json:"id"`
	CustomerID      uuid.UUID     `json:"customer_id" validate:"required"`
	Status          OrderStatus   `json:"status"`
	TotalAmount     float64       `json:"total_amount"`
	PaymentStatus   PaymentStatus `json:"payment_status"`
	PaymentIntentID string        `json:"payment_intent_id,omitempty"`
	ShippingAddress *Address      `json:"shipping_address" validate:"required"`
	Items           []OrderItem   `json:"items" validate:"required,min=1,dive"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type CreateOrderRequest struct {
	CustomerID      uuid.UUID   `json:"customer_id" validate:"required"`
	Items           []OrderItem `json:"items" validate:"required,min=1,dive"`
	ShippingAddress Address     `json:"shipping_address" validate:"required"`
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status" validate:"required,oneof=pending confirmed shipping delivered cancelled"`
}

type OrderResponse struct {
	Order *Order `json:"order"`
}

type OrderHistoryResponse struct {
	Orders []Order `json:"orders"`
	Total  int     `json:"total"`
	Page   int     `json:"page"`
	Size   int     `json:"size"`
}

type PaymentIntent struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"`
}

type CreatePaymentRequest struct {
	OrderID uuid.UUID `json:"order_id" validate:"required"`
}

type PaymentResponse struct {
	PaymentIntent *PaymentIntent `json:"payment_intent"`
	ClientSecret  string         `json:"client_secret"`
}
