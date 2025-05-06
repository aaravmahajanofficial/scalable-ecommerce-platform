package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/google/uuid"
)

type CartRepository interface {
	CreateCart(ctx context.Context, cart *models.Cart) error
	GetCartByCustomerID(ctx context.Context, customerID uuid.UUID) (*models.Cart, error)
	UpdateCart(ctx context.Context, cart *models.Cart) error
}

type cartRepository struct {
	DB *sql.DB
}

func NewCartRepo(db *sql.DB) CartRepository {
	return &cartRepository{DB: db}
}

func (r *cartRepository) CreateCart(ctx context.Context, cart *models.Cart) error {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	itemsJSON, err := json.Marshal(cart.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal cart items: %w", err)
	}

	query := `
		INSERT INTO carts (id, user_id, items, created_at, updated_at)
		VALUES($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	return r.DB.QueryRowContext(dbCtx, query, cart.ID, cart.UserID, itemsJSON).Scan(&cart.ID, &cart.CreatedAt, &cart.UpdatedAt)
}

func (r *cartRepository) GetCartByCustomerID(ctx context.Context, customerID uuid.UUID) (*models.Cart, error) {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		SELECT id, user_id, items, created_at, updated_at
		FROM carts
		WHERE user_id = $1
	`

	cart := &models.Cart{}

	var itemsJSON []byte

	err := r.DB.QueryRowContext(dbCtx, query, customerID).Scan(&cart.ID, &cart.UserID, &itemsJSON, &cart.CreatedAt, &cart.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("querying database: %w", err)
	}

	if err := json.Unmarshal(itemsJSON, &cart.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart items: %w", err)
	}

	return cart, nil
}

func (r *cartRepository) UpdateCart(ctx context.Context, cart *models.Cart) error {
	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	itemsJSON, err := json.Marshal(cart.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal cart items: %w", err)
	}

	query := `
		UPDATE carts
		SET items = $1, total = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := r.DB.ExecContext(dbCtx, query, itemsJSON, cart.Total, time.Now(), cart.ID)
	if err != nil {
		return fmt.Errorf("failed to update the cart: %w", err)
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get updated rows: %w", err)
	}

	if updatedRows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
