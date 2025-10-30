package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/pdfme/storage-service/pkg/cache"
	"github.com/pdfme/storage-service/pkg/database"
	minioPkg "github.com/pdfme/storage-service/pkg/minio"
	"github.com/pdfme/storage-service/pkg/rabbitmq"
)

func main() {
	log.Println("=== Storage Service Starting ===")

	// Get environment variables
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@rabbitmq:5672")
	queueName := getEnv("QUEUE_NAME", "storage_ready")

	minioEndpoint := getEnv("MINIO_ENDPOINT", "minio:9000")
	minioAccessKey := getEnv("MINIO_ROOT_USER", "minioadmin")
	minioSecretKey := getEnv("MINIO_ROOT_PASSWORD", "minioadmin")
	minioUseSSL := getEnv("MINIO_USE_SSL", "false") == "true"

	postgresHost := getEnv("POSTGRES_HOST", "localhost")
	postgresPort := getEnv("POSTGRES_PORT", "5432")
	postgresUser := getEnv("POSTGRES_USER", "pdfme")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "pdfme_secure_pass")
	postgresDB := getEnv("POSTGRES_DB", "pdfme")
	postgresMaxPool := getEnvInt("POSTGRES_MAX_POOL_SIZE", 10)

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	log.Printf("Config:\n")
	log.Printf("  RabbitMQ URL: %s\n", rabbitURL)
	log.Printf("  Queue Name: %s\n", queueName)
	log.Printf("  MinIO Endpoint: %s\n", minioEndpoint)
	log.Printf("  MinIO Use SSL: %v\n", minioUseSSL)
	log.Printf("  PostgreSQL: %s:%s/%s\n", postgresHost, postgresPort, postgresDB)
	log.Printf("  Redis: %s:%s\n", redisHost, redisPort)

	// Initialize PostgreSQL
	db, err := database.NewPostgresDB(database.Config{
		Host:     postgresHost,
		Port:     postgresPort,
		User:     postgresUser,
		Password: postgresPassword,
		DBName:   postgresDB,
		MaxPool:  postgresMaxPool,
	})
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL: %s", err)
	}
	defer db.Close()
	log.Println("✓ PostgreSQL connected")

	// Initialize Redis
	redisCache, err := cache.NewRedisClient(redisHost, redisPort, redisPassword)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %s", err)
	}
	defer redisCache.Close()
	log.Println("✓ Redis connected")

	// Initialize MinIO client
	minioClient, err := minioPkg.InitMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioUseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %s", err)
	}
	log.Println("✓ MinIO connected")

	// Create RabbitMQ consumer
	consumer, err := rabbitmq.NewConsumer(rabbitURL, queueName, minioClient, db, redisCache)
	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}
	defer consumer.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n[!] Shutdown signal received, closing...")
		consumer.Close()
		db.Close()
		redisCache.Close()
		os.Exit(0)
	}()

	// Start consuming
	log.Println("\n=== Storage Service Ready ===\n")
	if err := consumer.Start(); err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
