# RabbitMQ Integration

## Overview

RabbitMQ serves as the message broker for the **stateless PDF generation microservices architecture**.

## Queues

### 1. pdf_ready
- **Purpose**: Incoming PDF generation requests
- **Producer**: External applications/services
- **Consumer**: PDF Generator Service (Node.js)
- **Message Format**: See below

### 2. storage_ready
- **Purpose**: Generated PDFs ready for storage
- **Producer**: PDF Generator Service
- **Consumer**: Storage Service (Go)
- **Message Format**: See below

## Message Flow

```
External Producer → [pdf_ready] → PDF Generator → [storage_ready] → Storage Service → MinIO
```

## Message Formats

### pdf_ready Queue

```json
{
  "template_name": "template_form",
  "data": {
    "header_field": "value",
    "trans1_date": "2024-01-01",
    "trans1_description": "Item 1",
    ...
  },
  "pagination": {
    "itemPrefix": "trans",
    "itemsPerPage": 15
  },
  "bucket_name": "pdfs",
  "filename_prefix": "invoice"
}
```

**Required Fields**:
- `template_name`: Template to use (without .json extension)
- `data`: Data to populate in template

**Optional Fields**:
- `pagination`: Pagination settings for repeating rows
- `bucket_name`: MinIO bucket (default: "pdfs")
- `filename_prefix`: Prefix for filename (default: template_name)

### storage_ready Queue

```json
{
  "bucket_name": "pdfs",
  "filename": "invoice_742891.pdf",
  "file_content": "JVBERi0xLjQKJeLjz9MKMyAwIG9iago8PC9UeXBlIC9QYWdlCi..."
}
```

**Fields**:
- `bucket_name`: MinIO bucket name
- `filename`: Filename with random 6-digit suffix
- `file_content`: Base64-encoded PDF content

## Quick Start

### 1. Start All Services

```bash
docker-compose up -d
```

This starts:
- RabbitMQ
- MinIO
- PDF Generator Service
- Storage Service

### 2. Check Services

```bash
docker-compose ps
```

All services should show as "Up (healthy)".

### 3. Access RabbitMQ Management UI

Open http://localhost:15672:
- **Username**: admin
- **Password**: admin123

## Testing

### Send Test Message

Use the test producer script:

```bash
# Install dependencies (first time only)
npm install

# Send test message
node test_producer.js test_request_5trans.json
```

### Monitor Message Flow

