package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

var validate = validator.New()

func (h *UserHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if r.Method != http.MethodPost {
			slog.Warn("Invalid request method",
				slog.String("method", r.Method),
				slog.String("endpoint", r.URL.Path),
			)
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		// Decode the request body
		var req models.RegisterRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if errors.Is(err, io.EOF) {
			slog.Warn("Empty request body", slog.String("endpoint", r.URL.Path))
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("❌ Bad Request: request body cannot be empty")))
			return
		}

		if err != nil {
			slog.Error("Failed to decode request body",
				slog.String("error", err.Error()),
				slog.String("endpoint", r.URL.Path),
			)
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		if err := validate.Struct(req); err != nil {

			slog.Warn("User input validation failed",
				slog.String("endpoint", r.URL.Path),
				slog.String("error", err.Error()),
			)

			response.WriteJson(w, http.StatusBadRequest, response.ValidationError(err.(validator.ValidationErrors)))
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
		if r.Method != http.MethodPost {
			slog.Warn("Invalid request method",
				slog.String("method", r.Method),
				slog.String("endpoint", r.URL.Path),
			)
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		// Decode the request body
		var req models.LoginRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if errors.Is(err, io.EOF) {
			slog.Warn("Empty request body",
				slog.String("endpoint", r.URL.Path),
			)
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("❌ Bad Request: request body cannot be empty")))
			return
		}

		if err != nil {
			slog.Error("Failed to decode request body",
				slog.String("error", err.Error()),
				slog.String("endpoint", r.URL.Path),
			)
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		if err := validate.Struct(req); err != nil {

			slog.Warn("User input validation failed",
				slog.String("endpoint", r.URL.Path),
				slog.String("error", err.Error()),
			)

			response.WriteJson(w, http.StatusBadRequest, response.ValidationError(err.(validator.ValidationErrors)))
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
