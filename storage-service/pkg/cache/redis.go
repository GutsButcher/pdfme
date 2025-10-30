package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(host, port, password string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		DB:           0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// SetFileCompleted marks a file as completed in cache
func (r *RedisCache) SetFileCompleted(ctx context.Context, fileHash string) error {
	key := fmt.Sprintf("processed:%s", fileHash)

	err := r.client.Set(ctx, key, "completed", 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to set file completed in Redis: %w", err)
	}

	return nil
}
