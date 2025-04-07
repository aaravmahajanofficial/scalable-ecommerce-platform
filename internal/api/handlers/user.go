package handlers

import (
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userService service.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService, validator: validator.New()}
}

func (h *UserHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		var req models.RegisterRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		user, err := h.userService.Register(r.Context(), &req)
		if err != nil {
			logger.Error("User registration failed", slog.String("email", req.Email), slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("User registered", slog.String("userId", user.ID.String()))
		response.Success(w, http.StatusCreated, user)
	}
}

func (h *UserHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		// Decode the request body
		var req models.LoginRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		resp, err := h.userService.Login(r.Context(), &req)
		if err != nil {
			logger.Warn("Login attempt failed", slog.String("email", req.Email), slog.Any("error", err))
			response.Error(w, err)
			return
		}

		if !resp.Success {
			if resp.RetryAfter > 0 {
				logger.Warn("Too many login attempts", slog.String("email", req.Email))
				response.Error(w, errors.TooManyRequestsError("Too many login attempts").WithDetail("Please try again later"))
				return
			}
			logger.Warn("Invalid credentials provided", slog.String("email", req.Email))
			response.Error(w, errors.UnauthorizedError("Invalid email or password"))
			return
		}

		logger.Info("User logged in", slog.String("email", req.Email))
		response.Success(w, http.StatusOK, resp)
	}
}

func (h *UserHandler) Profile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		// Get user claims from context (set by middleware)
		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized access attempt: missing user claims in context")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))
		logger.Info("Attempting to fetch user profile")

		user, err := h.userService.GetUserByID(r.Context(), claims.UserID)
		if err != nil {
			logger.Warn("User not found", slog.String("userID", claims.UserID.String()))
			response.Error(w, err)
			return
		}

		logger.Info("User profile accessed", slog.String("userID", user.ID.String()))
		response.Success(w, http.StatusOK, user)
	}
}
