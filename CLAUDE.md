# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A **microservices architecture** for automated PDF generation and storage using message queues. The system processes bank statements through a multi-stage pipeline: file upload → parsing → PDF generation → storage.

### Core Architecture (NEW: Redis Blob Storage Design)

```
                    PostgreSQL (Job State Tracking)
                    Redis (Deduplication Cache + Blob Storage)
                              ↕                    ↕
MinIO (uploads) → File Watcher → [parse_ready] → Parser → [pdf_ready] → PDF Generator → [storage_ready] → Storage Service → MinIO (pdfs)
                      ↕              ↓ metadata      ↓ download blob
                  (Creates jobs)     only (1KB)      from Redis
                  (Uploads blob)                     (Cleanup blob)
```

**Key Components:**
- **PostgreSQL**: Tracks job state (pending → processing → completed/failed), prevents duplicate processing
- **Redis (Dual Purpose)**:
  1. **Deduplication Cache**: Fast-path cache (reduces DB queries by ~80%)
  2. **Blob Storage**: Temporary storage for large files (150MB+) with 1-hour TTL
- **RabbitMQ**: Three queues connecting services - carries **metadata only** (not file content!)
- **MinIO**: S3-compatible storage for uploads and generated PDFs

**NEW Design for Large Files (150MB+):**
- File-watcher downloads from MinIO → uploads to Redis → sends metadata via MQ
- Parser downloads from Redis → processes → deletes from Redis (cleanup)
- **Benefit**: RabbitMQ messages now ~1KB instead of ~200MB (base64 encoded)

## Build & Run Commands

### Start All Services

```bash
# Start all services in Docker
docker-compose up -d

# Build and restart all services
docker-compose up -d --build

# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
docker-compose logs -f parser-service
docker-compose logs -f file-watcher
```

### Scale Services Horizontally

```bash
# Scale PDF generators
docker-compose up -d --scale pdf-generator=3

# Scale storage service
docker-compose up -d --scale storage-service=2
```

### Development - Run Services Locally

**PDF Generator (Node.js):**
```bash
cd pdfme
npm install
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
npm start        # Production
npm run dev      # Development with nodemon
```

**Storage Service (Go):**
```bash
cd storage-service
go mod download
go build -o storage-service ./cmd/storage-service
# Or run directly:
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
export MINIO_ENDPOINT=localhost:9000
go run cmd/storage-service/main.go
```

**File Watcher (Go):**
```bash
cd file-watcher
go mod download
go build -o file-watcher ./cmd/file-watcher
# Or run directly:
go run cmd/file-watcher/main.go
```

**Parser Service (Java/Spring Boot):**
```bash
cd parser
./mvnw clean install
./mvnw spring-boot:run
# Or build JAR:
./mvnw package
java -jar target/parser-0.0.1-SNAPSHOT.jar
```

## Service Architecture

### 1. File Watcher (Go)
- **Location:** `file-watcher/`
- **Purpose:** Gateway - monitors MinIO `uploads` bucket, creates jobs, publishes to parser
- **Queue:** Produces to `parse_ready`
- **Database Access**: Read + Write (creates jobs, checks duplicates, retries stuck jobs)
- **Redis Access**: Read + Write (checks cache before DB, sets processing status)
- **Key Features:**
  - Polls every 10s, batch processes 100 files at 50/sec rate limit
  - Uses S3 ETag (MD5) as file hash for deduplication
  - Checks Redis cache first (fast-path), then PostgreSQL UNIQUE constraint
  - **Download timeout:** 5-minute timeout per file (prevents infinite hangs)
  - **Stuck job detection:** Two-layer protection:
    - Detects jobs stuck in `processing` (>1 hour) and marks for retry
    - Detects jobs stuck in `pending` (>10 minutes) - catches pod crashes during download
  - **Retry logic:** Max 3 attempts per job, then marks as failed
  - Extracts orgID from filename pattern `{orgId}_statement.{ext}`

### 2. Parser Service (Java/Spring Boot)
- **Location:** `parser/`
- **Purpose:** Pure Worker - parses bank statement files (text format) into structured JSON
- **Queues:** Consumes `parse_ready`, produces to `pdf_ready`
- **Database Access**: None (stateless worker)
- **Redis Access**: None
- **Key Files:**
  - `src/main/java/com/afs/parser/service/RabbitMQConsumer.java` - Queue consumer
  - `src/main/java/com/afs/parser/service/StatementParser.java` - Parsing logic
  - `src/main/java/com/afs/parser/Controllers/EStatementController.java` - HTTP API (optional)
