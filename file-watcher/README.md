# File Watcher Service

A Go microservice that monitors MinIO bucket for new files and triggers the PDF generation pipeline.

## Overview

This service watches the `nonparsed_files` MinIO bucket for uploaded files, then sends them to the parser service via RabbitMQ.

## Architecture

```
MinIO (nonparsed_files) → File Watcher → RabbitMQ (parse_ready) → Parser
```

## Features

- **Automatic File Detection**: Polls MinIO bucket every 10 seconds
- **Base64 Encoding**: Encodes file content for transport
- **OrgID Extraction**: Extracts organization ID from filename
- **Stateless**: No local file storage, tracks processed files in memory
- **Retry Logic**: Automatic reconnection to RabbitMQ and MinIO
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RABBITMQ_URL` | RabbitMQ connection string | `amqp://admin:admin123@rabbitmq:5672` |
| `QUEUE_NAME` | Queue to produce to | `parse_ready` |
| `BUCKET_NAME` | MinIO bucket to watch | `nonparsed_files` |
| `POLL_INTERVAL` | How often to check for new files | `10s` |
| `MINIO_ENDPOINT` | MinIO server endpoint | `minio:9000` |
| `MINIO_ROOT_USER` | MinIO access key | `minioadmin` |
| `MINIO_ROOT_PASSWORD` | MinIO secret key | `minioadmin` |
| `MINIO_USE_SSL` | Use SSL for MinIO | `false` |

## Message Format

Files are sent to `parse_ready` queue with this format:

```json
{
  "filename": "266_statement.pdf",
  "file_content": "JVBERi0xLjQKJeLjz9MK...",
  "org_id": "266"
}
```

**Fields**:
- `filename`: Original filename from MinIO
- `file_content`: Base64-encoded file content
- `org_id`: Organization ID extracted from filename

## Filename Convention

Files should be named with organization ID prefix:

```
{orgId}_{description}.{ext}
```

**Examples**:
- `266_statement.pdf` → orgId: "266"
- `org123_invoice.pdf` → orgId: "123"
- `456_receipt.pdf` → orgId: "456"

If no orgId can be extracted, it defaults to "unknown".

## Operation

### Processing Flow

1. **Poll**: Checks MinIO bucket every `POLL_INTERVAL`
2. **Detect**: Identifies new files not yet processed
3. **Download**: Fetches file content from MinIO
4. **Encode**: Encodes to base64
5. **Extract**: Extracts orgID from filename
6. **Publish**: Sends message to RabbitMQ `parse_ready` queue
7. **Track**: Marks file as processed (in memory)

### File Cleanup

By default, files remain in the `nonparsed_files` bucket after processing.

To **auto-delete** files after processing, uncomment in `pkg/minio/watcher.go`:

```go
// Uncomment to delete files after processing:
if err := w.deleteFile(object.Key); err != nil {
    log.Printf("[!] Error deleting %s: %s\n", object.Key, err)
}
```

## Deployment

### Docker Compose

Included in main `docker-compose.yml`:

```yaml
file-watcher:
  build:
    context: ./file-watcher
  environment:
    - BUCKET_NAME=nonparsed_files
    - QUEUE_NAME=parse_ready
    - POLL_INTERVAL=10s
  depends_on:
    - rabbitmq
    - minio
```

### Local Development

```bash
# Start dependencies
docker-compose up -d rabbitmq minio

# Set environment variables
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
export MINIO_ENDPOINT=localhost:9000
export BUCKET_NAME=nonparsed_files
export QUEUE_NAME=parse_ready

# Run service
go run cmd/file-watcher/main.go
```

## Monitoring

### Service Logs

```bash
docker-compose logs -f file-watcher
```

**Normal Operation**:
```
=== File Watcher Service Starting ===
✓ MinIO client initialized: minio:9000
✓ Bucket exists: nonparsed_files
✓ Connected to RabbitMQ, queue: parse_ready
=== File Watcher Service Ready ===
[*] Starting to poll bucket 'nonparsed_files' every 10s

[→] Found new file: 266_statement.pdf (size: 123456 bytes)
  Published to queue: 266_statement.pdf (size: 164608 bytes)
[✓] Processed: 266_statement.pdf
```

### Queue Status

