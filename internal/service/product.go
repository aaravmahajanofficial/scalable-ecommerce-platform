package service

import (
	"context"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repository"
)

type ProductService struct {
	repo *repository.ProductRepository
}

func NewProductService(repo *repository.ProductRepository) *ProductService {
	return &ProductService{repo: repo}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *models.CreateProductRequest) (*models.Product, error) {

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
		return nil, err
	}

	return product, nil

}

func (s *ProductService) GetProductByID(ctx context.Context, id int64) (*models.Product, error) {

	return s.repo.GetProductByID(ctx, id)

}

func (s *ProductService) UpdateProduct(ctx context.Context, id int64, req *models.UpdateProductRequest) (*models.Product, error) {

	product, err := s.repo.GetProductByID(ctx, id)

	if err != nil {
		return nil, err
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
		return nil, err
	}

	return product, err

}

// page means "page number requested"
// pageSize means "number of products to be displayed per page"
func (s *ProductService) ListProducts(ctx context.Context, page, pageSize int) ([]*models.Product, error) {

	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 10

	}

	offset := (page - 1) * pageSize

	return s.repo.ListProducts(ctx, offset, pageSize)

}
