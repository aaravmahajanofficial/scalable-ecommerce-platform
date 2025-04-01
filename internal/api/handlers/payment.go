package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type PaymentHandler struct {
	paymentService service.PaymentService
	validator      *validator.Validate
}

func NewPaymentService(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService, validator: validator.New()}
}

func (h *PaymentHandler) CreatePayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !validateMethod(w, r, http.MethodPost) {
			return
		}

		// Decode the request body
		var req models.PaymentRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !validateStruct(w, h.validator, req) {
			return
		}

		// Call the payment service
		payment, err := h.paymentService.CreatePayment(r.Context(), &req)

		if err != nil {
			slog.Error("Error during product creation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Payment initiated successfully", slog.String("productId", fmt.Sprintf("%v", payment.Payment.ID)))
		response.WriteJson(w, http.StatusCreated, payment)

	}
}

func (h *PaymentHandler) GetPayment() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		if idStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("payment ID is required")))
			return
		}

		// Call the service
		payment, err := h.paymentService.GetPaymentByID(r.Context(), idStr)

		if err != nil {
			slog.Error("Error while accessing payment", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		response.WriteJson(w, http.StatusOK, payment)

	}
}

func (h *PaymentHandler) ListPayments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		customerIDStr := r.URL.Query().Get("customer_id")

		if customerIDStr == "" {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("customer ID is required")))
			return
		}

		// extract pagination parameters
		page, size := 1, 10

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err != nil && p > 0 {
				page = p
			}
		}

		if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
			if s, err := strconv.Atoi(sizeStr); err != nil && s > 0 && s <= 100 {
				size = s
			}
		}

		// Call the service
		payments, total, err := h.paymentService.ListPaymentsByCustomer(r.Context(), customerIDStr, page, size)

		if err != nil {
			slog.Error("Error while listing the payments for the customer", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		response.WriteJson(w, http.StatusOK, map[string]any{
			"Payments": payments,
			"Total":    total,
			"Page":     page,
			"Size":     size,
		})

	}
}

func (h *PaymentHandler) HandleStripeWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// read the payload/body

		payload, err := io.ReadAll(r.Body)

		if err != nil {
			slog.Error("Error reading webhook body", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("failed to read request body")))
			return
		}

		signature := r.Header.Get("Stripe-Signature")

		if signature == "" {
			slog.Error("Missing Stripe signature")
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("missing stripe signature")))
			return
		}

		// Call the service
		_, err = h.paymentService.ProcessWebhook(r.Context(), payload, signature)

		if err != nil {
			slog.Error("Error processing webhook: %w", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusOK, map[string]string{"status": "received"})
			return
		}

		response.WriteJson(w, http.StatusOK, map[string]string{"status": "success"})

	}
}