- **Output:** EStatementRecord with transactions array
- **Important:** Passes through `job_id` and `file_hash` from input message

### 3. PDF Generator (Node.js)
- **Location:** `pdfme/`
- **Purpose:** Pure Worker - generates multi-page PDFs from templates and data
- **Queues:** Consumes `pdf_ready`, produces to `storage_ready`
- **Database Access**: None (stateless worker)
- **Redis Access**: None
- **Key Files:**
  - `src/services/pdfGenerator.js` - Core PDF generation with pagination
  - `src/services/parserDataTransformer.js` - Transforms parser output to template format
  - `src/services/rabbitmqConsumer.js` - Queue consumer
- **Templates:** Located in `templates/` directory (JSON format from pdfme.com)
- **Key Feature:** Automatic pagination - detects rows by Y-position, splits across pages
- **Important:** Passes through `job_id` and `file_hash` from input message

### 4. Storage Service (Go)
- **Location:** `storage-service/`
- **Purpose:** Finalizer - uploads generated PDFs to MinIO, completes job lifecycle
- **Queue:** Consumes `storage_ready`
- **Database Access**: Read + Write (checks if already completed, updates job status)
- **Redis Access**: Write (sets "completed" status with 24h TTL)
- **Key Features:**
  - Decodes base64 PDFs, creates buckets automatically
  - Idempotent: checks DB before upload to prevent duplicates
  - Updates job status to 'completed' with pdf_location
  - Updates Redis cache for file-watcher's fast-path check

## Message Queue Flow (NEW: Metadata Only!)

### Queue: `parse_ready`
Produced by: File Watcher → Consumed by: Parser
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "file_hash": "d41d8cd98f00b204e9800998ecf8427e",
  "filename": "266_statement.txt",
  "redis_key": "blob:d41d8cd98f00b204e9800998ecf8427e",
  "file_size": 157286400
}
```
**NEW Design:**
- **NO `file_content`** - File stored in Redis as blob!
- **`redis_key`** - Key where parser can download file
- **`file_size`** - File size in bytes (150MB = 157286400 bytes)
- **Benefit:** Message size ~1KB instead of ~200MB
- Parser downloads from Redis using `redis_key`, processes, then deletes blob

### Queue: `pdf_ready`
Produced by: Parser → Consumed by: PDF Generator
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "file_hash": "d41d8cd98f00b204e9800998ecf8427e",
  "orgId": "266",
  "name": "AHMED ADEL HUSAIN ALI",
  "cardNumber": "5117244499894536",
  "statementDate": "21/09/2025",
  "availableBalance": 1026.248,
  "transactions": [
    {
      "date": "06/09/2025",
      "postDate": "06/09/2025",
      "description": "Payment Received",
      "amountInBHD": 149.427,
      "cr": true
    }
  ]
}
```

### Queue: `storage_ready`
Produced by: PDF Generator → Consumed by: Storage Service
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "file_hash": "d41d8cd98f00b204e9800998ecf8427e",
  "bucket_name": "pdfs",
  "filename": "statement_266_123456.pdf",
  "file_content": "base64_encoded_pdf_content"
}
```

## Database & Deduplication

### PostgreSQL Schema

**Primary Table: `processing_jobs`**
```sql
id UUID PRIMARY KEY                    -- Job identifier
file_hash VARCHAR(64) UNIQUE NOT NULL  -- S3 ETag (MD5) - prevents duplicates
filename VARCHAR(255) NOT NULL
status VARCHAR(50)                     -- pending → processing → completed/failed
created_at, processing_started_at, completed_at TIMESTAMP
pdf_location TEXT                      -- S3 path after completion
error_message TEXT
retry_count INT, max_retries INT
```

**Key Indexes:**
- `file_hash` (UNIQUE) - Prevents duplicate job creation
- `status` - Fast status queries
- `processing_started_at WHERE status='processing'` - Find stuck jobs

**Helper Functions:**
- `find_stuck_jobs()` - Returns jobs processing >1 hour
- `mark_job_for_retry(uuid)` - Resets job to pending (max 3 retries)
- `cleanup_old_jobs(days)` - Deletes completed jobs older than N days

**Views:**
- `job_statistics` - Status counts, avg duration (last 24h)
- `failed_jobs` - Failed jobs with error details

### Redis Dual-Purpose Strategy

**Purpose 1: Deduplication Cache**
- **Key Pattern:** `processed:{file_hash}`
- **Values:**
  - `"processing"` - TTL: 1 hour
  - `"completed"` - TTL: 24 hours
- **Deduplication Flow:**
  1. File-watcher checks Redis first (fast-path)
     - `"completed"` → skip (no DB query)
     - `"processing"` → skip (trust TTL, no DB query)
     - `nil` → check PostgreSQL
  2. PostgreSQL INSERT with ON CONFLICT DO NOTHING
  3. Set Redis cache after DB operation
- **Result:** ~80% of duplicate checks skip database entirely

**Purpose 2: Blob Storage (NEW for 150MB+ files)**
- **Key Pattern:** `blob:{file_hash}`
- **Value:** Raw file bytes (not base64!)
- **TTL:** 1 hour (auto-cleanup)
- **Flow:**
  1. File-watcher: Download from MinIO → Upload to Redis as blob
  2. File-watcher: Send metadata to RabbitMQ (not file content!)
  3. Parser: Download blob from Redis → Process → Delete blob
- **Benefits:**
  - RabbitMQ messages: ~1KB (vs ~200MB with base64)
  - Temporary storage: Files auto-expire
  - Fast access: Redis in-memory storage

**Redis Configuration for Large Files:**
```
maxmemory: 2GB
proto-max-bulk-len: 256MB (max single value size)
```

### Job Lifecycle (NEW: With Redis Blob Storage)

```
1. File uploaded to MinIO 'uploads' bucket
   ↓
