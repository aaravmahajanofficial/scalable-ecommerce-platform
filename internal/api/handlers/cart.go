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
	cartService *service.CartService
	validator   *validator.Validate
}

func NewCartHandler(service *service.CartService) *CartHandler {
	return &CartHandler{
		cartService: service,
		validator:   validator.New(),
	}
}

func (h *CartHandler) GetCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized cart access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		cart, err := h.cartService.GetCart(r.Context(), claims.UserID)
		if err != nil {
			slog.Error("Failed to get cart",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, cart)
	}
}

func (h *CartHandler) AddItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized cart access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		_, err := h.cartService.GetCart(r.Context(), claims.UserID)
		if err != nil {
			if appErr, ok := errors.IsAppError(err); ok && appErr.Code == errors.ErrCodeNotFound {
				// cart not found, create it!
				_, err := h.cartService.CreateCart(r.Context(), claims.UserID)
				if err != nil {
					slog.Error("Failed to create cart",
						slog.String("userId", claims.UserID.String()),
						slog.String("error", err.Error()))
					response.Error(w, err)
					return
				}
			} else {
				slog.Error("Failed to check cart existence",
					slog.String("userId", claims.UserID.String()),
					slog.String("error", err.Error()))
				response.Error(w, err)
				return
			}
		}

		// decode the response body
		var req models.AddItemRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		cart, err := h.cartService.AddItem(r.Context(), claims.UserID, &req)
		if err != nil {
			slog.Error("Failed to add item to cart",
				slog.String("userId", claims.UserID.String()),
				slog.String("productId", req.ProductID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Item added to cart",
			slog.String("userId", claims.UserID.String()),
			slog.String("productId", req.ProductID.String()))
		response.Success(w, http.StatusOK, cart)
	}
}

func (h *CartHandler) UpdateQuantity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized cart access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		var req models.UpdateQuantityRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		cart, err := h.cartService.UpdateQuantity(r.Context(), claims.UserID, &req)
		if err != nil {
			slog.Error("Failed to update cart item",
				slog.String("userId", claims.UserID.String()),
				slog.String("itemId", req.ProductID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Cart item updated",
			slog.String("userId", claims.UserID.String()),
			slog.String("itemId", req.ProductID.String()))
		response.Success(w, http.StatusOK, cart)
	}
}
