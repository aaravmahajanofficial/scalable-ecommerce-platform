package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

		// Check for correct HTTP method
		if !utils.ValidateMethod(w, r, http.MethodPost) {
			return
		}

		// Decode the request body
		var req models.EmailNotificationRequest
		if err := utils.DecodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !utils.ValidateStruct(w, h.validator, req) {
			return
		}

		// Call the payment service
		notification, err := h.notificationService.SendEmail(r.Context(), &req)

		if err != nil {
			slog.Error("Error sending email", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Email sent successfully", slog.String("notificationID: ", fmt.Sprintf("%v", notification.ID)))
		response.WriteJson(w, http.StatusCreated, notification)

	}
}

func (h *NotificationHandler) GetNotification() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !utils.ValidateMethod(w, r, http.MethodGet) {
			return
		}

		idStr := r.PathValue("id")

		if idStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("notification ID is required")))
			return
		}

		id, err := uuid.Parse(idStr)

		if err != nil {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("invalid notification ID")))
			return
		}

		// Call the payment service
		notification, err := h.notificationService.GetNotification(r.Context(), id)

		if err != nil {
			slog.Error("Failed to retrieve notification", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		if notification == nil {
			response.WriteJson(w, http.StatusNotFound, response.GeneralError(fmt.Errorf("notification not found")))
			return
		}

		response.WriteJson(w, http.StatusOK, notification)

	}
}

func (h *NotificationHandler) ListNotifications() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !utils.ValidateMethod(w, r, http.MethodGet) {
			return
		}

		// extract pagination parameters
		page, size := 1, 10

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
			if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
				size = s
			}
		}

		// Call the service
		notifications, err := h.notificationService.ListNotifications(r.Context(), page, size)

		if err != nil {
			slog.Error("Failed to retrieve notifications", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		if notifications == nil {
			response.WriteJson(w, http.StatusNotFound, response.GeneralError(fmt.Errorf("notifications not found")))
			return
		}

		response.WriteJson(w, http.StatusOK, map[string]any{
			"Notifications": notifications,
			"Total":         len(notifications),
			"Page":          page,
			"Size":          size,
		})

	}
}
