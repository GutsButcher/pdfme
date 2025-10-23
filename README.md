# PDF Generation Microservices

A **fully stateless** microservices architecture for PDF generation and storage using message queues.

## Architecture

```
External Producer → [RabbitMQ: pdf_ready]
                            ↓
                    PDF Generator (Node.js)
                            ↓
                    [RabbitMQ: storage_ready]
                            ↓
                    Storage Service (Go)
                            ↓
                    MinIO Object Storage
```

## Components

| Service | Technology | Purpose |
|---------|-----------|---------|
| **PDF Generator** | Node.js, pdfme | Generates PDFs from templates |
| **Storage Service** | Go, MinIO SDK | Uploads PDFs to object storage |
| **RabbitMQ** | Message Broker | Asynchronous communication |
| **MinIO** | Object Storage | S3-compatible PDF storage |

## Features

✅ **Stateless Design**: No local file storage, fully scalable
✅ **Message-Driven**: Decoupled services via RabbitMQ
✅ **Base64 Transfer**: PDFs transferred as base64 in messages
✅ **Object Storage**: S3-compatible storage with MinIO
✅ **Automatic Pagination**: Smart pagination for repeating rows
✅ **Position-Based Detection**: Auto-detects template structure
✅ **Horizontal Scaling**: Scale any service independently
✅ **Template Designer**: Visual template editor included

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Node.js 18+ (for test producer)

### 1. Start All Services

```bash
docker-compose up -d
```

This starts:
- RabbitMQ (with Management UI)
- MinIO (with Console)
- PDF Generator Service
- Storage Service

### 2. Verify Services

```bash
docker-compose ps
```

All services should show "Up (healthy)".

### 3. Send Test Message

```bash
# Install dependencies (first time only)
npm install

# Send test PDF request
node test_producer.js test_request_5trans.json
```

### 4. Verify Result

