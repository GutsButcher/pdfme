# Developer Quick Start - Infrastructure Only

## TL;DR

```bash
# Start all infrastructure services
docker-compose -f docker-compose-infra.yml up -d

# Verify running
docker-compose -f docker-compose-infra.yml ps

# Stop everything
docker-compose -f docker-compose-infra.yml down
```

## What You Get

✅ **RabbitMQ** on `localhost:5672` (UI: http://localhost:15672)
✅ **MinIO** on `localhost:9000` (UI: http://localhost:9001)
✅ **PostgreSQL** on `localhost:5432`
✅ **Valkey (Redis)** on `localhost:6379`

## Connection Strings (Copy-Paste)

### Environment Variables

```bash
# RabbitMQ
export RABBITMQ_URL="amqp://admin:admin123@localhost:5672"
export RABBITMQ_HOST="localhost"
export RABBITMQ_PORT="5672"
export RABBITMQ_USERNAME="admin"
export RABBITMQ_PASSWORD="admin123"

# MinIO
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ROOT_USER="minioadmin"
export MINIO_ROOT_PASSWORD="minioadmin"
export MINIO_USE_SSL="false"

# PostgreSQL
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5432"
export POSTGRES_USER="pdfme"
export POSTGRES_PASSWORD="pdfme_secure_pass"
export POSTGRES_DB="pdfme"

# Valkey/Redis
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
```

### .env File

```bash
# Save to .env
cat > .env << 'EOF'
RABBITMQ_URL=amqp://admin:admin123@localhost:5672
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USERNAME=admin
RABBITMQ_PASSWORD=admin123

MINIO_ENDPOINT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_USE_SSL=false

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=pdfme
POSTGRES_PASSWORD=pdfme_secure_pass
POSTGRES_DB=pdfme

REDIS_HOST=localhost
REDIS_PORT=6379
EOF
```

## Access UIs

| Service | URL | Credentials |
|---------|-----|-------------|
| RabbitMQ Management | http://localhost:15672 | admin / admin123 |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| PostgreSQL | localhost:5432 | pdfme / pdfme_secure_pass |
| Valkey/Redis | localhost:6379 | (no auth) |

## Test Connections

```bash
# Test RabbitMQ
curl -u admin:admin123 http://localhost:15672/api/overview

# Test MinIO
curl http://localhost:9000/minio/health/live

# Test PostgreSQL
docker-compose -f docker-compose-infra.yml exec postgres psql -U pdfme -d pdfme -c "SELECT version();"

# Test Valkey/Redis
docker-compose -f docker-compose-infra.yml exec valkey valkey-cli PING
```

## Create MinIO Buckets

```bash
# Install MinIO client (if not already installed)
# macOS: brew install minio/stable/mc
# Linux: wget https://dl.min.io/client/mc/release/linux-amd64/mc && chmod +x mc

# Or use Docker
docker run --rm --network pdfme-dev-network -it minio/mc \
  alias set local http://minio:9000 minioadmin minioadmin

docker run --rm --network pdfme-dev-network -it minio/mc \
  mb local/uploads

docker run --rm --network pdfme-dev-network -it minio/mc \
  mb local/pdfs
```

## Common Commands

```bash
# View logs
docker-compose -f docker-compose-infra.yml logs -f

# Restart a service
docker-compose -f docker-compose-infra.yml restart rabbitmq

# Check health
docker-compose -f docker-compose-infra.yml ps

# Clean reset (DELETES ALL DATA!)
docker-compose -f docker-compose-infra.yml down -v
docker-compose -f docker-compose-infra.yml up -d
```

## Troubleshooting

**Services not starting?**
```bash
# Check ports are available
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Valkey/Redis
lsof -i :5672  # RabbitMQ
lsof -i :9000  # MinIO
```

**Can't connect from application?**
- Use `localhost` (NOT `rabbitmq`, `minio`, etc.)
- Check services are running: `docker-compose -f docker-compose-infra.yml ps`

**Need to reset everything?**
```bash
docker-compose -f docker-compose-infra.yml down -v && \
docker-compose -f docker-compose-infra.yml up -d
```

---

For detailed documentation, see `INFRASTRUCTURE.md`
