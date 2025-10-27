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
func NewRedisClient(host, port string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  30 * time.Second,  // Increased for large files (150MB+)
		WriteTimeout: 30 * time.Second,  // Increased for large files (150MB+)
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

// GetFileStatus gets the status of a file from cache
// Returns: "completed", "processing", or empty string if not found
func (r *RedisCache) GetFileStatus(ctx context.Context, fileHash string) (string, error) {
	key := fmt.Sprintf("processed:%s", fileHash)

	status, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Not found
	}
	if err != nil {
		return "", fmt.Errorf("failed to get file status from Redis: %w", err)
	}

	return status, nil
}

// SetFileStatus sets the status of a file in cache
func (r *RedisCache) SetFileStatus(ctx context.Context, fileHash, status string, ttl time.Duration) error {
	key := fmt.Sprintf("processed:%s", fileHash)

	err := r.client.Set(ctx, key, status, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set file status in Redis: %w", err)
	}

	return nil
}

// DeleteFileStatus removes a file status from cache
func (r *RedisCache) DeleteFileStatus(ctx context.Context, fileHash string) error {
	key := fmt.Sprintf("processed:%s", fileHash)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete file status from Redis: %w", err)
	}

	return nil
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, fileHash string) (bool, error) {
	key := fmt.Sprintf("processed:%s", fileHash)

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if key exists: %w", err)
	}

	return result > 0, nil
}

// ===== Blob Storage Functions (for large files) =====

// StoreFileBlob stores file content in Redis as a blob
// Key format: blob:{file_hash}
// TTL: 1 hour (file should be consumed by parser within this time)
func (r *RedisCache) StoreFileBlob(ctx context.Context, fileHash string, fileContent []byte, ttl time.Duration) error {
	key := fmt.Sprintf("blob:%s", fileHash)

	err := r.client.Set(ctx, key, fileContent, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store file blob in Redis: %w", err)
	}

	return nil
}

// GetFileBlob retrieves file content from Redis
func (r *RedisCache) GetFileBlob(ctx context.Context, fileHash string) ([]byte, error) {
	key := fmt.Sprintf("blob:%s", fileHash)

	content, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("file blob not found in Redis (may have expired)")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file blob from Redis: %w", err)
	}

	return content, nil
}

// DeleteFileBlob deletes file content from Redis
func (r *RedisCache) DeleteFileBlob(ctx context.Context, fileHash string) error {
	key := fmt.Sprintf("blob:%s", fileHash)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete file blob from Redis: %w", err)
	}

	return nil
}

// BlobExists checks if a file blob exists in Redis
func (r *RedisCache) BlobExists(ctx context.Context, fileHash string) (bool, error) {
	key := fmt.Sprintf("blob:%s", fileHash)

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if blob exists: %w", err)
	}

	return result > 0, nil
}