1. **RabbitMQ Management UI** (http://localhost:15672):
   - Go to "Queues" tab
   - Watch `pdf_ready` queue (should show message consumed)
   - Watch `storage_ready` queue (should show message consumed)

2. **PDF Generator Logs**:
   ```bash
   docker-compose logs -f pdf-generator
   ```
   Look for:
   - "Received message from pdf_ready"
   - "PDF generated (X bytes)"
   - "Sent to 'storage_ready' queue"

3. **Storage Service Logs**:
   ```bash
   docker-compose logs -f storage-service
   ```
   Look for:
   - "Received message"
   - "Successfully uploaded"

4. **MinIO Console** (http://localhost:9001):
   - Login: minioadmin / minioadmin
   - Check "pdfs" bucket for uploaded files

## Configuration

### RabbitMQ Settings

Edit `rabbitmq.conf` to customize RabbitMQ behavior.

### Queue Properties

Both queues are configured with:
- **Durable**: Yes (survives broker restart)
- **Persistent Messages**: Yes (messages survive restart)
- **Prefetch**: 1 (one message at a time per consumer)

### Connection Settings

Default connection string:
```
amqp://admin:admin123@rabbitmq:5672
```

Change credentials in `docker-compose.yml`:
```yaml
environment:
  RABBITMQ_DEFAULT_USER: your_user
  RABBITMQ_DEFAULT_PASS: your_password
```

## Monitoring

### Queue Status

Via Management UI:
1. Go to http://localhost:15672
2. Click "Queues" tab
3. View metrics for `pdf_ready` and `storage_ready`

Key metrics:
- **Ready**: Messages waiting to be consumed
- **Unacked**: Messages being processed
- **Total**: Total messages in queue
- **Rates**: Messages/second (incoming/outgoing)

### Consumer Status

In Management UI → Queues → Select Queue:
- Check "Consumers" section
- Should show active consumers
- Consumer tag and prefetch count

### Service Health

```bash
# Check all services
docker-compose ps

# View logs
docker-compose logs -f rabbitmq
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
```

## Troubleshooting

### No Consumers on pdf_ready

**Problem**: PDF Generator not consuming messages

**Solutions**:
1. Check PDF Generator is running: `docker-compose ps pdf-generator`
2. Check logs: `docker-compose logs pdf-generator`
3. Verify RabbitMQ connection in logs
4. Restart service: `docker-compose restart pdf-generator`

### No Consumers on storage_ready

**Problem**: Storage Service not consuming messages

**Solutions**:
1. Check Storage Service is running: `docker-compose ps storage-service`
2. Check logs: `docker-compose logs storage-service`
3. Verify MinIO is accessible
4. Restart service: `docker-compose restart storage-service`

### Messages Stuck in Queue

**Problem**: Messages not being processed

**Solutions**:
1. Check for errors in consumer logs
2. Verify message format is correct
3. Check if consumers are connected (Management UI)
4. Purge queue if needed (Management UI → Queue → Purge)

### Connection Errors

**Problem**: Services can't connect to RabbitMQ

**Solutions**:
1. Verify RabbitMQ is running and healthy
2. Check network connectivity: `docker network inspect pdfme-network`
3. Verify credentials match in all services
4. Check RabbitMQ logs for authentication errors

## Development

### Run Locally (Outside Docker)

**RabbitMQ** (keep in Docker):
```bash
docker-compose up -d rabbitmq
```

**PDF Generator** (locally):
```bash
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
npm start
```

**Storage Service** (locally):
```bash
cd storage-service
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
export MINIO_ENDPOINT=localhost:9000
go run cmd/storage-service/main.go
```

### Test Producer

```bash
# Set RabbitMQ URL if not using defaults
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672

# Run test producer
node test_producer.js test_request_5trans.json
```

## Production Considerations

### 1. Security

- **Change Default Credentials**: Update RABBITMQ_DEFAULT_USER/PASS
- **Enable SSL/TLS**: Use amqps:// protocol
- **User Permissions**: Create separate users for each service
- **Virtual Hosts**: Use vhosts to isolate environments

### 2. High Availability

- **Cluster RabbitMQ**: Run multiple RabbitMQ nodes
- **Mirrored Queues**: Enable queue mirroring
- **Load Balancing**: Use HAProxy or similar

### 3. Monitoring & Alerts

- **Prometheus**: Export RabbitMQ metrics
- **Grafana**: Visualize queue metrics
- **Alerting**: Set alerts for queue depth, consumer lag

### 4. Message Retention

- **TTL**: Set message time-to-live
- **Max Length**: Limit queue size
- **Dead Letter Exchange**: Handle failed messages

### 5. Backup

- **Configuration**: Backup RabbitMQ configuration
- **Definitions**: Export queue/exchange definitions
- **Messages**: Consider message durability settings

## Advanced Configuration

### Dead Letter Queue

To handle failed messages, configure dead-letter exchange:

```bash
# Create dead-letter queue
rabbitmqadmin declare queue name=pdf_ready.dlq durable=true

# Configure main queue with DLX
rabbitmqadmin declare queue name=pdf_ready durable=true \
  arguments='{"x-dead-letter-exchange":"dlx"}'
```

### Message TTL

Set message expiration:

```bash
rabbitmqadmin declare queue name=pdf_ready durable=true \
  arguments='{"x-message-ttl":3600000}'  # 1 hour
```

### Queue Length Limit

Limit queue size:

```bash
rabbitmqadmin declare queue name=pdf_ready durable=true \
  arguments='{"x-max-length":10000}'
```

## Useful Commands

```bash
# List queues
docker exec pdfme-rabbitmq rabbitmqctl list_queues

# List consumers
docker exec pdfme-rabbitmq rabbitmqctl list_consumers

# List connections
docker exec pdfme-rabbitmq rabbitmqctl list_connections

# Purge queue
docker exec pdfme-rabbitmq rabbitmqctl purge_queue pdf_ready

# Reset everything
docker-compose down -v
docker-compose up -d
```

## References

- [RabbitMQ Documentation](https://www.rabbitmq.com/documentation.html)
- [Management UI Guide](https://www.rabbitmq.com/management.html)
- [Architecture Documentation](../ARCHITECTURE.md)