Check RabbitMQ Management UI (http://localhost:15672):
- View `parse_ready` queue depth
- Monitor message rates
- Check for errors

### Bucket Status

Check MinIO Console (http://localhost:9001):
- View `nonparsed_files` bucket
- Monitor file uploads
- Check file sizes

## Testing

### Manual Test

1. **Upload test file**:
   ```bash
   # Create test file
   echo "test content" > test.pdf

   # Upload via MinIO Console or mc
   docker exec -it pdfme-minio sh -c 'echo "test" > /tmp/test.pdf'
   docker exec pdfme-minio mc cp /tmp/test.pdf local/nonparsed_files/266_test.pdf
   ```

2. **Watch logs**:
   ```bash
   docker-compose logs -f file-watcher
   ```

3. **Check queue**:
   - Go to http://localhost:15672
   - Check `parse_ready` queue
   - Should show 1 message

### Automated Test

Create a test script to upload files automatically:

```bash
#!/bin/bash
# test_file_upload.sh

FILE=$1
ORGID=${2:-"266"}

if [ -z "$FILE" ]; then
  echo "Usage: ./test_file_upload.sh <file> [orgId]"
  exit 1
fi

BASENAME=$(basename "$FILE")
TARGET="${ORGID}_${BASENAME}"

echo "Uploading $FILE as $TARGET..."
docker exec pdfme-minio mc cp "$FILE" "local/nonparsed_files/${TARGET}"
echo "Done! Watch logs: docker-compose logs -f file-watcher"
```

## Project Structure

```
file-watcher/
├── cmd/
│   └── file-watcher/
│       └── main.go           # Entry point
├── pkg/
│   ├── minio/
│   │   ├── client.go         # MinIO client initialization
│   │   └── watcher.go        # Bucket polling logic
│   ├── rabbitmq/
│   │   └── producer.go       # RabbitMQ producer
│   └── types/
│       └── message.go        # Message types
├── Dockerfile
├── go.mod
└── README.md
```

## Scaling

### Performance Tuning

**Faster Detection**:
```yaml
environment:
  - POLL_INTERVAL=5s  # Check every 5 seconds
```

**Multiple Instances**:
```bash
docker-compose up -d --scale file-watcher=2
```

**Note**: Multiple instances will process files redundantly unless you implement distributed locking.

### Production Considerations

1. **Distributed Locking**: Use Redis or database to track processed files
2. **Event Notifications**: Use MinIO event notifications instead of polling
3. **File Validation**: Add file type and size validation
4. **Dead Letter Queue**: Handle files that fail processing
5. **Metrics**: Export Prometheus metrics for monitoring

## Advanced: MinIO Event Notifications

Instead of polling, use MinIO event notifications for real-time processing:

```go
// Example using MinIO event notifications
for notificationInfo := range minioClient.ListenBucketNotification(
    context.Background(),
    bucketName,
    "",
    "",
    []string{"s3:ObjectCreated:*"},
) {
    if notificationInfo.Err != nil {
        log.Println(notificationInfo.Err)
        continue
    }

    for _, record := range notificationInfo.Records {
        processFile(record.S3.Object.Key)
    }
}
```

This eliminates polling delay and reduces resource usage.

## Troubleshooting

### Service Won't Start

**Check Dependencies**:
```bash
docker-compose ps rabbitmq minio
```

Both should be "healthy".

**Check Logs**:
```bash
docker-compose logs file-watcher
```

### Files Not Being Detected

**Verify Bucket**:
- Check MinIO Console (http://localhost:9001)
- Ensure `nonparsed_files` bucket exists
- Verify files are actually uploaded

**Check Logs**:
```bash
docker-compose logs -f file-watcher
```

Should show polling activity every 10 seconds.

**Verify Poll Interval**:
```bash
docker-compose exec file-watcher env | grep POLL
```

### Messages Not Sent to Queue

**Check RabbitMQ Connection**:
```bash
# View connections
docker exec pdfme-rabbitmq rabbitmqctl list_connections
```

**Check Queue**:
```bash
# List queues
docker exec pdfme-rabbitmq rabbitmqctl list_queues
```

**Verify Credentials**:
Ensure RABBITMQ_URL credentials match RabbitMQ configuration.

## References

- [MinIO Go SDK](https://github.com/minio/minio-go)
- [RabbitMQ Go Client](https://github.com/rabbitmq/amqp091-go)
- [Complete Workflow](../COMPLETE_WORKFLOW.md)
