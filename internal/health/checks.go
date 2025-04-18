package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/hellofresh/health-go/v5"
	"github.com/hellofresh/health-go/v5/checks/postgres"
	healthRedis "github.com/hellofresh/health-go/v5/checks/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/balance"
)

type Endpoints struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	StripeClient *stripeClient.Client
}

func NewHealthHandler(cfg *config.Config, endpoints *Endpoints) (*health.Health, error) {

	h, err := health.New(
		health.WithComponent(health.Component{

			Name:    "scalable-ecommerce-platform",
			Version: "1.0.0",
		}),
		health.WithSystemInfo(),
		health.WithChecks(
			health.Config{
				Name:      "database",
				Timeout:   3 * time.Second,
				SkipOnErr: false,
				Check: postgres.New(postgres.Config{
					DSN: cfg.Database.GetDSN(),
				}),
			},
			health.Config{
				Name:      "redis",
				Timeout:   2 * time.Second,
				SkipOnErr: false,
				Check: healthRedis.New(
					healthRedis.Config{
						DSN: cfg.RedisConnect.GetDSN(),
					},
				),
			},
			health.Config{
				Name:      "stripe",
				Timeout:   5 * time.Second,
				SkipOnErr: false,
				Check: func(ctx context.Context) error {
					if endpoints.StripeClient == nil {
						return fmt.Errorf("stripe client is not initialized")
					}
					params := &stripe.BalanceParams{
						Params: stripe.Params{
							Context: ctx,
						},
					}
					_, err := balance.Get(params)
					if err != nil {
						return fmt.Errorf("failed to connect to stripe: %w", err)
					}
					return nil
				},
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create health instance: %w", err)
	}

	return h, nil
}
