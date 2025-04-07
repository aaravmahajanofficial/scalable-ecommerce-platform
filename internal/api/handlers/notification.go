package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type NotificationHandler struct {
	notificationService service.NotificationService
	validator           *validator.Validate
}

func NewNotificationHandler(notificationService service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		validator:           validator.New(),
	}
}

func (h *NotificationHandler) SendEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized notification creation attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		// Decode the request body
		var req models.EmailNotificationRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid notification input")
			return
		}

		logger.Info("Attempting to send email notification")
		// Call the payment service
		notification, err := h.notificationService.SendEmail(r.Context(), &req)
		if err != nil {
			logger.Error("Failed to create notification",
				slog.String("type", "Email"),
				slog.Any("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Notification created successfully",
			slog.String("notificationId", notification.ID.String()))
		response.Success(w, http.StatusCreated, notification)
	}
}

func (h *NotificationHandler) ListNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		if !ok {
			logger.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		logger = logger.With(slog.String("userID", claims.UserID.String()))

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}
		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		logger = logger.With(slog.Int("page", page), slog.Int("pageSize", pageSize))
		logger.Info("Attempting to list notifications")
		// Call the service
		notifications, total, err := h.notificationService.ListNotifications(r.Context(), page, pageSize)
		if err != nil {
			logger.Error("Failed to get user notifications",
				slog.Any("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Notifications listed successfully", slog.Int("count", len(notifications)), slog.Int("total", total))
		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     notifications,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}
