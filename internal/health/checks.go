package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
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

type HealthEndpoint struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	StripeClient *stripeClient.Client
}

func NewReadinessHandler(cfg *config.Config, healthEndpoint *HealthEndpoint) (http.Handler, error) {

	h, err := health.New(

		health.WithComponent(health.Component{
			Name:    cfg.OTel.ServiceName,
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
					if healthEndpoint.StripeClient == nil {
						return fmt.Errorf("stripe client is not initialized")
					}

					reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
					defer cancel()

					params := &stripe.BalanceParams{
						Params: stripe.Params{
							Context: reqCtx,
						},
					}
					_, err := balance.Get(params)
					if err != nil {
						if ctxErr := reqCtx.Err(); ctxErr == context.DeadlineExceeded {
							return fmt.Errorf("stripe API call timed out: %w", ctxErr)
						}
						return fmt.Errorf("failed to connect to stripe: %w", err)
					}
					return nil
				},
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create readiness health instance: %w", err)
	}

	return h.Handler(), nil
}

func NewLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Service is alive. Time: %s\n", time.Now().Format(time.RFC3339))
	}
}
