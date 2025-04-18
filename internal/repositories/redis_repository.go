package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/redis/go-redis/v9"
)

type RateLimitRepository interface {
	CheckLoginRateLimit(ctx context.Context, username string) (bool, int, int, error)
}

type redisRepository struct {
	client *redis.Client
	cfg    *config.Config
}

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {

	redisURL := cfg.RedisConnect.GetDSN()
	slog.Info("Connecting to Redis", slog.String("url", fmt.Sprintf("redis://%s:<password>@%s:%s", cfg.RedisConnect.Username, cfg.RedisConnect.Host, cfg.RedisConnect.Port)))

	// Parse the Redis URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Error("Failed to parse Redis URL", slog.Any("error", err), slog.String("url", redisURL))
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	opt.DB = cfg.RedisConnect.DB

	client := redis.NewClient(opt)

	// Connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Error("Failed to connect to Redis", slog.Any("error", err))
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("✅ Successfully connected to Redis")
	return client, nil

}

func NewRateLimitRepo(client *redis.Client, cfg *config.Config) RateLimitRepository {
	return &redisRepository{client: client, cfg: cfg}
}

// Returns isAllowed, attempts left, seconds to wait, error
func (r *redisRepository) CheckLoginRateLimit(ctx context.Context, username string) (bool, int, int, error) {

	logger := middleware.LoggerFromContext(ctx)

	// create a username key
	key := fmt.Sprintf("login_attempts:%s", username)

	now := time.Now().Unix()

	// This means only login attempts after 'this time' are counted.
	windowStart := now - int64(r.cfg.RateConfig.WindowSize.Seconds())

	// redis pipeline for executing multiple commands
	pipe := r.client.Pipeline()

	// remove old entries from the pipeline
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// add the current login attempt
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

	// count the number of login attempts, currently in the window
	count := pipe.ZCard(ctx, key)

	// delete the redis key after expiry
	pipe.Expire(ctx, key, r.cfg.RateConfig.WindowSize)

	// execute the commands
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Error("Redis pipeline execution failed for rate limit", slog.String("key", key), slog.Any("error", err))
		return false, 0, 0, fmt.Errorf("redis pipeline error for rate limit check: %w", err)
	}

	// remaining attempts
	attempts := count.Val()
	remaining := r.cfg.RateConfig.MaxAttempts - attempts

	if attempts >= r.cfg.RateConfig.MaxAttempts {

		oldestScoreCmd := r.client.ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
			Key: key, Start: 0, Stop: 0,
		})

		scores, err := oldestScoreCmd.Result()
		if err != nil || len(scores) == 0 {
			logger.Error("Failed to get oldest attempt time for rate limit", slog.String("key", key), slog.Any("error", err))
			return false, 0, int(r.cfg.RateConfig.WindowSize.Seconds()), fmt.Errorf("failed to get oldest attempt time: %w", err)
		}

		oldestTimestamp := int64(scores[0].Score)

		retryAfter := max((oldestTimestamp+int64(r.cfg.RateConfig.WindowSize.Seconds()))-now, 0)

		logger.Warn("Rate limit exceeded for user", slog.String("username", username), slog.Int64("attempts", attempts))
		return false, 0, int(retryAfter), nil
	}

	logger.Debug("Rate limit check passed", slog.String("username", username), slog.Int64("attempts", attempts), slog.Int64("remaining", remaining))
	return true, int(remaining), 0, nil
}

// login attempts stored in redis
/*
	Score → A numerical value that determines the order (sorting) of elements.
	Member (Value) → The actual stored data (a unique identifier for the element).

	login_attempts:johndoe
	--------------------------
	| Score (timestamp) | Value  |
	--------------------------
	| 1700000000       | 1700000000 |
	| 1700000020       | 1700000020 |
	| 1700000045       | 1700000045 |


	User	Redis Key				Stored Login Attempts (timestamps)
	Alice	login_attempts:alice	1700000020, 1700000045 (2 attempts)
	Bob		login_attempts:bob		1700000030, 1700000060, 1700000075, 1700000090 (4 attempts)
	Charlie	login_attempts:charlie	1700000010, 1700000025, 1700000050, 1700000080, 1700000095, 1700000105 (6 attempts)

*/
