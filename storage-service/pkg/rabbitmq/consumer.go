package rabbitmq

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	minioPkg "github.com/pdfme/storage-service/pkg/minio"
	"github.com/pdfme/storage-service/pkg/types"

	"github.com/minio/minio-go/v7"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer handles RabbitMQ message consumption
type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	queueName   string
	minioClient *minio.Client
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(rabbitURL, queueName string, minioClient *minio.Client) (*Consumer, error) {
	conn, err := connectWithRetry(rabbitURL, 10, 5*time.Second)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %s", err)
	}

	// Declare queue
	_, err = channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %s", err)
	}

	// Set prefetch count
	err = channel.Qos(1, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %s", err)
	}

	log.Printf("✓ Connected to RabbitMQ, queue: %s\n", queueName)

	return &Consumer{
		conn:        conn,
		channel:     channel,
		queueName:   queueName,
		minioClient: minioClient,
	}, nil
}

// connectWithRetry attempts to connect to RabbitMQ with retries
func connectWithRetry(url string, maxRetries int, delay time.Duration) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to RabbitMQ (attempt %d/%d)...\n", i+1, maxRetries)
		conn, err = amqp.Dial(url)
		if err == nil {
			log.Println("✓ Connected to RabbitMQ")
			return conn, nil
		}

		log.Printf("Failed to connect: %s\n", err)
		if i < maxRetries-1 {
			log.Printf("Retrying in %v...\n", delay)
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %s", maxRetries, err)
}

// Start begins consuming messages
func (c *Consumer) Start() error {
	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %s", err)
	}

	log.Println("[*] Waiting for messages. To exit press CTRL+C")

	// Process messages
	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			if err := c.processMessage(msg); err != nil {
				log.Printf("[✗] Error processing message: %s\n", err)
				// Reject and don't requeue to avoid infinite loop
				msg.Nack(false, false)
			} else {
				msg.Ack(false)
			}
		}
	}()

	<-forever
	return nil
}

// processMessage processes a single message
func (c *Consumer) processMessage(msg amqp.Delivery) error {
	log.Println("\n[→] Received message")

	// Parse message
	var storageMsg types.StorageMessage
	if err := json.Unmarshal(msg.Body, &storageMsg); err != nil {
		return fmt.Errorf("failed to parse message: %s", err)
	}

	log.Printf("  Bucket: %s\n", storageMsg.BucketName)
	log.Printf("  Filename: %s\n", storageMsg.Filename)

	// Decode base64 content
	fileBytes, err := base64.StdEncoding.DecodeString(storageMsg.FileContent)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %s", err)
	}

	log.Printf("  Size: %d bytes\n", len(fileBytes))

	// Ensure bucket exists
	if err := minioPkg.EnsureBucket(storageMsg.BucketName, c.minioClient); err != nil {
		return fmt.Errorf("failed to ensure bucket: %s", err)
	}

	// Upload to MinIO
	if err := minioPkg.UploadObject(fileBytes, storageMsg.Filename, storageMsg.BucketName, c.minioClient); err != nil {
		return fmt.Errorf("failed to upload to MinIO: %s", err)
	}

	log.Println("[✓] Message processed successfully\n")
	return nil
}

// Close closes the consumer connection
func (c *Consumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
