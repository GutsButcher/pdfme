package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/pdfme/file-watcher/pkg/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Producer handles RabbitMQ message production
type Producer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// NewProducer creates a new RabbitMQ producer
func NewProducer(rabbitURL, queueName string) (*Producer, error) {
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

	log.Printf("✓ Connected to RabbitMQ, queue: %s\n", queueName)

	return &Producer{
		conn:      conn,
		channel:   channel,
		queueName: queueName,
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

// PublishFile publishes a file message to the queue
func (p *Producer) PublishFile(message *types.FileMessage) error {
	// Marshal message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %s", err)
	}

	// Publish message
	err = p.channel.Publish(
		"",           // exchange
		p.queueName,  // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Persistent message
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %s", err)
	}

	log.Printf("  Published to queue: %s (size: %d bytes)\n", message.Filename, message.FileSize)
	return nil
}

// Close closes the producer connection
func (p *Producer) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
