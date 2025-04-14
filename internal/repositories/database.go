package repository

import (
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/cache"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/redis/go-redis/v9"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	_ "github.com/lib/pq"
)

type Repositories struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	User         UserRepository
	Product      ProductRepository
	Cart         CartRepository
	Order        OrderRepository
	Payment      PaymentRepository
	Notification NotificationRepository
	RateLimiter  RateLimitRepository
	Cache        cache.Cache
}

func New(cfg *config.Config, redisClient *redis.Client, cacheImpl cache.Cache, rateLimiter RateLimitRepository) (*Repositories, error) {

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
		RedisClient:  redisClient,
		User:         NewUserRepo(db),
		Product:      NewProductRepo(db),
		Cart:         NewCartRepo(db),
		Order:        NewOrderRepository(db),
		Payment:      NewPaymentRepository(db),
		Notification: NewNotificationRepo(db),
		RateLimiter:  rateLimiter,
		Cache:        cacheImpl,
	}, nil
}

func (r *Repositories) Close() error {
	// Close DB connection
	dbErr := r.DB.Close()
	redisErr := r.RedisClient.Close()

	if dbErr != nil {
		return fmt.Errorf("error closing database: %w", dbErr)
	}
	if redisErr != nil {
		return fmt.Errorf("error closing redis: %w", redisErr)
	}
	return nil
}
