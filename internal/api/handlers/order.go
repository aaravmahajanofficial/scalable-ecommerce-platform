package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderService *service.OrderService
	validator    *validator.Validate
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService, validator: validator.New()}
}

func (h *OrderHandler) CreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !validateMethod(w, r, http.MethodPost) {
			return
		}

		// Decode the request body
		var req models.CreateOrderRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		// Call the service
		order, err := h.orderService.CreateOrder(r.Context(), &req)

		if err != nil {
			slog.Error("Error during order creation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Order created successfully", slog.String("productId", fmt.Sprintf("%v", order.ID)))
		response.WriteJson(w, http.StatusCreated, order)

	}
}

func (h *OrderHandler) GetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		if idStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("order ID is required")))
			return
		}

		id, err := uuid.Parse(idStr)

		if err != nil {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("invalid order ID")))
			return
		}

		// Call the service
		order, err := h.orderService.GetOrderById(r.Context(), id)

		if err != nil {
			slog.Error("Error while accessing order", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		response.WriteJson(w, http.StatusOK, order)

	}
}

func (h *OrderHandler) ListOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		customerIDStr := r.URL.Query().Get("customer_id")

		if customerIDStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("customer ID is required")))
			return
		}

		// Parse the customer ID
		customerID, err := uuid.Parse(customerIDStr)

		if err != nil {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("invalid customer ID")))
			return
		}

		// extract pagination parameters
		page, size := 1, 10

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
			if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
				size = s
			}
		}

		// Call the service
		orders, total, err := h.orderService.ListOrdersByCustomer(r.Context(), customerID, page, size)

		if err != nil {
			slog.Error("Error while listing the orders for the customer", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		response.WriteJson(w, http.StatusOK, map[string]any{
			"Orders": orders,
			"Total":  total,
			"Page":   page,
			"Size":   size,
		})

	}
}

func (h *OrderHandler) UpdateOrderStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		if idStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("order ID is required")))
			return
		}

		id, err := uuid.Parse(idStr)

		if err != nil {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("invalid order ID")))
			return
		}

		// Check for correct HTTP method
		if !validateMethod(w, r, http.MethodPatch) {
			return
		}

		// Decode the request body
		var req models.UpdateOrderStatusRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		// Call the service
		err = h.orderService.UpdateOrderStatus(r.Context(), id, req.Status)

		if err != nil {
			slog.Error("Error while updating the order status for order", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		response.WriteJson(w, http.StatusOK, map[string]string{"message": "Order status updated successfully"})

	}
}
