package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

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

		claims, ok := r.Context().Value("user").(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized notification creation attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		// Decode the request body
		var req models.EmailNotificationRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the payment service
		notification, err := h.notificationService.SendEmail(r.Context(), &req)
		if err != nil {
			slog.Error("Failed to create notification",
				slog.String("type", "Email"),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Notification created",
			slog.String("notificationId", notification.ID.String()),
			slog.String("createdBy", claims.UserID.String()))
		response.Success(w, http.StatusCreated, notification)
	}
}

func (h *NotificationHandler) ListNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value("user").(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}
		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		// Call the service
		notifications, err := h.notificationService.ListNotifications(r.Context(), page, pageSize)
		if err != nil {
			slog.Error("Failed to get user notifications",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, notifications)
	}
}
