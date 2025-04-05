package handlers

import (
	"io"
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

type PaymentHandler struct {
	paymentService service.PaymentService
	validator      *validator.Validate
}

func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, validator: validator.New()}
}

func (h *PaymentHandler) CreatePayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value("user").(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		// Decode the request body
		var req models.PaymentRequest
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		if req.CustomerID != claims.UserID.String() {
			slog.Warn("User attempted to pay for another user's order",
				slog.String("requesterId", claims.UserID.String()))
			response.Error(w, errors.ForbiddenError("You can only make payments for your own orders"))
			return
		}

		// Call the payment service
		payment, err := h.paymentService.CreatePayment(r.Context(), &req)
		if err != nil {
			slog.Error("Failed to initiate payment",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Payment initiated",
			slog.String("paymentId", payment.Payment.ID),
			slog.String("userId", claims.UserID.String()))
		response.Success(w, http.StatusOK, payment)
	}
}

func (h *PaymentHandler) GetPayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		claims, ok := r.Context().Value("user").(*models.Claims)
		if !ok {
			slog.Warn("Unauthorized order access attempt")
			response.Error(w, errors.UnauthorizedError("Authentication required"))
			return
		}

		idStr := r.PathValue("id")
		if idStr == "" {
			response.Error(w, errors.BadRequestError("Payment ID is required"))
			return
		}

		// Call the service
		payment, err := h.paymentService.GetPaymentByID(r.Context(), idStr)
		if err != nil {
			slog.Error("Failed to get payment",
				slog.String("paymentId", idStr),
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, payment)
	}
}

func (h *PaymentHandler) ListPayments() http.HandlerFunc {
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
		payments, err := h.paymentService.ListPaymentsByCustomer(r.Context(), claims.UserID.String(), page, pageSize)
		if err != nil {
			slog.Error("Failed to list user payments",
				slog.String("userId", claims.UserID.String()),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"payments": payments,
			"total":    len(payments),
			"page":     page,
			"pageSize": pageSize,
		})
	}
}

func (h *PaymentHandler) HandleStripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// read the payload/body

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Error reading webhook body", slog.String("error", err.Error()))
			response.Error(w, errors.BadRequestError("Failed to read request body"))
			return
		}

		signature := r.Header.Get("Stripe-Signature")
		if signature == "" {
			slog.Error("Missing Stripe signature")
			response.Error(w, errors.BadRequestError("Stripe Signature is required").WithError(err))
			return
		}

		// Call the service
		event, err := h.paymentService.ProcessWebhook(r.Context(), payload, signature)
		if err != nil {
			slog.Error("Failed to process payment webhook",
				slog.String("paymentId", event.ID),
				slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Payment webhook processed",
			slog.String("paymentId", event.ID))
		response.Success(w, http.StatusOK, map[string]bool{"success": true})
	}
}
