# System Design

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          PostgreSQL (Job State)                      │
│                    Redis (Deduplication Cache)                       │
└─────────────────────────────────────────────────────────────────────┘
         ↑                                                      ↑
         │ create/check                                         │ update
         │                                                      │
┌────────────────┐         ┌────────────┐         ┌────────────────────┐
│  File Watcher  │────────▶│  RabbitMQ  │────────▶│  Storage Service   │
│  (Gateway)     │         │ (3 queues) │         │  (Finalizer)       │
└────────────────┘         └────────────┘         └────────────────────┘
                                │    │
                                │    │
                       ┌────────┘    └────────┐
                       ▼                      ▼
                  ┌─────────┐          ┌──────────────┐
                  │ Parser  │          │ PDF Generator│
                  │(Worker) │          │  (Worker)    │
                  └─────────┘          └──────────────┘
```

## Design Principles

### 1. Job State Tracking Only
- Database stores **state**, not content
- Content flows through RabbitMQ (base64)
- Single table: `processing_jobs`

### 2. Two-Service Database Pattern
- **File-Watcher**: Creates jobs, checks duplicates
- **Storage**: Finalizes jobs, updates completion
- **Parser/PDF Generator**: Pure stateless workers (no DB)

### 3. Deduplication Strategy
- **S3 ETag** as file hash (MD5, no download needed)
- **PostgreSQL UNIQUE constraint**: Prevents duplicate jobs
- **Redis cache**: Fast-path optimization
  - "completed" (24h TTL): Skip, no DB query
  - "processing" (1h TTL): Skip, no DB query (trust TTL)
  - Stuck job detection via separate CheckStuckJobs()

### 4. Natural Throttling
- RabbitMQ: One message per consumer (prefetch=1)
- File-watcher: Batch processing (100 files), rate limiting (50/sec)
- Database: Minimal load (only 2 services, ~20 connections)

## Data Flow

```
1. Upload to S3 'uploads' bucket
   ↓
2. File-Watcher (every 10s):
   - List S3 files (metadata only, uses ETag as hash)
   - Check Redis:
     → "completed"? → skip (no DB query)
     → "processing"? → skip (< 1h by TTL, no DB query)
     → nil? → continue to DB
   - Try INSERT into DB (ON CONFLICT DO NOTHING)
     → Duplicate? → Check status for retry logic
     → New? → download file, publish to MQ
   - Update status: 'processing'
   - Set Redis cache: "processing" (1h TTL)
   ↓
3. Parser:
   - Consume from parse_ready
   - Parse file content
   - Pass through: job_id, file_hash
   - Publish to pdf_ready
   ↓
4. PDF Generator:
   - Consume from pdf_ready
   - Generate PDF
   - Pass through: job_id, file_hash
   - Publish to storage_ready
   ↓
5. Storage:
   - Consume from storage_ready
   - Check DB: already completed? → skip
   - Upload to MinIO
   - Update DB: status='completed', pdf_location
   - Update Redis: 'completed' (24h TTL)
   - ACK message
