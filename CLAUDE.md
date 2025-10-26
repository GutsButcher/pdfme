# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A **microservices architecture** for automated PDF generation and storage using message queues. The system processes bank statements through a multi-stage pipeline: file upload → parsing → PDF generation → storage.

### Core Architecture

```
MinIO (uploads) → File Watcher → [parse_ready] → Parser → [pdf_ready] → PDF Generator → [storage_ready] → Storage Service → MinIO (pdfs)
```

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
- **Purpose:** Monitors MinIO `uploads` bucket, sends files to parser
- **Queue:** Produces to `parse_ready`
- **Key Feature:** Polls every 10s, extracts orgID from filename pattern `{orgId}_statement.{ext}`

### 2. Parser Service (Java/Spring Boot)
- **Location:** `parser/`
- **Purpose:** Parses bank statement files (text format) into structured JSON
- **Queues:** Consumes `parse_ready`, produces to `pdf_ready`
- **Key Files:**
  - `src/main/java/com/afs/parser/service/RabbitMQConsumer.java` - Queue consumer
  - `src/main/java/com/afs/parser/service/StatementParser.java` - Parsing logic
  - `src/main/java/com/afs/parser/Controllers/EStatementController.java` - HTTP API (optional)
- **Output:** EStatementRecord with transactions array

### 3. PDF Generator (Node.js)
- **Location:** `pdfme/`
- **Purpose:** Generates multi-page PDFs from templates and data
- **Queues:** Consumes `pdf_ready`, produces to `storage_ready`
- **Key Files:**
  - `src/services/pdfGenerator.js` - Core PDF generation with pagination
  - `src/services/parserDataTransformer.js` - Transforms parser output to template format
  - `src/services/rabbitmqConsumer.js` - Queue consumer
- **Templates:** Located in `templates/` directory (JSON format from pdfme.com)
- **Key Feature:** Automatic pagination - detects rows by Y-position, splits across pages

### 4. Storage Service (Go)
- **Location:** `storage-service/`
- **Purpose:** Uploads generated PDFs to MinIO object storage
- **Queue:** Consumes `storage_ready`
- **Key Feature:** Decodes base64 PDFs, creates buckets automatically

## Message Queue Flow

### Queue: `parse_ready`
Produced by: File Watcher → Consumed by: Parser
```json
{
  "filename": "266_statement.txt",
  "file_content": "base64_encoded_content",
  "org_id": "266"
}
```

### Queue: `pdf_ready`
Produced by: Parser → Consumed by: PDF Generator
```json
{
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
  "bucket_name": "pdfs",
  "filename_prefix": "statement_266",
  "file_content": "base64_encoded_pdf_content"
}
```

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
| PDF Generator API | http://localhost:3000 | - |
| Parser API | http://localhost:8080 | - |

## Key Design Principles

1. **Fully Stateless**: No local file storage, all data transfers via base64 in messages
2. **Horizontally Scalable**: Each service can be scaled independently
3. **Event-Driven**: All communication via RabbitMQ queues
4. **Automatic Pagination**: Position-based row detection in PDF templates
5. **Graceful Degradation**: Services auto-reconnect to RabbitMQ/MinIO on failure

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
3. Verify message format matches queue schema
4. Ensure RabbitMQ and MinIO are healthy

### Reset Everything
```bash
docker-compose down -v  # Remove volumes
docker-compose up -d    # Fresh start
```

## Additional Documentation

- `README.md` - Comprehensive getting started guide
- `docs/message-flow.md` - Detailed message queue flow
- `docs/template-mapping.md` - Template field mapping details
- `docs/parser.md` - Parser service documentation
- `docs/pdfme.md` - PDF generator documentation
- `parser/CHANGES.md` - Parser RabbitMQ integration changes
