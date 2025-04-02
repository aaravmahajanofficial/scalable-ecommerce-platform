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

func New(cfg *config.Config) (*Repository, *UserRepository, *ProductRepository, *CartRepository, *OrderRepository, *PaymentRepository, *NotificationRepository, error) {

	db, err := sql.Open("postgres", cfg.Database.GetDSN())

	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Test the connection to make sure DB is reachable
	if err := db.Ping(); err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repositories
	postgresInstance := &Repository{DB: db}
	userRepo := NewUserRepo(db)       // Initialize UserRepository
	productRepo := NewProductRepo(db) // Initialize ProductRepository
	cartRepo := NewCartRepo(db)       // Initialize CartRepository
	orderRepo := NewOrderRepository(db)
	paymentRepo := NewPaymentRepository(db)
	notificationRepo := NewNotificationRepo(db)

	return postgresInstance, userRepo, productRepo, cartRepo, orderRepo, paymentRepo, notificationRepo, nil
}

func (p *Repository) Close() error {
	return p.DB.Close()
}
