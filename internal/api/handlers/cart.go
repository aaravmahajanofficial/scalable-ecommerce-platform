package handlers

import (
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type CartHandler struct {
	cartService service.CartService
	validator   *validator.Validate
}

func NewCartHandler(service service.CartService) *CartHandler {
	return &CartHandler{
		cartService: service,
		validator:   validator.New(),
	}
}

// GetCart godoc
//
//	@Summary		Get the user's shopping cart
//	@Description	Retrieves the current shopping cart contents for the authenticated user. Creates a cart if one doesn't exist.
//	@Tags			Cart
//	@Produce		json
//	@Success		200	{object}	models.Cart				"Successfully retrieved or created cart"
//	@Failure		401	{object}	response.ErrorResponse	"Authentication required"
//	@Failure		500	{object}	response.ErrorResponse	"Internal server error (e.g., failed to create cart)"
//	@Security		BearerAuth
//	@Router			/carts [get]
func (h *CartHandler) GetCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized cart access attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))
		logger.Info("Attempting to get cart")

		cart, err := h.cartService.GetCart(r.Context(), claims.UserID)
		if err != nil {
			logger.Error("Failed to get cart", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Cart retrieved successfully")
		response.Success(w, http.StatusOK, cart)
	}
}

// AddItem godoc
//
//	@Summary		Add an item to the cart
//	@Description	Adds a specified quantity of a product to the authenticated user's shopping cart. Creates cart if needed.
//	@Tags			Cart
//	@Accept			json
//	@Produce		json
//	@Param			item	body		models.AddItemRequest	true	"Item details (Product ID and Quantity)"
//	@Success		200		{object}	models.Cart				"Item successfully added/updated in cart"
//	@Failure		400		{object}	response.ErrorResponse	"Validation error or invalid product ID/quantity"
//	@Failure		401		{object}	response.ErrorResponse	"Authentication required"
//	@Failure		404		{object}	response.ErrorResponse	"Product not found"
//	@Failure		500		{object}	response.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/carts/items [post]
func (h *CartHandler) AddItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized cart add item attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))
		logger.Info("Checking for existing cart before adding item")

		_, err := h.cartService.GetCart(r.Context(), claims.UserID)
		if err != nil {
			if appErr, ok := errors.IsAppError(err); ok && appErr.Code == errors.ErrCodeNotFound {
				logger.Info("Cart not found, attempting to create one")
				// cart not found, create it!
				_, err := h.cartService.CreateCart(r.Context(), claims.UserID)
				if err != nil {
					logger.Error("Failed to create cart automatically", slog.Any("error", err))
					response.Error(w, err)

					return
				}

				logger.Info("Cart created successfully")
			} else {
				logger.Error("Failed to check cart existence before adding item", slog.Any("error", err))
				response.Error(w, err)

				return
			}
		}

		// decode the response body
		var req models.AddItemRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid add item input")

			return
		}

		logger = logger.With(slog.String("productID", req.ProductID.String()), slog.Int("quantity", req.Quantity))
		logger.Info("Attempting to add item to cart")

		cart, err := h.cartService.AddItem(r.Context(), claims.UserID, &req)
		if err != nil {
			logger.Error("Failed to add item to cart", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Item added to cart successfully")
		response.Success(w, http.StatusOK, cart)
	}
}

// UpdateQuantity godoc
//
//	@Summary		Update item quantity in the cart
//	@Description	Updates the quantity of a specific item in the authenticated user's shopping cart.
//	@Tags			Cart
//	@Accept			json
//	@Produce		json
//	@Param			item	body		models.UpdateQuantityRequest	true	"Item details (Product ID and new Quantity)"
//	@Success		200		{object}	models.Cart						"Quantity successfully updated"
//	@Failure		400		{object}	response.ErrorResponse			"Validation error or invalid product ID/quantity"
//	@Failure		401		{object}	response.ErrorResponse			"Authentication required"
//	@Failure		404		{object}	response.ErrorResponse			"Cart or item not found"
//	@Failure		500		{object}	response.ErrorResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/carts/items [put]
func (h *CartHandler) UpdateQuantity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized cart update quantity attempt: missing user claims")
			response.Error(w, errors.UnauthorizedError("Authentication required"))

			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		var req models.UpdateQuantityRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid update quantity input")

			return
		}

		logger = middleware.LoggerFromContext(r.Context()).
			With(slog.String("userID", claims.UserID.String())).
			With(slog.String("productID", req.ProductID.String()), slog.Int("newQuantity", req.Quantity))
		logger.Info("Attempting to update cart item quantity")

		cart, err := h.cartService.UpdateQuantity(r.Context(), claims.UserID, &req)
		if err != nil {
			logger.Error("Failed to update cart item quantity", slog.Any("error", err))
			response.Error(w, err)

			return
		}

		logger.Info("Cart item quantity updated successfully")
		response.Success(w, http.StatusOK, cart)
	}
}