2. File-watcher:
   - Extract S3 ETag as file_hash
   - Check Redis dedup cache → "completed"? skip
   - Check Redis dedup cache → "processing"? skip
   - Insert into DB (ON CONFLICT DO NOTHING)
   - Download file from MinIO
   - Upload file to Redis blob storage (key: blob:{file_hash}, TTL: 1h)
   - Publish metadata to parse_ready (NOT file content!)
   - Update DB status: 'processing'
   - Set Redis dedup cache: "processing" (1h TTL)
   ↓
3. Parser:
   - Download file from Redis blob (using redis_key)
   - Parse file (no DB access)
   - Delete blob from Redis (cleanup!)
   - Publish to pdf_ready
   ↓
4. PDF Generator: Generate PDF (no DB access)
   ↓
5. Storage Service:
   - Check DB: already completed? skip
   - Upload to MinIO
   - Update DB: status='completed', pdf_location
   - Set Redis dedup cache: "completed" (24h TTL)
```

**Key Changes:**
- File-watcher uploads to Redis blob storage BEFORE publishing to MQ
- RabbitMQ carries only metadata (~1KB), not file content (~200MB)
- Parser downloads from Redis and deletes after processing

**Error Handling:**
- **Stuck jobs in 'processing' (>1 hour):** Detected by file-watcher, retried up to 3 times
- **Stuck jobs in 'pending' (>10 minutes):** Detected by file-watcher (catches pod crashes during download), retried up to 3 times
- **Download timeout:** 5-minute timeout per file prevents infinite hangs, triggers automatic retry
- **Service crashes:** RabbitMQ redelivers, storage checks DB for idempotency
- **Duplicate uploads:** Blocked by UNIQUE constraint + Redis cache
- **Max retries:** After 3 failed attempts, job marked as 'failed' with error message

### Database Access Patterns

**Services with DB Access (2):**
- **File-Watcher**: Creates jobs, checks duplicates, retries stuck jobs
- **Storage**: Finalizes jobs, updates completion status

**Services without DB Access (2):**
- **Parser**: Pure worker, passes through job_id/file_hash
- **PDF Generator**: Pure worker, passes through job_id/file_hash

**Connection Pools:**
- File-Watcher: 10 connections
- Storage: 10 connections
- Parser: 15 connections (includes connection to DB for potential future use)
- PDF Generator: 20 connections

## PDF Template System

### Template Location
Templates are stored in `templates/` as JSON files exported from https://pdfme.com/template-design

### Template Field Naming Convention
For repeating rows (transactions), use numbered fields:
- `trans1_date`, `trans1_description`, `trans1_amount`
- `trans2_date`, `trans2_description`, `trans2_amount`

The PDF generator automatically:
1. Groups fields by Y-position (within 1.0 unit tolerance)
2. Detects rows and pagination structure
3. Splits data across multiple pages based on `itemsPerPage`

### Data Transformation
`parserDataTransformer.js` maps parser output to template fields:
- Flattens transaction arrays into numbered fields
- Maps field names (e.g., `name` → `account_name`)
- Handles pagination metadata

## Service URLs & Credentials

| Service | URL | Credentials |
|---------|-----|-------------|
| RabbitMQ Management | http://localhost:15672 | admin / admin123 |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| PostgreSQL | localhost:5432 | pdfme / pdfme_secure_pass |
| Redis | localhost:6379 | (no auth) |
| PDF Generator API | http://localhost:3000 | - |
| Parser API | http://localhost:8080 | - |

## Key Design Principles

1. **Job State Tracking**: Database stores state, not content (content flows via RabbitMQ)
2. **Two-Service Database Pattern**: Only file-watcher (gateway) and storage (finalizer) access DB
3. **Content-Based Deduplication**: S3 ETag (MD5) prevents duplicate processing regardless of filename
4. **Fast-Path Caching**: Redis reduces DB queries by ~80% with TTL-based cache
5. **Stateless Workers**: Parser and PDF Generator are pure workers, horizontally scalable
6. **Fully Stateless Data Flow**: No local file storage, all data transfers via base64 in messages
7. **Event-Driven**: All communication via RabbitMQ queues
8. **Automatic Pagination**: Position-based row detection in PDF templates
9. **Idempotent Operations**: Storage service checks DB before upload, safe to replay messages
10. **Graceful Degradation**: Services auto-reconnect to RabbitMQ/MinIO/PostgreSQL/Redis on failure

## Package Managers & Dependencies

- **PDF Generator:** npm (Node.js) - uses pdfme library for PDF generation
- **Storage Service:** Go modules - uses minio-go SDK
- **File Watcher:** Go modules - uses minio-go SDK
- **Parser:** Maven (Java) - Spring Boot with AMQP

## Environment Variables

Key environment variables are defined in `docker-compose.yml`. For local development:

**All Services:**
- `RABBITMQ_URL` / `RABBITMQ_HOST`: RabbitMQ connection
- `RABBITMQ_USERNAME` / `RABBITMQ_PASSWORD`: RabbitMQ credentials

**MinIO-dependent Services:**
- `MINIO_ENDPOINT`: MinIO server address
- `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`: MinIO credentials
- `MINIO_USE_SSL`: Enable/disable SSL

**Service-specific:**
- `BUCKET_NAME`: Target bucket name
- `POLL_INTERVAL`: File watcher polling interval
- `DEFAULT_BUCKET`: Default PDF output bucket

## Monitoring & Operations

### Database Queries

**Check job status:**
```sql
-- Status breakdown
SELECT status, COUNT(*) FROM processing_jobs GROUP BY status;

