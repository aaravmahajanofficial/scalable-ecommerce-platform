package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
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

		logger := middleware.LoggerFromContext(r.Context())

		// Decode the request body
		var req models.CreateProductRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			return
		}

		logger.Info("Attempting to create product", slog.String("name", req.Name))

		// Call the register service
		product, err := h.productService.CreateProduct(r.Context(), &req)

		if err != nil {
			logger.Error("Error during product creation", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Product created successfully", slog.String("productId", product.ID.String()))
		response.Success(w, http.StatusCreated, product)
	}
}

func (h *ProductHandler) GetProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		id, err := utils.ParseID(r, "id")
		if err != nil {
			logger.Warn("Invalid product ID in path", slog.Any("error", err), slog.String("pathValue", r.PathValue("id")))
			response.Error(w, err)
			return
		}

		logger = logger.With(slog.String("productId", id.String()))
		logger.Info("Attempting to get product")

		product, err := h.productService.GetProductByID(r.Context(), id)
		if err != nil {
			logger.Warn("Failed to get product", slog.Any("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Product retrieved successfully")
		response.Success(w, http.StatusOK, product)
	}
}

func (h *ProductHandler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		id, err := utils.ParseID(r, "id")
		if err != nil {
			slog.Warn("Invalid product id", slog.String("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger = logger.With(slog.String("productId", id.String()))

		// Decode the request body
		var req models.UpdateProductRequest

		// Validate Input
		if !utils.ParseAndValidate(r, w, &req, h.validator) {
			logger.Warn("Invalid product update input")
			return
		}

		logger.Info("Attempting to update product")
		// Call the service
		product, err := h.productService.UpdateProduct(r.Context(), id, &req)

		if err != nil {
			logger.Error("Error during product update", slog.Any("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Product updated successfully")
		response.Success(w, http.StatusOK, product)
	}
}

// for eg: GET /products?page=1&page_size=10
func (h *ProductHandler) ListProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logger := middleware.LoggerFromContext(r.Context())

		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			page = 1
		}

		pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}

		logger = logger.With(slog.Int("page", page), slog.Int("pageSize", pageSize))

		products, total, err := h.productService.ListProducts(r.Context(), page, pageSize)
		if err != nil {
			logger.Error("Failed to fetch products", slog.Any("error", err.Error()))
			response.Error(w, err)
			return
		}

		logger.Info("Products listed successfully", slog.Int("count", len(products)), slog.Int("total", total))
		response.Success(w, http.StatusOK, models.PaginatedResponse{
			Data:     products,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		})
	}
}
