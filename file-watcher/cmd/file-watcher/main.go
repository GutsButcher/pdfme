package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	minioPkg "github.com/pdfme/file-watcher/pkg/minio"
	"github.com/pdfme/file-watcher/pkg/rabbitmq"
	"github.com/pdfme/file-watcher/pkg/types"
)

func main() {
	log.Println("=== File Watcher Service Starting ===")

	// Get environment variables
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://admin:admin123@rabbitmq:5672")
	queueName := getEnv("QUEUE_NAME", "parse_ready")
	bucketName := getEnv("BUCKET_NAME", "nonparsed_files")
	pollInterval := getEnv("POLL_INTERVAL", "10s")

	minioEndpoint := getEnv("MINIO_ENDPOINT", "minio:9000")
	minioAccessKey := getEnv("MINIO_ROOT_USER", "minioadmin")
	minioSecretKey := getEnv("MINIO_ROOT_PASSWORD", "minioadmin")
	minioUseSSL := getEnv("MINIO_USE_SSL", "false") == "true"

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

	// Initialize MinIO client
	minioClient, err := minioPkg.InitMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioUseSSL)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO client: %s", err)
	}

	// Create file watcher
	watcher, err := minioPkg.NewFileWatcher(minioClient, bucketName)
	if err != nil {
		log.Fatalf("Failed to create file watcher: %s", err)
	}

	// Create RabbitMQ producer
	producer, err := rabbitmq.NewProducer(rabbitURL, queueName)
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n[!] Shutdown signal received, closing...")
		producer.Close()
		os.Exit(0)
	}()

	log.Println("=== File Watcher Service Ready ===\n")

	// Start polling
	err = watcher.PollForNewFiles(interval, func(filename string, content []byte) error {
		// Extract orgID from filename if possible (format: orgId_xxx.pdf or just use "unknown")
		orgID := extractOrgIDFromFilename(filename)

		// Encode to base64
		base64Content := minioPkg.EncodeToBase64(content)

		// Create message
		message := &types.FileMessage{
			Filename:    filename,
			FileContent: base64Content,
			OrgID:       orgID,
		}

		// Publish to RabbitMQ
		return producer.PublishFile(message)
	})

	if err != nil {
		log.Fatalf("Failed to poll for files: %s", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// extractOrgIDFromFilename tries to extract orgID from filename
// Expects format like: "266_statement.pdf" or "org266_file.pdf"
func extractOrgIDFromFilename(filename string) string {
	base := filepath.Base(filename)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Try to extract orgID (first part before underscore)
	parts := strings.Split(base, "_")
	if len(parts) > 0 && parts[0] != "" {
		// Check if it's numeric or starts with "org"
		if strings.HasPrefix(parts[0], "org") {
			return strings.TrimPrefix(parts[0], "org")
		}
		// Assume first part is orgID
		return parts[0]
	}

	return "unknown"
}
