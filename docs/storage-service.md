# Storage Service

**Technology**: Go
**Container**: `pdfme-storage`

## Function

Finalizer service that uploads PDFs to MinIO, updates job completion in database, and updates Redis cache.

## Key Features

- **Idempotency**: Checks DB before processing (skip if completed)
- **Job finalization**: Marks jobs as complete with PDF location
- **Redis update**: Marks file as 'completed' for file-watcher cache
- **Error tracking**: Updates job with error message on failure

## Configuration

```yaml
# RabbitMQ
RABBITMQ_URL: amqp://admin:admin123@rabbitmq:5672
QUEUE_NAME: storage_ready

# MinIO
MINIO_ENDPOINT: minio:9000
MINIO_ROOT_USER: minioadmin
MINIO_ROOT_PASSWORD: minioadmin
MINIO_USE_SSL: false

# PostgreSQL
POSTGRES_HOST: postgres
POSTGRES_PORT: 5432
POSTGRES_USER: pdfme
POSTGRES_PASSWORD: pdfme_secure_pass
POSTGRES_DB: pdfme
POSTGRES_MAX_POOL_SIZE: 10

# Redis
REDIS_HOST: redis
REDIS_PORT: 6379
```

## Processing Flow

```
Consume from storage_ready:
1. Parse message (job_id, file_hash, PDF content)
2. Check DB: job already completed? → skip (idempotency)
3. Decode base64 PDF
4. Ensure bucket exists
5. Upload to MinIO
6. Update DB:
   - status = 'completed'
   - completed_at = NOW()
   - pdf_location = 'bucket/filename'
7. Update Redis: 'completed' (24h TTL)
8. ACK message
```

## Database Operations

**Check completion:**
```sql
SELECT status FROM processing_jobs WHERE id = $1
```

**Mark complete:**
```sql
UPDATE processing_jobs
SET status = 'completed',
    completed_at = NOW(),
    pdf_location = $1
WHERE id = $2
```

**Mark failed:**
```sql
UPDATE processing_jobs
SET status = 'failed',
    error_message = $1
WHERE id = $2
```

## Redis Operations

**Update cache:**
```
SETEX processed:{file_hash} 86400 "completed"
```

## Input

From `storage_ready` queue:
```json
{
  "job_id": "uuid",
  "file_hash": "md5-hash",
  "bucket_name": "pdfs",
  "filename": "statement_266_1234567.pdf",
  "file_content": "base64-encoded-pdf..."
}
```

## Output

- PDF uploaded to MinIO
- Job marked complete in database
- Redis cache updated
