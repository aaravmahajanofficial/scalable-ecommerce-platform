package repository

import (
	"database/sql"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type CartRepository struct {
	DB *sql.DB
}

func NewCartRepo(db *sql.DB) *CartRepository {
	return &CartRepository{DB: db}
}

func (r *CartRepository) CreateCart(cart *models.Cart) error {

	query := `
		INSERT INTO carts (id, user_id, items, created_at, updated_at)
		VALUES($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	_, err := r.DB.Exec(query, cart.ID, cart.UserID, cart.Items, cart.CreatedAt, cart.UpdatedAt)

	return err
}

func (r *CartRepository) GetCart(cartID string) (*models.Cart, error) {

	query := `
		SELECT id, user_id, items, created_at, updated_at
		FROM carts
		WHERE id = $1
	`

	cart := &models.Cart{}

	err := r.DB.QueryRow(query, cartID).Scan(&cart.ID, &cart.UserID, &cart.Items, &cart.CreatedAt, &cart.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return cart, nil

}

func (r *CartRepository) UpdateCart(cart *models.Cart) error {

	query := `
		UPDATE carts
		SET items = $1, total = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := r.DB.Exec(query, cart.ID, cart.Total, time.Now(), cart.ID)

	return err

}
