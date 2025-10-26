# User Manual

## System Overview

Automated PDF generation pipeline with job state tracking and duplicate prevention.

**Services**: RabbitMQ, MinIO, PostgreSQL, Redis, File Watcher, Parser, PDF Generator, Storage

## Quick Start

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps
```

## File Upload & Processing

### Method 1: MinIO Console (Recommended)

1. **Access**: http://localhost:9001
   - Username: `minioadmin`
   - Password: `minioadmin`

2. **Upload** to `uploads` bucket
   - Any filename (e.g., `266003.txt`, `statement_jan.txt`)
   - System uses file hash, not name

3. **Wait** ~10 seconds (file-watcher polls every 10s)

4. **Download PDF** from `pdfs` bucket
   - Filename: `statement_{orgId}_{timestamp}.pdf`

### Method 2: CLI Upload

```bash
# Using MinIO client in container
docker-compose exec minio mc cp /path/to/file.txt local/uploads/

# Or use script
./test_upload_file.sh test_data/266003.txt
```

### Method 3: HTTP API (Direct PDF)

```bash
curl -X POST http://localhost:3000/pdf/parser \
  -H "Content-Type: application/json" \
  -d @test_data/parser_test.json \
  --output statement.pdf
```

## Monitoring

### Job Status (Database)

```bash
# Check processing jobs
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT filename, status,
         EXTRACT(EPOCH FROM (completed_at - created_at)) as duration_sec
  FROM processing_jobs
  ORDER BY created_at DESC
  LIMIT 10;
"

# Job statistics (last 24h)
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT * FROM job_statistics;
"

# Failed jobs
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT * FROM failed_jobs;
"

# Find stuck jobs (>1 hour)
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT * FROM find_stuck_jobs();
"
```

### Service Logs

```bash
# Watch all logs
docker-compose logs -f

# Specific service
docker-compose logs -f file-watcher
docker-compose logs -f parser-service
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
```

### RabbitMQ Management

**URL**: http://localhost:15672
- Username: `admin`
- Password: `admin123`

**Monitor**:
- Queue depths
- Message rates
- Consumer connections

### MinIO Console

**URL**: http://localhost:9001
- Username: `minioadmin`
- Password: `minioadmin`

**Buckets**:
- `uploads` - Input files
- `pdfs` - Generated PDFs

### Redis Cache

```bash
# Check cached files
docker-compose exec redis redis-cli KEYS "processed:*"

# Check specific file
docker-compose exec redis redis-cli GET "processed:{file-hash}"

# Memory usage
docker-compose exec redis redis-cli INFO memory
```

## Duplicate Prevention

### How It Works

Files are identified by **content hash** (S3 ETag), not filename.

**Scenarios:**

1. **Same file, same name** → Skipped (Redis cache)
2. **Same file, different name** → Skipped (hash matches)
3. **Different file, same name** → Processed (different hash)

**Example:**
```bash
# Upload file
mc cp file.txt local/uploads/266003.txt
→ Processed

# Upload again (same name)
mc cp file.txt local/uploads/266003.txt
→ Skipped (duplicate)

# Upload with different name
mc cp file.txt local/uploads/DIFFERENT_NAME.txt
→ Skipped (same hash)
```

## Service Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| RabbitMQ Mgmt | http://localhost:15672 | admin / admin123 |
| PDF Generator | http://localhost:3000 | - |
| Parser API | http://localhost:8080 | - |
| PostgreSQL | localhost:5432 | pdfme / pdfme_secure_pass |
| Redis | localhost:6379 | - |

## Troubleshooting

### File Not Processing

**Check each stage:**

```bash
# 1. File-watcher detected it?
docker-compose logs file-watcher | grep "your-file.txt"

# 2. Check job status in DB
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT * FROM processing_jobs WHERE filename='your-file.txt';
"

# 3. Check queue
# Visit http://localhost:15672 → Queues tab

# 4. Parser logs
docker-compose logs parser-service | tail -50

# 5. PDF generator logs
docker-compose logs pdf-generator | tail -50

# 6. Storage logs
docker-compose logs storage-service | tail -50
```

### Stuck Job (>1 hour)

```bash
# Find stuck jobs
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT * FROM find_stuck_jobs();
"

# Manually retry
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT mark_job_for_retry('job-uuid-here');
"

# Or restart file-watcher (auto-detects stuck jobs)
docker-compose restart file-watcher
```

### Reset System

```bash
# Stop all services
docker-compose down

# Clear volumes (WARNING: deletes all data!)
docker-compose down -v

# Fresh start
docker-compose up -d --build
```

### Clear Completed Jobs

```bash
# Delete jobs older than 90 days
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT cleanup_old_jobs(90);
"
```

## Scaling

### For Month-End Batch (5000 files)

```bash
# Scale services
docker-compose up -d \
  --scale file-watcher=2 \
  --scale parser-service=5 \
  --scale pdf-generator=10 \
  --scale storage-service=2

# Monitor progress
docker-compose exec postgres psql -U pdfme -d pdfme -c "
  SELECT status, COUNT(*) FROM processing_jobs GROUP BY status;
"
```

## Maintenance

### Database Backup

```bash
# Backup
docker-compose exec postgres pg_dump -U pdfme pdfme > backup_$(date +%Y%m%d).sql

# Restore
cat backup_20241026.sql | docker-compose exec -T postgres psql -U pdfme pdfme
```

### Redis Backup

Redis automatically persists:
- AOF (append-only file): Real-time
- RDB snapshot: Every 60s if 1000 keys changed

### Template Management

Templates stored in `templates/` directory.

**Add new template:**
1. Design at https://pdfme.com/template-design
2. Export JSON
3. Copy to `templates/your-template.json`
4. Restart: `docker-compose restart pdf-generator`

## Technical Details

See:
- `DESIGN.md` - System architecture and design principles
- `README.md` - Project overview and quick start
- `docs/` - Service-specific documentation
