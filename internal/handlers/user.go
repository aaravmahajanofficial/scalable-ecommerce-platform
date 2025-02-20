package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userService *service.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService, validator: validator.New()}
}

func (h *UserHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !validateMethod(w, r) {
			return
		}

		// Decode the request body
		var req models.RegisterRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		// Call the register service
		user, err := h.userService.Register(&req)

		if err != nil {
			slog.Error("Error during user registration", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("An unexpected error occurred")))
			return
		}

		slog.Info("User registered successfully", slog.String("userId", user.ID))
		response.WriteJson(w, http.StatusCreated, map[string]string{"id": user.ID})

	}
}

func (h *UserHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !validateMethod(w, r) {
			return
		}

		// Decode the request body
		var req models.LoginRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		// Call the register service
		token, err := h.userService.Login(&req)

		if err != nil {
			response.WriteJson(w, http.StatusUnauthorized, response.GeneralError(err))
			return
		}

		slog.Info("User logged in successfully", slog.String("email", req.Email))
		response.WriteJson(w, http.StatusOK, map[string]string{"token": token})

	}
}

func (h *UserHandler) Profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !validateMethod(w, r) {
			return
		}

		// Get user claims from context (set by middleware)
		claims, ok := r.Context().Value("user").(*models.Claims)

		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := h.userService.GetUserByID(claims.UserID)

		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		response.WriteJson(w, http.StatusFound, user)
	}
}
