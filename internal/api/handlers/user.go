package handlers

import (
	"errors"
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
		resp, err := h.userService.Register(r.Context(), &req)

		if err != nil {
			slog.Error("User registration failed", slog.String("email", req.Email), slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		slog.Info("User registered", slog.String("userId", resp.ID.String()))
		response.WriteJson(w, http.StatusCreated, resp)

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
			slog.Warn("Login failed", slog.String("email", req.Email), slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusUnauthorized, response.GeneralError(err))
			return
		}

		if !resp.Success {
			status := http.StatusUnauthorized
			if resp.RetryAfter > 0 {
				status = http.StatusTooManyRequests
			}

			response.WriteJson(w, status, resp)
			return
		}

		slog.Info("User logged in", slog.String("email", req.Email))
		response.WriteJson(w, http.StatusOK, resp)

	}
}

func (h *UserHandler) Profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user claims from context (set by middleware)
		claims, ok := r.Context().Value("user").(*models.Claims)

		if !ok {
			slog.Warn("Unauthorized access attempt")
			response.WriteJson(w, http.StatusNotFound, response.GeneralError(errors.New("unauthorized")))
			return
		}

		resp, err := h.userService.GetUserByID(r.Context(), claims.UserID)

		if err != nil {
			slog.Warn("User not found", slog.String("userID", claims.UserID.String()))
			response.WriteJson(w, http.StatusNotFound, response.GeneralError(errors.New("user not found")))
			return
		}

		slog.Info("User profile accessed", slog.String("userID", resp.ID.String()))
		response.WriteJson(w, http.StatusFound, resp)
	}
}
