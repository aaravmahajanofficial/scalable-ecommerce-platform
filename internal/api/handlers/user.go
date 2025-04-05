package handlers

import (
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
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
			slog.Error("User registration failed", slog.String("email", req.Email), slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("User registered", slog.String("userId", user.ID.String()))
		response.Success(w, http.StatusCreated, user)
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
			response.Error(w, err)
			return
		}

		if !resp.Success {
			if resp.RetryAfter > 0 {
				response.Error(w, errors.TooManyRequestsError("Too many login attempts").WithDetail("Please try again later"))
				return
			}

			response.Error(w, errors.UnauthorizedError("Invalid email or password"))
			return
		}

		slog.Info("User logged in", slog.String("email", req.Email))
		response.Success(w, http.StatusCreated, resp)
	}
}

func (h *UserHandler) Profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get user claims from context (set by middleware)
		claims, ok := r.Context().Value("user").(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		user, err := h.userService.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			slog.Warn("User not found", slog.String("userID", claims.UserID.String()))
			response.Error(w, err)
			return
		}

		slog.Info("User profile accessed", slog.String("userID", user.ID.String()))
		response.Success(w, http.StatusOK, user)
	}
}
