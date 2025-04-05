package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

func (h *CartHandler) CreateCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		raw := r.Context().Value(middleware.UserContextKey)
		userId, ok := raw.(uuid.UUID)
		if !ok {
			response.WriteJson(w, http.StatusUnauthorized, "Unauthorized: User ID missing")
			return
		}

		// Call the cart service
		req, err := h.cartService.CreateCart(r.Context(), userId)

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		if err != nil {
			slog.Error("Error during cart creation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Cart created successfully", slog.String("cartId", fmt.Sprintf("%v", req.ID)), slog.String("userId", fmt.Sprintf("%v", userId)))
		response.WriteJson(w, http.StatusCreated, map[string]any{"id": req.ID})

	}
}

func (h *CartHandler) GetCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		raw := r.Context().Value(middleware.UserContextKey)
		userId, ok := raw.(uuid.UUID)
		if !ok {
			response.WriteJson(w, http.StatusUnauthorized, "Unauthorized: User ID missing")
			return
		}

		cart, err := h.cartService.GetCart(r.Context(), userId)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}

}

func (h *CartHandler) AddItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		raw := r.Context().Value(middleware.UserContextKey)
		userId, ok := raw.(uuid.UUID)
		if !ok {
			response.WriteJson(w, http.StatusUnauthorized, "Unauthorized: User ID missing")
			return
		}

		// decode the response body
		var req models.AddItemRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		cart, err := h.cartService.AddItem(r.Context(), userId, &req)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}
}

func (h *CartHandler) UpdateQuantity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		raw := r.Context().Value(middleware.UserContextKey)
		userId, ok := raw.(uuid.UUID)
		if !ok {
			response.WriteJson(w, http.StatusUnauthorized, "Unauthorized: User ID missing")
			return
		}

		var req models.UpdateQuantityRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		cart, err := h.cartService.UpdateQuantity(r.Context(), userId, &req)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}
}
