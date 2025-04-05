package handlers

import (
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
	productService service.ProductService
	validator      *validator.Validate
}

func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService, validator: validator.New()}
}

func (h *ProductHandler) CreateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Decode the request body
		var req models.CreateProductRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		product, err := h.productService.CreateProduct(r.Context(), &req)

		if err != nil {
			slog.Error("Error during product creation", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Product created successfully", slog.String("productId", product.ID.String()))
		response.Success(w, http.StatusCreated, product)
	}
}

func (h *ProductHandler) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id, err := utils.ParseID(r, "id")
		if err != nil {
			slog.Warn("Invalid product id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		product, err := h.productService.GetProductByID(r.Context(), id)
		if err != nil {
			slog.Warn("Failed to get product", slog.String("id", id.String()), slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, product)
	}
}

func (h *ProductHandler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id, err := utils.ParseID(r, "id")
		if err != nil {
			slog.Warn("Invalid product id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		// Decode the request body
		var req models.UpdateProductRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		// Call the register service
		product, err := h.productService.UpdateProduct(r.Context(), id, &req)

		if err != nil {
			slog.Error("Error during product update", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		slog.Info("Product updated successfully", slog.String("productId", product.ID.String()))
		response.Success(w, http.StatusOK, product)
	}
}

// for eg: GET /products?page=1&page_size=10
func (h *ProductHandler) ListProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		products, total, err := h.productService.ListProducts(r.Context(), page, pageSize)
		if err != nil {
			slog.Error("Failed to fetch products", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     products,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}
