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

// CreateProduct godoc
//	@Summary		Create a new product
//	@Description	Adds a new product to the catalog. Requires authentication.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			product	body		models.CreateProductRequest	true	"Product Creation Details"
//	@Success		201		{object}	models.Product				"Successfully created product"
//	@Failure		400		{object}	response.ErrorResponse		"Validation error or invalid input"
//	@Failure		401		{object}	response.ErrorResponse		"Authentication required"
//	@Failure		500		{object}	response.ErrorResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/products [post]
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

// GetProduct godoc
//	@Summary		Get a product by ID
//	@Description	Retrieves details for a specific product using its ID. Requires authentication.
//	@Tags			Products
//	@Produce		json
//	@Param			id	path		string					true	"Product ID (UUID)"	Format(uuid)
//	@Success		200	{object}	models.Product			"Successfully retrieved product"
//	@Failure		400	{object}	response.ErrorResponse	"Invalid product ID format"
//	@Failure		401	{object}	response.ErrorResponse	"Authentication required"
//	@Failure		404	{object}	response.ErrorResponse	"Product not found"
//	@Failure		500	{object}	response.ErrorResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/products/{id} [get]
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

// UpdateProduct godoc
//	@Summary		Update a product by ID
//	@Description	Updates details for an existing product using its ID. Requires authentication.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Product ID (UUID)"	Format(uuid)
//	@Param			product	body		models.UpdateProductRequest	true	"Product Update Details"
//	@Success		200		{object}	models.Product				"Successfully updated product"
//	@Failure		400		{object}	response.ErrorResponse		"Invalid product ID format or validation error"
//	@Failure		401		{object}	response.ErrorResponse		"Authentication required"
//	@Failure		404		{object}	response.ErrorResponse		"Product not found"
//	@Failure		500		{object}	response.ErrorResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/products/{id} [put]
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

// ListProducts godoc
//	@Summary		List products with pagination
//	@Description	Retrieves a paginated list of available products. Requires authentication.
//	@Tags			Products
//	@Produce		json
//	@Param			page		query		int												false	"Page number for pagination (default: 1)"			minimum(1)
//	@Param			pageSize	query		int												false	"Number of items per page (default: 10, max: 100)"	minimum(1)	maximum(100)
//	@Success		200			{object}	models.PaginatedResponse{Data=[]models.Product}	"Successfully retrieved list of products"
//	@Failure		401			{object}	response.ErrorResponse							"Authentication required"
//	@Failure		500			{object}	response.ErrorResponse							"Internal server error"
//	@Security		BearerAuth
//	@Router			/products [get]
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
