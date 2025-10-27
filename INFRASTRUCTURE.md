# Infrastructure Services for Local Development

This document explains how to run the infrastructure services (databases, message queues, storage) for local development.

## What's Included

This `docker-compose-infra.yml` provides **ONLY the backing services** - no application code:

| Service | Purpose | Port | UI/Credentials |
|---------|---------|------|----------------|
| **RabbitMQ** | Message broker | 5672 | UI: http://localhost:15672<br>User: `admin` / Pass: `admin123` |
| **MinIO** | S3-compatible storage | 9000 | Console: http://localhost:9001<br>User: `minioadmin` / Pass: `minioadmin` |
| **PostgreSQL** | Job state database | 5432 | User: `pdfme` / Pass: `pdfme_secure_pass`<br>DB: `pdfme` |
| **Valkey** | Redis-compatible cache | 6379 | Compatible with Redis clients |

## Quick Start

### 1. Start Infrastructure

```bash
docker-compose -f docker-compose-infra.yml up -d
```

### 2. Verify All Services Running

```bash
docker-compose -f docker-compose-infra.yml ps
```

All services should show "Up (healthy)".

### 3. View Logs

```bash
# All services
docker-compose -f docker-compose-infra.yml logs -f

# Specific service
docker-compose -f docker-compose-infra.yml logs -f rabbitmq
```

### 4. Stop Infrastructure

```bash
docker-compose -f docker-compose-infra.yml down
```

### 5. Clean Reset (Remove Data)

```bash
docker-compose -f docker-compose-infra.yml down -v
```

---

## Connection Details for Your Application

### RabbitMQ

```bash
# Connection URL
RABBITMQ_URL=amqp://admin:admin123@localhost:5672

# Or separate variables
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USERNAME=admin
RABBITMQ_PASSWORD=admin123
```

**Management UI:** http://localhost:15672

### MinIO (S3-Compatible)

```bash
# Connection
MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_USE_SSL=false
```

**Console:** http://localhost:9001

**Create Buckets:**
```bash
# Using MinIO client (mc)
docker-compose -f docker-compose-infra.yml exec minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker-compose -f docker-compose-infra.yml exec minio mc mb local/uploads
docker-compose -f docker-compose-infra.yml exec minio mc mb local/pdfs
```

### PostgreSQL

```bash
# Connection
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=pdfme
POSTGRES_PASSWORD=pdfme_secure_pass
POSTGRES_DB=pdfme
```

**Connect via psql:**
```bash
docker-compose -f docker-compose-infra.yml exec postgres psql -U pdfme -d pdfme
```

**Database is auto-initialized** with schema from `init-db-simplified.sql`.

### Valkey (Redis-Compatible)

```bash
# Connection
REDIS_HOST=localhost
REDIS_PORT=6379
```

**Test Connection:**
```bash
docker-compose -f docker-compose-infra.yml exec valkey valkey-cli ping
# Should return: PONG
```

**Valkey is Redis-compatible** - use any Redis client library!

---

## Architecture Notes

### Valkey (Redis) Dual Purpose

**1. Deduplication Cache:**
- Keys: `processed:{file_hash}`
- Values: `"processing"` or `"completed"`
- TTL: 1h (processing) or 24h (completed)

**2. Blob Storage (Large Files 150MB+):**
- Keys: `blob:{file_hash}`
- Values: Raw file bytes
- TTL: 1 hour
- Config: Max value size 256MB

### RabbitMQ Queues

Your application should create these queues:
- `parse_ready` - File metadata for parser
- `pdf_ready` - Parsed data for PDF generator
- `storage_ready` - PDFs for storage

### PostgreSQL Schema

Schema is automatically created on first startup from `init-db-simplified.sql`:
- Table: `processing_jobs`
- Functions: `find_stuck_jobs()`, `find_stuck_pending_jobs()`, `mark_job_for_retry()`
- Views: `job_statistics`, `failed_jobs`

---

## Monitoring

### Check Service Health

```bash
# All services health
docker-compose -f docker-compose-infra.yml ps

# RabbitMQ queues
docker-compose -f docker-compose-infra.yml exec rabbitmq rabbitmqctl list_queues

# PostgreSQL connections
docker-compose -f docker-compose-infra.yml exec postgres psql -U pdfme -d pdfme -c "SELECT * FROM job_statistics"

# Valkey keys
docker-compose -f docker-compose-infra.yml exec valkey valkey-cli DBSIZE
```

### View Logs

```bash
# Tail all logs
docker-compose -f docker-compose-infra.yml logs -f

# Specific service
docker-compose -f docker-compose-infra.yml logs -f postgres
docker-compose -f docker-compose-infra.yml logs -f valkey
```

---

## Troubleshooting

### Port Already in Use

If ports are already taken, edit `docker-compose-infra.yml` and change the left side of port mappings:

```yaml
ports:
  - "5433:5432"  # Changed PostgreSQL from 5432 to 5433
```

Then update your application connection strings accordingly.

### Reset Everything

```bash
# Stop and remove all data
docker-compose -f docker-compose-infra.yml down -v

# Start fresh
docker-compose -f docker-compose-infra.yml up -d
```

### Can't Connect from Application

1. **Check services are running:**
   ```bash
   docker-compose -f docker-compose-infra.yml ps
   ```

2. **Check network:**
   ```bash
   docker network ls | grep pdfme-dev-network
   ```

3. **Use `localhost` (NOT container names)** when connecting from your local application:
   - ✅ `RABBITMQ_HOST=localhost`
   - ❌ `RABBITMQ_HOST=rabbitmq` (this only works inside Docker network)

---

## Network Information

- **Network Name:** `pdfme-dev-network`
- **Driver:** bridge
- **All services are on the same network** for inter-container communication

If your application is also running in Docker and needs to connect to these services, you can join this network:

```yaml
# In your application's docker-compose.yml
networks:
  default:
    external: true
    name: pdfme-dev-network
```

---

## Configuration

### Increase Valkey/Redis Memory

Edit `docker-compose-infra.yml`:

```yaml
valkey:
  command: >
    valkey-server
    --maxmemory 4gb  # Increased from 2gb
```

### Increase PostgreSQL Connections

Edit `docker-compose-infra.yml`:

```yaml
postgres:
  command: >
    postgres
    -c max_connections=800  # Increased from 400
```

---

## Production Notes

**This infrastructure setup is for LOCAL DEVELOPMENT only!**

For production:
1. Change all default passwords
2. Enable SSL/TLS
3. Configure proper backups
4. Use managed services (RDS, ElastiCache, etc.)
5. Implement monitoring and alerting
6. Configure proper resource limits

---

## Support

See main `README.md` and `CLAUDE.md` for full architecture documentation.
