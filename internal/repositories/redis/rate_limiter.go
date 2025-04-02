package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client *redis.Client
	config *config.Config
}

func NewRedisRepo(cfg *config.Config) (*RedisRepo, error) {

	// client := redis.NewClient(&redis.Options{
	// 	Addr:     cfg.RedisConnect.Host,
	// 	Username: cfg.RedisConnect.Username,
	// 	Password: cfg.RedisConnect.Password,
	// 	DB:       cfg.RedisConnect.DB,
	// })

	// Redis connection
	redisURL := "redis://default:fctMBxFvOnFqMammXZeavtHQhwNsvJKw@ballast.proxy.rlwy.net:25502"

	// Parse the Redis URL
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test the connection to make sure Redis is reachable
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return &RedisRepo{client: client, config: cfg}, nil

}

// Returns isAllowed, attempts left, seconds to wait, error
func (r *RedisRepo) CheckLoginRateLimit(ctx context.Context, username string) (bool, int, int, error) {

	// create a username key
	key := fmt.Sprintf("login_attempts:%s", username)

	now := time.Now().Unix()

	// This means only login attempts after 'this time' are counted.
	windowStart := now - int64(r.config.RateConfig.WindowSize.Seconds())

	// redis pipeline for executing multiple commands
	pipe := r.client.Pipeline()

	// remove old entries from the pipeline
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// add the current login attempt
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

	// count the number of login attempts
	count := pipe.ZCard(ctx, key)

	// delete the redis key after expiry
	pipe.Expire(ctx, key, r.config.RateConfig.WindowSize)

	// execute the commands
	_, err := pipe.Exec(ctx)

	if err != nil {
		return false, 0, 0, err
	}

	// remaining attempts
	attempts := count.Val()
	remaining := r.config.RateConfig.MaxAttempts - attempts

	if attempts >= r.config.RateConfig.MaxAttempts {
		oldest, err := r.client.ZRange(ctx, key, 0, 0).Result()
		if err != nil || len(oldest) == 0 {
			return false, 0, 0, err
		}

		// convert the oldest attempt into a time value.
		oldestTime, err := strconv.ParseInt(oldest[0], 10, 64)
		if err != nil {
			return false, 0, 0, err
		}

		retryAfter := int64(r.config.RateConfig.WindowSize.Seconds()) - (now - oldestTime)

		return false, 0, int(retryAfter), err
	}

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
