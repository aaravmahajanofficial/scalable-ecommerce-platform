package service

import (
	"context"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
)

type ProductService interface {
	CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	UpdateProduct(ctx context.Context, id uuid.UUID, req *models.UpdateProductRequest) (*models.Product, error)
	ListProducts(ctx context.Context, page, pageSize int) ([]*models.Product, int, error)
}
type productService struct {
	repo repository.ProductRepository
}

func NewProductService(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {

	product := &models.Product{
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		SKU:           req.SKU,
		Status:        "active",
	}

	err := s.repo.CreateProduct(ctx, product)
	if err != nil {
		return nil, errors.DatabaseError("Failed to create product").WithError(err)
	}

	return product, nil
}

func (s *productService) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {

	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Product not found").WithError(err)
	}

	return product, nil
}

func (s *productService) UpdateProduct(ctx context.Context, id uuid.UUID, req *models.UpdateProductRequest) (*models.Product, error) {

	product, err := s.repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, errors.NotFoundError("Product not found").WithError(err)
	}

	if req.CategoryID != nil {
		product.CategoryID = *req.CategoryID
	}
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}
	if req.Status != nil {
		product.Status = *req.Status
	}

	err = s.repo.UpdateProduct(ctx, product)
	if err != nil {
		return nil, errors.DatabaseError("Failed to update product").WithError(err)
	}

	return product, err
}

// page means "page number requested"
// pageSize means "number of products to be displayed per page"
func (s *productService) ListProducts(ctx context.Context, page, pageSize int) ([]*models.Product, int, error) {

	products, total, err := s.repo.ListProducts(ctx, page, pageSize)
	if err != nil {
		return nil, 0, errors.DatabaseError("Failed to fetch products").WithError(err)
	}

	return products, total, nil
}
