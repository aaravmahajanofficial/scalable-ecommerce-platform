package repository

import (
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	_ "github.com/lib/pq"
)

type Repositories struct {
	DB           *sql.DB
	User         UserRepository
	Product      ProductRepository
	Cart         CartRepository
	Order        OrderRepository
	Payment      PaymentRepository
	Notification NotificationRepository
}

func New(cfg *config.Config) (*Repositories, error) {

	db, err := otelsql.Open("postgres", cfg.Database.GetDSN(),
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithAttributes(semconv.DBNamespace(cfg.Database.Name)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open instrumented database connection: %w", err)
	}

	// DB stats collector
	if err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
		semconv.DBNamespace(cfg.Database.Name),
	)); err != nil {
		return nil, fmt.Errorf("failed to register DB stats metrics: %w", err)
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
		User:         NewUserRepo(db),
		Product:      NewProductRepo(db),
		Cart:         NewCartRepo(db),
		Order:        NewOrderRepository(db),
		Payment:      NewPaymentRepository(db),
		Notification: NewNotificationRepo(db),
	}, nil
}

func (r *Repositories) Close() error {
	return r.DB.Close()
}
