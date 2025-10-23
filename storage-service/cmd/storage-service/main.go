package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	log.Printf("Config:\n")
	log.Printf("  RabbitMQ URL: %s\n", rabbitURL)
	log.Printf("  Queue Name: %s\n", queueName)
	log.Printf("  MinIO Endpoint: %s\n", minioEndpoint)
	log.Printf("  MinIO Use SSL: %v\n", minioUseSSL)

	// Initialize MinIO client
	minioClient, err := minioPkg.InitMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioUseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %s", err)
	}

	// Create RabbitMQ consumer
	consumer, err := rabbitmq.NewConsumer(rabbitURL, queueName, minioClient)
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
		os.Exit(0)
	}()

	// Start consuming
	log.Println("=== Storage Service Ready ===\n")
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
