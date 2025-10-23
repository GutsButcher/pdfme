# Storage Service

A Go microservice that consumes PDF files from RabbitMQ and uploads them to MinIO object storage.

## Overview

This service is part of a stateless microservices architecture for PDF generation and storage.

**Role**:
- Consumes messages from `storage_ready` RabbitMQ queue
- Decodes base64-encoded PDF content
- Uploads PDFs to MinIO object storage

## Architecture

```
RabbitMQ (storage_ready) → Storage Service → MinIO
```

## Features

- **Stateless**: No local file storage
- **Automatic Bucket Creation**: Creates buckets if they don't exist
- **Retry Logic**: Automatic reconnection to RabbitMQ
- **Base64 Decoding**: Handles base64-encoded PDF content
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals

## Requirements

- Go 1.21+
- RabbitMQ
- MinIO (or S3-compatible storage)

## Installation

### Local Development

```bash
# Install dependencies
go mod download

# Build
go build -o storage-service ./cmd/storage-service

# Run
./storage-service
```

### Docker

```bash
# Build image
docker build -t storage-service .

# Run container
docker run -d \
  -e RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672 \
  -e MINIO_ENDPOINT=minio:9000 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  storage-service
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RABBITMQ_URL` | RabbitMQ connection string | `amqp://admin:admin123@rabbitmq:5672` |
| `QUEUE_NAME` | Queue to consume from | `storage_ready` |
| `MINIO_ENDPOINT` | MinIO server endpoint | `minio:9000` |
| `MINIO_ROOT_USER` | MinIO access key | `minioadmin` |
| `MINIO_ROOT_PASSWORD` | MinIO secret key | `minioadmin` |
| `MINIO_USE_SSL` | Use SSL for MinIO | `false` |

## Message Format

The service expects JSON messages with this format:

```json
{
  "bucket_name": "pdfs",
  "filename": "invoice_742891.pdf",
  "file_content": "JVBERi0xLjQKJeLjz9MKMyAwIG9iago8..."
}
```

**Fields**:
- `bucket_name` (string): MinIO bucket name
- `filename` (string): Filename for the PDF
- `file_content` (string): Base64-encoded PDF content

## Operation

### Processing Flow

1. **Connect**: Establishes connection to RabbitMQ with retry logic (max 10 attempts)
2. **Consume**: Listens for messages on `storage_ready` queue
3. **Decode**: Decodes base64 PDF content
4. **Ensure Bucket**: Creates bucket if it doesn't exist
5. **Upload**: Uploads PDF to MinIO
6. **Acknowledge**: Acknowledges message after successful upload

### Error Handling

- **Decode Errors**: Message is acknowledged (not requeued)
- **Bucket Creation Errors**: Message is acknowledged (not requeued)
- **Upload Errors**: Message is acknowledged (not requeued)
- **Connection Errors**: Service exits (Docker will restart)

In production, consider using dead-letter queues for failed messages.

## Monitoring

### Logs

The service logs all operations:

```
=== Storage Service Starting ===
Config:
  RabbitMQ URL: amqp://admin:admin123@rabbitmq:5672
  Queue Name: storage_ready
  MinIO Endpoint: minio:9000
  MinIO Use SSL: false
✓ MinIO client initialized: minio:9000
✓ Connected to RabbitMQ
✓ Connected to RabbitMQ, queue: storage_ready
[*] Waiting for messages. To exit press CTRL+C

[→] Received message
  Bucket: pdfs
  Filename: invoice_742891.pdf
  Size: 123456 bytes
✓ Bucket exists: pdfs
✓ Successfully uploaded: pdfs/invoice_742891.pdf (size: 123456 bytes)
[✓] Message processed successfully
```

### Health Checks

Monitor service health by checking:
1. **Process Status**: Is the service running?
2. **RabbitMQ Connection**: Check consumer count in RabbitMQ UI
3. **Message Processing**: Monitor queue depth and processing rate
4. **MinIO Upload**: Check MinIO for uploaded files

## Development

### Project Structure

