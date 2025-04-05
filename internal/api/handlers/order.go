package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type OrderHandler struct {
	orderService service.OrderService
	validator    *validator.Validate
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService, validator: validator.New()}
}

func (h *OrderHandler) CreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order creation attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		// Decode the request body, validate
		var req models.CreateOrderRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		order, err := h.orderService.CreateOrder(r.Context(), &req)
		if err != nil {
			slog.Error("Failed to create order",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Order created", slog.String("orderId", order.ID.String()), slog.String("userID", claims.UserID.String()))
		response.Success(w, http.StatusCreated, order)
	}
}

func (h *OrderHandler) GetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		id, err := utils.ParseID(r, "id")
		if err != nil {
			slog.Warn("Invalid product id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		// Call the service
		order, err := h.orderService.GetOrderById(r.Context(), id)
		if err != nil {
			slog.Error("Failed to get order",
				slog.String("orderId", id.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		if order.CustomerID != claims.UserID {
			slog.Warn("Attempted to access another user's order",
				slog.String("orderId", id.String()),
				slog.String("requesterId", claims.UserID.String()),
				slog.String("ownerId", order.CustomerID.String()))
			response.Error(w, errors.ForbiddenError("You don't have permission to access this order"))
			return
		}

		response.Success(w, http.StatusOK, order)
	}
}

func (h *OrderHandler) ListOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}
		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		// Call the service
		orders, total, err := h.orderService.ListOrdersByCustomer(r.Context(), claims.UserID, page, pageSize)
		if err != nil {
			slog.Error("Failed to list orders",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     orders,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}

func (h *OrderHandler) UpdateOrderStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		id, err := utils.ParseID(r, "id")
		if err != nil {
			slog.Warn("Invalid product id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		// Decode the request body
		var req models.UpdateOrderStatusRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		order, err := h.orderService.UpdateOrderStatus(r.Context(), id, req.Status)
		if err != nil {
			slog.Error("Failed to update order status",
				slog.String("orderId", id.String()),
				slog.String("status", string(req.Status)),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Order status updated",
			slog.String("orderId", id.String()),
			slog.String("status", string(req.Status)),
			slog.String("updatedBy", claims.UserID.String()))
		response.Success(w, http.StatusOK, order)
	}
}