**Check MinIO Console** (http://localhost:9001):
- Username: `minioadmin`
- Password: `minioadmin`
- Navigate to "pdfs" bucket
- Find your generated PDF

## Service URLs

| Service | URL | Credentials |
|---------|-----|-------------|
| RabbitMQ Management | http://localhost:15672 | admin / admin123 |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| PDF Generator API | http://localhost:3000 | - |

## How It Works

### 1. Send PDF Request

Send JSON message to RabbitMQ `pdf_ready` queue:

```json
{
  "template_name": "template_form",
  "data": {
    "account_name": "John Doe",
    "trans1_date": "2024-01-01",
    "trans1_description": "Purchase",
    "trans1_amount": "$50.00"
  },
  "pagination": {
    "itemPrefix": "trans",
    "itemsPerPage": 15
  },
  "bucket_name": "pdfs",
  "filename_prefix": "invoice"
}
```

### 2. PDF Generation

PDF Generator service:
- Consumes message from `pdf_ready`
- Loads template from `templates/`
- Generates PDF using pdfme
- Encodes to base64
- Sends to `storage_ready` queue

### 3. Storage Upload

Storage Service:
- Consumes message from `storage_ready`
- Decodes base64 content
- Creates bucket if needed
- Uploads to MinIO
- File stored at: `bucket_name/filename_XXXXXX.pdf`

## Testing

### Available Test Files

Located in `test_data/`:
- `test_request_5trans.json` - 5 transactions
- `test_request_15trans.json` - 15 transactions
- `test_request_25trans.json` - 25 transactions (pagination)

### Send Test Request

```bash
node test_producer.js test_request_5trans.json
```

### Monitor Processing

**Watch Logs**:
```bash
# PDF Generator
docker-compose logs -f pdf-generator

# Storage Service
docker-compose logs -f storage-service

# All services
docker-compose logs -f
```

**Watch Queues** (RabbitMQ UI):
1. Go to http://localhost:15672
2. Click "Queues" tab
3. Watch `pdf_ready` and `storage_ready` queues

## Template Management

### Create Template

1. **Design Template**:
   - Go to https://pdfme.com/template-design
   - Design your template visually
   - Export as JSON

2. **Save Template**:
   ```bash
   # Save to templates/ directory
   cp my_template.json templates/
   ```

3. **Use Template**:
   ```json
   {
     "template_name": "my_template",
     "data": { ... }
   }
   ```

### Pagination Support

For templates with repeating rows (transactions, line items):

1. **Name fields with numbers**: `trans1_date`, `trans2_date`, etc.
2. **Align fields vertically**: Fields at same Y-position = one row
3. **Specify pagination**:
   ```json
   {
     "pagination": {
       "itemPrefix": "trans",
       "itemsPerPage": 15
     }
   }
   ```

The system automatically:
- Detects rows by Y-position
- Splits data across pages
- Maps data to template positions

## Configuration

### Environment Variables

**PDF Generator** (docker-compose.yml):
```yaml
environment:
  - NODE_ENV=production
  - RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672
  - DEFAULT_BUCKET=pdfs
```

**Storage Service** (docker-compose.yml):
```yaml
environment:
  - RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672
  - QUEUE_NAME=storage_ready
  - MINIO_ENDPOINT=minio:9000
  - MINIO_ROOT_USER=minioadmin
  - MINIO_ROOT_PASSWORD=minioadmin
  - MINIO_USE_SSL=false
```

## Scaling

### Horizontal Scaling

Scale any service independently:

```bash
# Scale PDF generators
docker-compose up -d --scale pdf-generator=3

# Scale storage service
docker-compose up -d --scale storage-service=2
```

Each instance consumes from the queue independently.

### Performance Tips

- **RabbitMQ**: Prefetch = 1 (one message per consumer)
- **PDF Generation**: CPU-intensive, scale based on load
- **Storage Upload**: Network I/O bound
- **Base64 Encoding**: Adds ~33% to message size

## Development

### Local Development

**Run services locally**:

```bash
# Start dependencies
docker-compose up -d rabbitmq minio

# Run PDF Generator locally
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
npm start

# Run Storage Service locally
cd storage-service
export RABBITMQ_URL=amqp://admin:admin123@localhost:5672
export MINIO_ENDPOINT=localhost:9000
go run cmd/storage-service/main.go
```

### Project Structure

```
pdfme/
├── src/                    # PDF Generator (Node.js)
│   ├── index.js           # Express API + Consumer
│   └── services/
│       ├── pdfGenerator.js
│       └── rabbitmqConsumer.js
├── storage-service/        # Storage Service (Go)
│   ├── cmd/storage-service/
│   └── pkg/
│       ├── minio/
│       ├── rabbitmq/
│       └── types/
├── templates/              # PDF templates
├── test_data/             # Test requests
├── RabbitMQ/              # RabbitMQ config
├── docker-compose.yml     # All services
└── ARCHITECTURE.md        # Detailed architecture
```

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Detailed architecture and design
- **[RabbitMQ/README.md](RabbitMQ/README.md)** - RabbitMQ configuration and monitoring
- **[storage-service/README.md](storage-service/README.md)** - Storage service documentation
- **[USER_MANUAL.md](USER_MANUAL.md)** - PDF generation API manual

## Monitoring

### RabbitMQ

**Management UI** (http://localhost:15672):
- Queue depth and rates
- Consumer connections
- Message flow

### MinIO

**Console** (http://localhost:9001):
- Storage usage
- Bucket contents
- Object count

### Logs

```bash
# View all logs
docker-compose logs -f

# View specific service
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
```

## Troubleshooting

### Service Not Starting

```bash
# Check status
docker-compose ps

# Check logs
docker-compose logs <service-name>

# Restart service
docker-compose restart <service-name>
```

### Messages Not Processing

1. **Check RabbitMQ**: Are consumers connected?
2. **Check Logs**: Any errors in service logs?
3. **Verify Message Format**: Is JSON valid?
4. **Check Dependencies**: Are RabbitMQ/MinIO healthy?

### Reset Everything

```bash
# Stop and remove everything
docker-compose down -v

# Start fresh
docker-compose up -d
```

## Production Considerations

### Security

1. **Change Default Credentials**:
   - RabbitMQ: admin/admin123
   - MinIO: minioadmin/minioadmin

2. **Enable TLS/SSL**:
   - Use `amqps://` for RabbitMQ
   - Enable MinIO SSL

3. **Network Security**:
   - Internal networks only
   - Firewall rules
   - No public exposure

### High Availability

- **RabbitMQ Cluster**: Multiple RabbitMQ nodes
- **Mirrored Queues**: Queue replication
- **MinIO Cluster**: Distributed MinIO setup
- **Load Balancing**: HAProxy or similar

### Monitoring

- **Prometheus**: Metrics collection
- **Grafana**: Visualization
- **Alerting**: Queue depth, errors, latency

### Backup

- **Templates**: Backup templates/ directory
- **MinIO Data**: S3 sync or MinIO backup
- **RabbitMQ Config**: Export definitions

## API Reference

### HTTP API (Optional)

The PDF Generator also exposes HTTP API:

```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "invoice",
    "data": {"customer": "John"}
  }' \
  --output invoice.pdf
```

Note: Using RabbitMQ is recommended for production (stateless, scalable).

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes
4. Test thoroughly
5. Submit pull request

## License

MIT License

## Support

- **Issues**: GitHub Issues
- **Documentation**: See docs/ directory
- **Architecture**: See ARCHITECTURE.md

## Authors

- PDF Generation Architecture Design
- Microservices Implementation
- MinIO Integration inspired by /home/kaido/afs/pdf-generator