```
storage-service/
├── cmd/
│   └── storage-service/
│       └── main.go           # Entry point
├── pkg/
│   ├── minio/
│   │   ├── client.go         # MinIO client initialization
│   │   ├── upload.go         # Upload function
│   │   └── bucket.go         # Bucket operations
│   ├── rabbitmq/
│   │   └── consumer.go       # RabbitMQ consumer
│   └── types/
│       └── message.go        # Message types
├── Dockerfile
├── go.mod
└── README.md
```

### Running Locally

```bash
# Start dependencies
docker-compose up -d rabbitmq minio

# Set environment variables
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
export MINIO_ENDPOINT=localhost:9000
export MINIO_ROOT_USER=minioadmin
export MINIO_ROOT_PASSWORD=minioadmin

# Run service
go run cmd/storage-service/main.go
```

### Testing

1. **Send Test Message**:
   ```bash
   # Use the test producer from parent directory
   cd ..
   node test_producer.js test_request_5trans.json
   ```

2. **Verify Upload**:
   - Go to MinIO Console: http://localhost:9001
   - Login: minioadmin / minioadmin
   - Check "pdfs" bucket for uploaded files

## Deployment

### Docker Compose

The service is included in the main `docker-compose.yml`:

```yaml
storage-service:
  build:
    context: ./storage-service
  environment:
    - RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672
    - QUEUE_NAME=storage_ready
    - MINIO_ENDPOINT=minio:9000
    - MINIO_ROOT_USER=minioadmin
    - MINIO_ROOT_PASSWORD=minioadmin
  depends_on:
    - rabbitmq
    - minio
```

### Kubernetes

Example Kubernetes deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: storage-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: storage-service
  template:
    metadata:
      labels:
        app: storage-service
    spec:
      containers:
      - name: storage-service
        image: storage-service:latest
        env:
        - name: RABBITMQ_URL
          value: "amqp://user:pass@rabbitmq-service:5672"
        - name: MINIO_ENDPOINT
          value: "minio-service:9000"
        - name: MINIO_ROOT_USER
          valueFrom:
            secretKeyRef:
              name: minio-secret
              key: accesskey
        - name: MINIO_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: minio-secret
              key: secretkey
```

## Scaling

### Horizontal Scaling

Run multiple instances:

```bash
docker-compose up -d --scale storage-service=3
```

Each instance will consume from the queue independently.

### Performance

- **Prefetch**: Set to 1 (one message at a time)
- **Concurrency**: Scale horizontally for more throughput
- **Network**: Upload speed depends on MinIO network bandwidth

## Troubleshooting

### Service Won't Start

1. **Check RabbitMQ Connection**:
   ```bash
   # Verify RabbitMQ is accessible
   telnet rabbitmq 5672
   ```

2. **Check MinIO Connection**:
   ```bash
   # Verify MinIO is accessible
   curl http://minio:9000/minio/health/live
   ```

3. **Check Logs**:
   ```bash
   docker-compose logs storage-service
   ```

### Messages Not Being Consumed

1. **Verify Queue Exists**: Check RabbitMQ Management UI
2. **Check Consumer Count**: Should show at least 1 consumer
3. **Verify Message Format**: Ensure message is valid JSON
4. **Check Logs**: Look for errors in service logs

### Upload Failures

1. **Check MinIO Status**: Verify MinIO is running
2. **Check Credentials**: Verify MINIO_ROOT_USER/PASSWORD
3. **Check Bucket**: Verify bucket exists or can be created
4. **Check Logs**: Look for MinIO-specific errors

## Security

### Production Recommendations

1. **Change Default Credentials**:
   - MinIO: Change minioadmin/minioadmin
   - RabbitMQ: Change admin/admin123

2. **Enable TLS**:
   ```go
   minioClient, err := minio.New(endpoint, &minio.Options{
       Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
       Secure: true, // Enable TLS
   })
   ```

3. **Use Secrets Management**:
   - Store credentials in Kubernetes secrets
   - Use environment variable injection
   - Rotate credentials regularly

4. **Network Security**:
   - Use internal networks only
   - Don't expose MinIO publicly
   - Use firewall rules

## License

See parent project license.

## Contributing

See parent project contributing guidelines.

## References

- [MinIO Go SDK](https://github.com/minio/minio-go)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [Architecture Documentation](../ARCHITECTURE.md)