```

## Database Schema

```sql
processing_jobs
├── id (UUID, PK)
├── file_hash (VARCHAR(64), UNIQUE) -- S3 ETag (MD5)
├── filename (VARCHAR(255))
├── status (VARCHAR(50)) -- pending → processing → completed/failed
├── created_at, processing_started_at, completed_at
├── pdf_location (TEXT) -- S3 path after upload
└── error_message, retry_count, max_retries
```

**Indexes:**
- `file_hash` (UNIQUE - prevents duplicates)
- `status` (query by status)
- `processing_started_at` WHERE status='processing' (find stuck jobs)

## Redis Keys

```
processed:{file_hash}
├── Value: "processing" or "completed"
├── TTL: 1h (processing) or 24h (completed)
└── Purpose: Fast deduplication check
```

## Message Formats

### parse_ready
```json
{
  "job_id": "uuid",
  "file_hash": "md5-hash",
  "filename": "266003.txt",
  "file_content": "base64..."
}
```

### pdf_ready
```json
{
  "job_id": "uuid",
  "file_hash": "md5-hash",
  "orgId": "266",
  "name": "AHMED ADEL...",
  "transactions": [...]
}
```

### storage_ready
```json
{
  "job_id": "uuid",
  "file_hash": "md5-hash",
  "bucket_name": "pdfs",
  "filename": "statement_266_1234567.pdf",
  "file_content": "base64..."
}
```

## Error Handling

### Stuck Jobs (>1 hour)
- File-watcher detects via DB query
- Marks for retry (max 3 attempts)
- After max retries: status='failed'

### Service Crash
- RabbitMQ redelivers message
- Storage checks DB: already completed? → skip
- Idempotency guaranteed

### Duplicate Files
- Same file uploaded twice
- Redis cache hit → skip (fast path)
- Or DB UNIQUE constraint violation → skip
- No duplicate processing

## Scaling

### Month-End Processing (5000 files)

**Recommended Configuration:**
```yaml
file-watcher: 1-3 pods   # DB UNIQUE prevents race conditions
parser: 5-10 pods        # CPU-bound (parsing)
pdf-generator: 10-20 pods # CPU-bound, slowest step
storage: 1-3 pods        # I/O-bound (MinIO uploads)
```

**Database Load:**
- File-watcher: ~2 queries per file (INSERT + UPDATE)
- Storage: ~2 queries per file (SELECT + UPDATE)
- **Total: ~20,000 queries** for 5000 files (trivial)

**Processing Time:**
- File queuing: ~2 minutes (50 files/sec)
- End-to-end: ~30-40 minutes (with scaling)

## Services

### File-Watcher (Go)
**Role**: Gateway
- Monitors S3 for new files
- Creates jobs in database
- Publishes to parse_ready queue
- Detects and retries stuck jobs

**DB Access**: Read + Write
**Redis Access**: Read + Write

### Parser (Java/Spring Boot)
**Role**: Pure Worker
- Parses text files
- No database access
- Passes through job_id/file_hash

**DB Access**: None
**Redis Access**: None

### PDF Generator (Node.js)
**Role**: Pure Worker
- Generates multi-page PDFs
- No database access
- Passes through job_id/file_hash

**DB Access**: None
**Redis Access**: None

### Storage (Go)
**Role**: Finalizer
- Uploads PDFs to MinIO
- Updates job completion in DB
- Updates Redis cache for file-watcher

**DB Access**: Read + Write
**Redis Access**: Write

## Monitoring

### Database Queries

```sql
-- Job status breakdown
SELECT status, COUNT(*) FROM processing_jobs GROUP BY status;

-- Statistics
SELECT * FROM job_statistics;

-- Failed jobs
SELECT * FROM failed_jobs;

-- Find stuck jobs
SELECT * FROM find_stuck_jobs();

-- Retry a job
SELECT mark_job_for_retry('job-uuid');

-- Cleanup old jobs (90+ days)
SELECT cleanup_old_jobs(90);
```

### Redis Monitoring

```bash
# Check cached files
docker-compose exec redis redis-cli KEYS "processed:*"

# Check memory usage
docker-compose exec redis redis-cli INFO memory
```

### RabbitMQ Monitoring
- UI: http://localhost:15672
- Monitor queue depths
- Check consumer connections

## Configuration

### PostgreSQL
- Max connections: 400
- Shared buffers: 512MB
- Effective cache size: 1536MB

### Redis
- Max memory: 512MB
- Eviction policy: allkeys-lru
- Persistence: AOF + RDB snapshots

### RabbitMQ
- Prefetch: 1 (process one at a time)
- Durable queues
- Manual acknowledgment

## Key Features

✅ **Content-based deduplication** - Hash prevents duplicates regardless of filename
✅ **Minimal database load** - Only 2 services access DB
✅ **Fast-path caching** - Redis reduces DB queries by ~80%
✅ **Stateless workers** - Parser/PDF Generator scale independently
✅ **Automatic retry** - Stuck jobs detected and retried
✅ **Horizontal scaling** - All services scale independently
✅ **No data loss** - DB tracks every job through pipeline
