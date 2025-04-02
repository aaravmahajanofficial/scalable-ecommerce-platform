package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
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

		var req models.RegisterRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		user, err := h.userService.Register(r.Context(), &req)

		if err != nil {
			slog.Error("Error during user registration", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("User registered successfully", slog.String("userId", user.ID))
		response.WriteJson(w, http.StatusCreated, map[string]string{"id": user.ID})

	}
}

func (h *UserHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Decode the request body
		var req models.LoginRequest
		
		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		resp, err := h.userService.Login(r.Context(), &req)

		if err != nil {
			response.WriteJson(w, http.StatusUnauthorized, response.GeneralError(err))
			return
		}

		if !resp.Success {

			if resp.RetryAfter > 0 {
				response.WriteJson(w, http.StatusTooManyRequests, resp)
				return
			}

			response.WriteJson(w, http.StatusUnauthorized, resp)
			return
		}

		slog.Info("User logged in successfully", slog.String("email", req.Email))
		response.WriteJson(w, http.StatusOK, resp)

	}
}

func (h *UserHandler) Profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user claims from context (set by middleware)
		claims, ok := r.Context().Value("user").(*models.Claims)

		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := h.userService.GetUserByID(r.Context(), claims.UserID)

		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		response.WriteJson(w, http.StatusFound, user)
	}
}
