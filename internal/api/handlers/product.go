package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

type ProductHandler struct {
	productService *service.ProductService
	validator      *validator.Validate
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService, validator: validator.New()}
}

func (h *ProductHandler) CreateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if !utils.ValidateMethod(w, r, http.MethodPost) {
			return
		}

		// Decode the request body
		var req models.CreateProductRequest
		if err := utils.DecodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !utils.ValidateStruct(w, h.validator, req) {
			return
		}

		// Call the register service
		product, err := h.productService.CreateProduct(r.Context(), &req)

		if err != nil {
			slog.Error("Error during product creation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Product created successfully", slog.String("productId", fmt.Sprintf("%v", product.ID)))
		response.WriteJson(w, http.StatusCreated, product)

	}
}

func (h *ProductHandler) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")

		id, err := strconv.ParseInt(idStr, 10, 64)

		if err != nil {
			http.Error(w, "Invalid product id", http.StatusBadRequest)
			return
		}

		product, err := h.productService.GetProductByID(r.Context(), id)

		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		response.WriteJson(w, http.StatusCreated, product)

	}
}

func (h *ProductHandler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)

		if err != nil {
			http.Error(w, "Invalid product id", http.StatusBadRequest)
			return
		}

		// Decode the request body
		var req models.UpdateProductRequest
		if err := utils.DecodeJSONBody(w, r, &req); err != nil {
			return
		}

		// Validate Input
		if !utils.ValidateStruct(w, h.validator, req) {
			return
		}

		// Call the register service
		product, err := h.productService.UpdateProduct(r.Context(), id, &req)

		if err != nil {
			slog.Error("Error during product updation", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("an unexpected error occurred")))
			return
		}

		slog.Info("Product updated successfully", slog.String("productId", fmt.Sprintf("%v", product.ID)))
		response.WriteJson(w, http.StatusOK, product)

	}
}

// for eg: GET /products?page=1&page_size=10
func (h *ProductHandler) ListProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

		products, err := h.productService.ListProducts(r.Context(), page, pageSize)

		if err != nil {
			slog.Error("Failed to fetch products", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(fmt.Errorf("failed to fetch products")))
			return
		}

		response.WriteJson(w, http.StatusOK, products)

	}
}
