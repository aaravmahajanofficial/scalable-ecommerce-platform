package repository

import (
	"database/sql"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"

	_ "github.com/lib/pq"
)

type Repository struct {
	DB *sql.DB
}

func New(cfg *config.Config) (*Repository, *UserRepository, *ProductRepository, *CartRepository, error) {

	db, err := sql.Open("postgres", cfg.Database.GetDSN())

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection to make sure DB is reachable
	if err := db.Ping(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	// Initialize database schema
	if err := initSchema(db); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Initialize repositories
	postgresInstance := &Repository{DB: db}
	userRepo := NewUserRepo(db)       // Initialize UserRepository
	productRepo := NewProductRepo(db) // Initialize ProductRepository
	cartRepo := NewCartRepo(db)       // Initialize CartRepository

	return postgresInstance, userRepo, productRepo, cartRepo, nil
}

func (p *Repository) Close() error {
	return p.DB.Close()
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW() NOT NULL,
		updated_at TIMESTAMP DEFAULT NOW() NOT NULL
	);

	CREATE TABLE IF NOT EXISTS categories (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS products (
		id SERIAL PRIMARY KEY,
		category_id INTEGER REFERENCES categories(id),
		name VARCHAR(200) NOT NULL,
		description TEXT,
		price DECIMAL(10,2) NOT NULL,
		stock_quantity INTEGER NOT NULL DEFAULT 0,
		sku VARCHAR(50) UNIQUE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, discontinued
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS carts (
            id UUID PRIMARY KEY,
            user_id VARCHAR(255) NOT NULL,
            items JSONB NOT NULL DEFAULT '{}'::jsonb,
            total DECIMAL(10,2) NOT NULL DEFAULT 0.00,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to execute schema creation: %w", err)
	}

	return nil
}
