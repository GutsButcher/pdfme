# Deployment Guide

## Quick Start

```bash
docker-compose up -d --build
```

## Service Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| RabbitMQ Management | http://localhost:15672 | admin / admin123 |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| PDF Generator API | http://localhost:3000 | - |
| Parser API | http://localhost:8080 | - |

## Services

| Container | Image | Ports | Dependencies |
|-----------|-------|-------|--------------|
| pdfme-rabbitmq | rabbitmq:3.12-management-alpine | 5672, 15672 | - |
| pdfme-minio | minio/minio:latest | 9000, 9001 | - |
| pdfme-file-watcher | pdfme-file-watcher | - | rabbitmq, minio |
| pdfme-parser | pdfme-parser-service | 8080 | rabbitmq |
| pdfme-generator | pdfme-pdf-generator | 3000 | rabbitmq |
| pdfme-storage | pdfme-storage-service | - | rabbitmq, minio |

## Volumes

```yaml
rabbitmq_data: RabbitMQ persistence
minio_data: MinIO object storage
```

## Network

```yaml
pdfme-network: bridge
```

## Scaling

```bash
# Scale PDF generators
docker-compose up -d --scale pdf-generator=3

# Scale storage service
docker-compose up -d --scale storage-service=2
```

## Monitoring

```bash
# All logs
docker-compose logs -f

# Specific service
docker-compose logs -f <service-name>

# Queue status
docker exec pdfme-rabbitmq rabbitmqctl list_queues

# Container status
docker-compose ps
```

## Shutdown

```bash
# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Health Checks

```bash
# RabbitMQ
curl http://localhost:15672

# MinIO
curl http://localhost:9000/minio/health/live

# PDF Generator
curl http://localhost:3000/health

# Parser
curl http://localhost:8080/actuator/health
```
