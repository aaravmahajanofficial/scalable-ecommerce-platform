package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/google/uuid"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, product *models.Product) error
	GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error)
	UpdateProduct(ctx context.Context, product *models.Product) error
	ListProducts(ctx context.Context, page, size int) ([]*models.Product, int, error)
}

type productRepository struct {
	DB *sql.DB
}

func NewProductRepo(db *sql.DB) ProductRepository {
	return &productRepository{DB: db}
}

func (r *productRepository) CreateProduct(ctx context.Context, product *models.Product) error {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `INSERT INTO products (category_id, name, description, price, stock_quantity, sku, status)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  RETURNING id, created_at, updated_at
	`

	return r.DB.QueryRowContext(dbCtx, query, product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.SKU, product.Status).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

func (r *productRepository) GetProductByID(ctx context.Context, id uuid.UUID) (*models.Product, error) {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	product := &models.Product{}

	query := `
        SELECT p.id, p.category_id, p.name, p.description, p.price, 
               p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
               c.id, c.name, c.description
        FROM products p
        LEFT JOIN categories c ON p.category_id = c.id
        WHERE p.id = $1`

	var category models.Category

	err := r.DB.QueryRowContext(dbCtx, query, id).Scan(&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.SKU, &product.Status, &product.CreatedAt, &product.UpdatedAt, &category.ID, &category.Name, &category.Description)
	if err != nil {
		return nil, fmt.Errorf("querying database: %w", err)
	}

	product.Category = &category

	return product, nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *models.Product) error {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		UPDATE products SET category_id = $1, name = $2, description = $3, price = $4, stock_quantity = $5, status = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING updated_at
	`

	return r.DB.QueryRowContext(dbCtx, query, product.CategoryID, product.Name, product.Description, product.Price, product.StockQuantity, product.Status, product.ID).Scan(&product.UpdatedAt)
}

func (r *productRepository) ListProducts(ctx context.Context, page, size int) ([]*models.Product, int, error) {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	var total int

	countQuery := `SELECT COUNT(*) FROM products`

	err := r.DB.QueryRowContext(dbCtx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Offset
	offset := (page - 1) * size

	query := `
		SELECT p.id, p.category_id, p.name, p.description, p.price, 
		p.stock_quantity, p.sku, p.status, p.created_at, p.updated_at,
		c.id, c.name, c.description
		FROM products p
		LEFT JOIN categories c on p.category_id = c.id
		ORDER BY p.id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.DB.QueryContext(dbCtx, query, size, offset)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var products []*models.Product

	for rows.Next() {
		product := &models.Product{}
		category := &models.Category{}

		err := rows.Scan(&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.StockQuantity, &product.SKU, &product.Status, &product.CreatedAt, &product.UpdatedAt, &category.ID, &category.Name, &category.Description)
		if err != nil {
			return nil, 0, err
		}

		product.Category = category
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