-- Statistics (last 24h)
SELECT * FROM job_statistics;

-- Failed jobs
SELECT * FROM failed_jobs;

-- Find stuck jobs in 'processing' (>1 hour)
SELECT * FROM find_stuck_jobs();

-- Find stuck jobs in 'pending' (>10 minutes) - catches pod crashes
SELECT * FROM find_stuck_pending_jobs();
```

**Manage jobs:**
```sql
-- Retry a stuck job
SELECT mark_job_for_retry('job-uuid-here');

-- Cleanup old completed jobs (90+ days)
SELECT cleanup_old_jobs(90);
```

**Access database:**
```bash
# Via psql
docker-compose exec postgres psql -U pdfme -d pdfme

# Via docker-compose
docker-compose exec postgres psql -U pdfme -d pdfme -c "SELECT * FROM job_statistics"
```

### Redis Monitoring

**Check cache status:**
```bash
# Count cached files
docker-compose exec redis redis-cli DBSIZE

# List processed files
docker-compose exec redis redis-cli KEYS "processed:*"

# Check specific file status
docker-compose exec redis redis-cli GET "processed:d41d8cd98f00b204e9800998ecf8427e"

# Check TTL
docker-compose exec redis redis-cli TTL "processed:d41d8cd98f00b204e9800998ecf8427e"

# Memory usage
docker-compose exec redis redis-cli INFO memory

# Clear cache (if needed)
docker-compose exec redis redis-cli FLUSHDB
```

### RabbitMQ Monitoring

**Management UI:** http://localhost:15672
- Queue depths (parse_ready, pdf_ready, storage_ready)
- Consumer connections and rates
- Message rates (publish/deliver/ack)

**CLI:**
```bash
# Queue status
docker-compose exec rabbitmq rabbitmqctl list_queues name messages consumers

