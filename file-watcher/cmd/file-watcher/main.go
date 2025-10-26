package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pdfme/file-watcher/pkg/cache"
	"github.com/pdfme/file-watcher/pkg/database"
	minioPkg "github.com/pdfme/file-watcher/pkg/minio"
	"github.com/pdfme/file-watcher/pkg/processor"
	"github.com/pdfme/file-watcher/pkg/rabbitmq"
)

func main() {
	log.Println("=== File Watcher Service Starting ===")

	// Get environment variables
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@rabbitmq:5672")
	queueName := getEnv("QUEUE_NAME", "parse_ready")
	bucketName := getEnv("BUCKET_NAME", "uploads")
	pollInterval := getEnv("POLL_INTERVAL", "10s")
	batchSize := getEnvInt("BATCH_SIZE", 100)
	rateLimit := getEnvInt("RATE_LIMIT_PER_SECOND", 50)

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

	interval, err := time.ParseDuration(pollInterval)
	if err != nil {
		log.Fatalf("Invalid POLL_INTERVAL: %s", err)
	}

	log.Printf("Config:\n")
	log.Printf("  RabbitMQ URL: %s\n", rabbitURL)
	log.Printf("  Queue Name: %s\n", queueName)
	log.Printf("  MinIO Endpoint: %s\n", minioEndpoint)
	log.Printf("  Bucket Name: %s\n", bucketName)
	log.Printf("  Poll Interval: %v\n", interval)
	log.Printf("  Batch Size: %d\n", batchSize)
	log.Printf("  Rate Limit: %d files/sec\n", rateLimit)
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
	redisCache, err := cache.NewRedisClient(redisHost, redisPort)
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

	// Ensure bucket exists
	ctx := context.Background()
	if err := minioPkg.EnsureBucketExists(ctx, minioClient, bucketName); err != nil {
		log.Fatalf("Failed to ensure bucket exists: %s", err)
	}

	// Create RabbitMQ producer
	producer, err := rabbitmq.NewProducer(rabbitURL, queueName)
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()
	log.Println("✓ RabbitMQ connected")

	// Create file processor
	fileProcessor := processor.NewFileProcessor(
		minioClient,
		db,
		redisCache,
		producer,
		bucketName,
		batchSize,
		rateLimit,
	)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n[!] Shutdown signal received, closing...")
		producer.Close()
		db.Close()
		redisCache.Close()
		os.Exit(0)
	}()

	log.Println("\n=== File Watcher Service Ready ===\n")

	// Start polling
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial scan
	log.Println("[*] Starting initial scan...")
	if err := fileProcessor.ProcessFiles(ctx); err != nil {
		log.Printf("[!] Error during initial scan: %v\n", err)
	}

	// Check for stuck jobs
	log.Println("\n[*] Checking for stuck jobs...")
	if err := fileProcessor.CheckStuckJobs(ctx); err != nil {
		log.Printf("[!] Error checking stuck jobs: %v\n", err)
	}

	// Poll periodically
	for range ticker.C {
		log.Println("\n[*] Starting periodic scan...")

		if err := fileProcessor.ProcessFiles(ctx); err != nil {
			log.Printf("[!] Error during scan: %v\n", err)
		}

		// Check for stuck jobs every scan
		if err := fileProcessor.CheckStuckJobs(ctx); err != nil {
			log.Printf("[!] Error checking stuck jobs: %v\n", err)
		}
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
