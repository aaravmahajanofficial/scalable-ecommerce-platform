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

// CreateOrder godoc
//	@Summary		Create a new order
//	@Description	Creates a new order from the user's current cart items and provided shipping details. Requires authentication.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			order	body		models.CreateOrderRequest	true	"Order Creation Details (includes shipping, uses current cart)"
//	@Success		201		{object}	models.Order				"Successfully created order"
//	@Failure		400		{object}	response.ErrorResponse		"Validation error, empty cart, or insufficient stock"
//	@Failure		401		{object}	response.ErrorResponse		"Authentication required"
//	@Failure		404		{object}	response.ErrorResponse		"Cart not found (should be created implicitly if needed)"
//	@Failure		500		{object}	response.ErrorResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/orders [post]
func (h *OrderHandler) CreateOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized order creation attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}
		logger = logger.With(slog.String("userID", claims.UserID.String()))

		// Decode the request body, validate
		var req models.CreateOrderRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid create order input")
			return
		}

		order, err := h.orderService.CreateOrder(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to create order", slog.Any("error", err))
			response.Error(w, err)
			return
		}

		logger.Info("Order created successfully", slog.String("orderId", order.ID.String()))
		response.Success(w, http.StatusCreated, order)
	}
}

// GetOrder godoc
//	@Summary		Get an order by ID
//	@Description	Retrieves details for a specific order placed by the authenticated user. Requires authentication.
//	@Tags			Orders
//	@Produce		json
//	@Param			id	path		string					true	"Order ID (UUID)"	Format(uuid)
//	@Success		200	{object}	models.Order			"Successfully retrieved order"
//	@Failure		400	{object}	response.ErrorResponse	"Invalid order ID format"
//	@Failure		401	{object}	response.ErrorResponse	"Authentication required"
//	@Failure		403	{object}	response.ErrorResponse	"Forbidden - User does not own this order"
//	@Failure		404	{object}	response.ErrorResponse	"Order not found"
//	@Failure		500	{object}	response.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/orders/{id} [get]
func (h *OrderHandler) GetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized order access attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		id, err := utils.ParseID(r, "id")
		if err != nil {
			logger.Warn("Invalid order id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger = logger.With(slog.String("orderId", id.String()))

		// Call the service
		order, err := h.orderService.GetOrderById(r.Context(), id)
		if err != nil {
			logger.Error("Failed to get order",
				slog.String("orderId", id.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		if order.CustomerID != claims.UserID {
			logger.Warn("Attempted to access another user's order",
				slog.String("requesterId", claims.UserID.String()),
				slog.String("ownerId", order.CustomerID.String()))
			response.Error(w, errors.ForbiddenError("You don't have permission to access this order"))
			return
		}

		logger.Info("Order retrieved successfully")
		response.Success(w, http.StatusOK, order)
	}
}

// ListOrders godoc
//	@Summary		List user's orders with pagination
//	@Description	Retrieves a paginated list of orders placed by the authenticated user. Requires authentication.
//	@Tags			Orders
//	@Produce		json
//	@Param			page		query		int												false	"Page number for pagination (default: 1)"			minimum(1)
//	@Param			pageSize	query		int												false	"Number of items per page (default: 10, max: 100)"	minimum(1)	maximum(100)
//	@Success		200			{object}	models.PaginatedResponse{Data=[]models.Order}	"Successfully retrieved list of orders"
//	@Failure		401			{object}	response.ErrorResponse							"Authentication required"
//	@Failure		500			{object}	response.ErrorResponse							"Internal server error"
//	@Security		BearerAuth
//	@Router			/orders [get]
func (h *OrderHandler) ListOrders() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized order list attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}
		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		logger = logger.With(slog.Int("page", page), slog.Int("pageSize", pageSize))

		// Call the service
		orders, total, err := h.orderService.ListOrdersByCustomer(r.Context(), claims.UserID, page, pageSize)
		if err != nil {
			logger.Error("Failed to list orders", slog.Any("error", err))
			response.Error(w, err)
			return
		}

		logger.Info("Orders listed successfully", slog.Int("count", len(orders)), slog.Int("total", total))
		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     orders,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}

// UpdateOrderStatus godoc
//	@Summary		Update order status (Admin/Internal)
//	@Description	Updates the status of a specific order. Requires authentication (potentially admin-level).
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Order ID (UUID)"	Format(uuid)
//	@Param			status	body		models.UpdateOrderStatusRequest	true	"New Order Status"
//	@Success		200		{object}	models.Order					"Successfully updated order status"
//	@Failure		400		{object}	response.ErrorResponse			"Invalid order ID format or invalid status value"
//	@Failure		401		{object}	response.ErrorResponse			"Authentication required"
//	@Failure		403		{object}	response.ErrorResponse			"Forbidden - Insufficient permissions to update status"	//	If	applicable
//	@Failure		404		{object}	response.ErrorResponse			"Order not found"
//	@Failure		500		{object}	response.ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized order status update attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("updaterUserID", claims.UserID.String()))

		id, err := utils.ParseID(r, "id")
		if err != nil {
			logger.Warn("Invalid order id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger = logger.With(slog.String("orderId", id.String()))

		// Decode the request body
		var req models.UpdateOrderStatusRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid update order status input")
			return
		}

		logger = logger.With(slog.String("newStatus", string(req.Status)))

		order, err := h.orderService.UpdateOrderStatus(r.Context(), id, req.Status)
		if err != nil {
			logger.Error("Failed to update order status", slog.Any("error", err))
			response.Error(w, err)
			return
		}

		logger.Info("Order status updated successfully")
		response.Success(w, http.StatusOK, order)
	}
}