# Connection status
docker-compose exec rabbitmq rabbitmqctl list_connections
```

### Service Health

**Check all services:**
```bash
docker-compose ps
docker-compose logs -f
```

**Check specific service:**
```bash
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
docker-compose logs -f file-watcher
docker-compose logs -f parser-service
```

## Troubleshooting

### Service Won't Start
```bash
docker-compose ps              # Check service status
docker-compose logs <service>  # Check error logs
docker-compose restart <service>
```

### Messages Not Processing
1. Check RabbitMQ UI - are consumers connected?
2. Check service logs for errors
3. Verify message format matches queue schema (includes job_id/file_hash)
4. Ensure all dependencies are healthy: RabbitMQ, MinIO, PostgreSQL, Redis

### Duplicate Files Being Processed
1. Check Redis cache: `docker-compose exec redis redis-cli GET "processed:{file_hash}"`
2. Check database: `SELECT * FROM processing_jobs WHERE file_hash = '{hash}'`
3. Verify file-watcher is setting cache after DB operations
4. Check storage service is setting "completed" cache

### Jobs Stuck in Processing or Pending
**Jobs stuck in 'processing' (>1 hour):**
1. Run: `SELECT * FROM find_stuck_jobs();` in PostgreSQL
2. Retry job: `SELECT mark_job_for_retry('{job_id}');`
3. File-watcher automatically detects and retries every scan (10s interval)

**Jobs stuck in 'pending' (>10 minutes) - Pod Crashes:**
1. Run: `SELECT * FROM find_stuck_pending_jobs();` in PostgreSQL
2. Retry job: `SELECT mark_job_for_retry('{job_id}');`
3. Common cause: File-watcher pod crashed during MinIO download
4. File-watcher automatically detects and retries every scan (10s interval)

**Download Timeout:**
- Each file has 5-minute download timeout
- After timeout, job marked for retry automatically
- Check logs for "Download timeout after 5 minutes" messages

### Database Connection Issues
1. Check PostgreSQL is running: `docker-compose ps postgres`
2. Verify connection string in docker-compose.yml
3. Check connection pool settings (file-watcher: 10, storage: 10)
4. View logs: `docker-compose logs postgres`

### Redis Connection Issues
1. Check Redis is running: `docker-compose ps redis`
2. Test connection: `docker-compose exec redis redis-cli PING`
3. Check memory usage: `docker-compose exec redis redis-cli INFO memory`
4. If full, increase maxmemory or clean old keys

### Reset Everything
```bash
docker-compose down -v  # Remove volumes
docker-compose up -d    # Fresh start
```

## Additional Documentation

- `README.md` - Comprehensive getting started guide
- `DESIGN.md` - Detailed system design and architecture decisions
- `init-db-simplified.sql` - PostgreSQL schema with helper functions
- `docs/message-flow.md` - Detailed message queue flow
- `docs/template-mapping.md` - Template field mapping details
- `docs/parser.md` - Parser service documentation
- `docs/pdfme.md` - PDF generator documentation
- `docs/file-watcher.md` - File watcher with database integration
- `docs/storage-service.md` - Storage service with database integration
- `parser/CHANGES.md` - Parser RabbitMQ integration changes

## Scaling Recommendations

### Month-End Processing (5000 files)

**Recommended Configuration:**
```yaml
file-watcher: 1-3 replicas    # Database UNIQUE constraint prevents race conditions
parser: 5-10 replicas          # CPU-bound (text parsing)
pdf-generator: 10-20 replicas  # CPU-bound, slowest step (PDF rendering)
storage: 1-3 replicas          # I/O-bound (MinIO uploads)

postgres: 400 max connections
redis: 512MB memory, allkeys-lru eviction
rabbitmq: Prefetch=1 per consumer
```

**Expected Performance:**
- File queuing: ~2 minutes (50 files/sec rate limit)
- End-to-end processing: ~30-40 minutes with scaling
- Database load: ~20,000 queries total (4 per file: 2 from file-watcher, 2 from storage)
- Redis hit rate: ~80% (reduces DB load significantly)

**Key Scaling Points:**
- Parser/PDF Generator scale horizontally (no shared state)
- File-watcher can scale to 3+ (DB UNIQUE prevents duplicate jobs)
- Storage can scale to 3+ (idempotent operations)
- Database is bottleneck-free: only 2 services, minimal queries
- Redis cache is critical: enables fast duplicate detection without DB queries
