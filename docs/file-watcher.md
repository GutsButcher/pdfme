# File-Watcher Service

**Technology**: Go
**Container**: `pdfme-file-watcher`

## Function

Gateway service that monitors S3 for new files, creates jobs in database, and publishes to RabbitMQ.

## Key Features

- **S3 ETag hashing**: Uses S3 object ETag (MD5) as file hash - no download needed for dedup check
- **Redis fast-path**: Checks cache before database
- **Atomic job creation**: `ON CONFLICT DO NOTHING` prevents race conditions
- **Timeout detection**: Finds jobs stuck >1 hour, marks for retry
- **Rate limiting**: 50 files/second
- **Batch processing**: 100 files per batch
- **Multiple pods safe**: Database UNIQUE constraint prevents duplicates

## Configuration

```yaml
# RabbitMQ
RABBITMQ_URL: amqp://admin:admin123@rabbitmq:5672
QUEUE_NAME: parse_ready

# MinIO
BUCKET_NAME: uploads
POLL_INTERVAL: 10s
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

# Processing
BATCH_SIZE: 100
RATE_LIMIT_PER_SECOND: 50
```

## Processing Flow

```
Every 10 seconds:
1. List S3 objects (metadata only)
2. For each file:
   a. Use ETag as file_hash
   b. Check Redis: processed:{hash}?
      → "completed": skip
      → "processing": check timeout
      → nil: continue
   c. Try INSERT into processing_jobs
      → Duplicate: skip
      → Success: continue
   d. Download file (only if new)
   e. Publish to parse_ready
   f. Update status: 'processing'
   g. Set Redis cache (1h TTL)

Stuck job detection:
3. Find jobs with processing_started_at > 1h
4. Retry count < max (3)?
   → Yes: Reset to 'pending', clear cache
   → No: Mark as 'failed'
```

## Database Operations

**Creates jobs:**
```sql
INSERT INTO processing_jobs (file_hash, filename, status)
VALUES ($1, $2, 'pending')
ON CONFLICT (file_hash) DO NOTHING
RETURNING id
```

**Updates status:**
```sql
UPDATE processing_jobs
SET status = 'processing',
    processing_started_at = NOW()
WHERE id = $1
```

**Finds stuck jobs:**
```sql
SELECT * FROM find_stuck_jobs()
```

## Redis Operations

**Check cache:**
```
GET processed:{file_hash}
```

**Set cache:**
```
SETEX processed:{file_hash} 3600 "processing"
```

## Output

Publishes to `parse_ready` queue:
```json
{
  "job_id": "uuid",
  "file_hash": "md5-hash",
  "filename": "266003.txt",
  "file_content": "base64..."
}
```
