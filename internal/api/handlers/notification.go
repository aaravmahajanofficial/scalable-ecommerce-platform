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

// SendEmail godoc
//	@Summary		Send an email notification (Admin/Internal)
//	@Description	Creates and sends an email notification record. This might be an admin-triggered action or for specific internal purposes. Requires authentication.
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			notification	body		models.EmailNotificationRequest	true	"Email Notification Details (Recipient User ID, Subject, Body)"
//	@Success		201				{object}	models.Notification				"Successfully created and potentially queued email notification"
//	@Failure		400				{object}	response.ErrorResponse			"Validation error or invalid input"
//	@Failure		401				{object}	response.ErrorResponse			"Authentication required"
//	@Failure		403				{object}	response.ErrorResponse			"Forbidden - Insufficient permissions"	//	If	restricted
//	@Failure		404				{object}	response.ErrorResponse			"Recipient User not found"
//	@Failure		500				{object}	response.ErrorResponse			"Internal server error or email sending provider error"
//	@Security		BearerAuth
//	@Router			/notifications/email [post]
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

// ListNotifications godoc
//	@Summary		List notifications for the user
//	@Description	Retrieves a paginated list of notifications for the authenticated user. Requires authentication.
//	@Tags			Notifications
//	@Produce		json
//	@Param			page		query		int														false	"Page number for pagination (default: 1)"			minimum(1)
//	@Param			pageSize	query		int														false	"Number of items per page (default: 10, max: 100)"	minimum(1)	maximum(100)
//	@Success		200			{object}	models.PaginatedResponse{Data=[]models.Notification}	"Successfully retrieved list of notifications"
//	@Failure		401			{object}	response.ErrorResponse									"Authentication required"
//	@Failure		500			{object}	response.ErrorResponse									"Internal server error"
//	@Security		BearerAuth
//	@Router			/notifications [get]
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
