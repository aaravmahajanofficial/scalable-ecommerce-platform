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

// Register godoc
//	@Summary		Register a new user
//	@Description	Creates a new user account with the provided details.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		models.RegisterRequest	true	"User Registration Details"
//	@Success		201		{object}	models.User				"Successfully created user"
//	@Failure		400		{object}	response.ErrorResponse	"Validation error or invalid input"
//	@Failure		409		{object}	response.ErrorResponse	"User with email already exists"
//	@Failure		500		{object}	response.ErrorResponse	"Internal server error"
//	@Router			/users/register [post]
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

// Login godoc
//	@Summary		Log in a user
//	@Description	Authenticates a user and returns a JWT token upon successful login.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		models.LoginRequest		true	"User Login Credentials"
//	@Success		200			{object}	models.LoginResponse	"Successful login, includes JWT token"
//	@Failure		400			{object}	response.ErrorResponse	"Validation error or invalid input"
//	@Failure		401			{object}	response.ErrorResponse	"Invalid email or password"
//	@Failure		429			{object}	response.ErrorResponse	"Too many login attempts"
//	@Failure		500			{object}	response.ErrorResponse	"Internal server error"
//	@Router			/users/login [post]
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

// Profile godoc
//	@Summary		Get user profile
//	@Description	Retrieves the profile information for the currently authenticated user.
//	@Tags			Users
//	@Produce		json
//	@Success		200	{object}	models.User				"Successfully retrieved user profile"
//	@Failure		401	{object}	response.ErrorResponse	"Authentication required (invalid or missing token)"
//	@Failure		404	{object}	response.ErrorResponse	"User not found"
//	@Failure		500	{object}	response.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/users/profile [get]
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
