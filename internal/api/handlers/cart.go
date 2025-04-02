package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
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

func (h *CartHandler) CreateCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// get the userId from the header
		userId := r.Header.Get("X-User-ID")

		if userId == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// validate the method
		if !validateMethod(w, r, http.MethodPost) {
			return
		}

		// Call the cart service
		cart, err := h.cartService.CreateCart(r.Context(), userId)

		if err != nil {
			slog.Error("Error during cart creation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Cart created successfully", slog.String("cartId", fmt.Sprintf("%v", cart.ID)), slog.String("userId", fmt.Sprintf("%v", userId)))
		response.WriteJson(w, http.StatusCreated, map[string]string{"id": cart.ID})

	}
}

func (h *CartHandler) GetCart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cartID := r.PathValue("id")

		if cartID == "" {

			response.WriteJson(w, http.StatusBadRequest, "Cart ID is required")
			return

		}

		cart, err := h.cartService.GetCart(r.Context(), cartID)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}

}

func (h *CartHandler) AddItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cartID := r.PathValue("id")

		if cartID == "" {
			response.WriteJson(w, http.StatusBadRequest, "Cart ID is required")
			return
		}

		// decode the response body
		var req models.AddItemRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		cart, err := h.cartService.AddItem(r.Context(), cartID, &req)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}
}

func (h *CartHandler) UpdateQuantity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		cartID := r.PathValue("id")

		if cartID == "" {
			response.WriteJson(w, http.StatusBadRequest, "Cart ID is required")
			return
		}

		var req models.UpdateQuantityRequest

		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		cart, err := h.cartService.UpdateQuantity(r.Context(), cartID, &req)

		if err != nil {
			response.WriteJson(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.WriteJson(w, http.StatusOK, cart)

	}
}
