package repository

import (
	"database/sql"
	"fmt"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"

	_ "github.com/lib/pq"
)

type Repositories struct {
	DB           *sql.DB
	User         *UserRepository
	Product      *ProductRepository
	Cart         *CartRepository
	Order        *OrderRepository
	Payment      *PaymentRepository
	Notification *NotificationRepository
}

func New(cfg *config.Config) (*Repositories, error) {

	db, err := sql.Open("postgres", cfg.Database.GetDSN())

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Test the connection to make sure DB is reachable
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repositories
	return &Repositories{
		DB:           db,
		User:         NewUserRepo(db),    // Initialize UserRepository
		Product:      NewProductRepo(db), // Initialize ProductRepository
		Cart:         NewCartRepo(db),    // Initialize CartRepository
		Order:        NewOrderRepository(db),
		Payment:      NewPaymentRepository(db),
		Notification: NewNotificationRepo(db)}, nil
}

func (r *Repositories) Close() error {
	return r.DB.Close()
}
