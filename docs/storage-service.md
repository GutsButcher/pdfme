# Storage Service

**Technology**: Go
**Container**: `pdfme-storage`

## Function
Decodes PDF → uploads to MinIO

## Configuration

```yaml
RABBITMQ_URL: amqp://admin:admin123@rabbitmq:5672
QUEUE_NAME: storage_ready
MINIO_ENDPOINT: minio:9000
MINIO_ROOT_USER: minioadmin
MINIO_ROOT_PASSWORD: minioadmin
MINIO_USE_SSL: false
```

## Input

**Source**: RabbitMQ queue `storage_ready`

**Message Format**:
```json
{
  "bucket_name": "pdfs",
  "filename": "266_4536_542462.pdf",
  "file_content": "JVBERi0xLjQK..."
}
```

**Fields**:
- `bucket_name` (string): Target MinIO bucket
- `filename` (string): PDF filename
- `file_content` (string): Base64-encoded PDF content

## Output

**Destination**: MinIO bucket specified in `bucket_name`

**Location**: `{bucket_name}/{filename}`

**Content-Type**: `application/pdf`

## Behavior
- Auto-creates bucket if doesn't exist
- Decodes base64 to binary
- Uploads to MinIO
- Acknowledges message after successful upload
