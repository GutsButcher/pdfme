# File Watcher Service

**Technology**: Go
**Container**: `pdfme-file-watcher`

## Function
Polls MinIO bucket for new files → sends to RabbitMQ

## Configuration

```yaml
RABBITMQ_URL: amqp://admin:admin123@rabbitmq:5672
QUEUE_NAME: parse_ready
BUCKET_NAME: uploads
POLL_INTERVAL: 10s
MINIO_ENDPOINT: minio:9000
MINIO_ROOT_USER: minioadmin
MINIO_ROOT_PASSWORD: minioadmin
MINIO_USE_SSL: false
```

## Input
- **Source**: MinIO bucket `uploads`
- **Files**: Any file type
- **Filename convention**: `{orgId}_{filename}.{ext}`

## Output

**Destination**: RabbitMQ queue `parse_ready`

**Message Format**:
```json
{
  "filename": "266_statement.txt",
  "file_content": "base64_encoded_content",
  "org_id": "266"
}
```

**Fields**:
- `filename` (string): Original filename
- `file_content` (string): Base64-encoded file content
- `org_id` (string): Extracted from filename (first part before `_`)

## Behavior
- Polls every 10 seconds
- Tracks processed files in memory
- Does not delete files from bucket
- Extracts orgId from filename pattern
