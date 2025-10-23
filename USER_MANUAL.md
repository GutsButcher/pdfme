# User Manual

## System Overview

Automated PDF generation pipeline using microservices and message queues.

## Quick Start

```bash
docker-compose up -d
```

**Services**: RabbitMQ, MinIO, File Watcher, Parser, PDF Generator, Storage

## Usage Methods

### Method 1: File Upload (Automated)

1. **Access MinIO Console**: http://localhost:9001
   - Username: `minioadmin`
   - Password: `minioadmin`

2. **Upload file** to `uploads` bucket
   - Naming: `{orgId}_{filename}.txt`
   - Example: `266_statement.txt`

3. **Wait ~10 seconds** for automatic processing

4. **Download PDF** from `pdfs` bucket
   - Filename: `{orgId}_{last4digits}_{random6}.pdf`

### Method 2: Test Script

```bash
./test_upload_file.sh test_data/266003.txt 266
```

### Method 3: HTTP API

**Parser Format**:
```bash
curl -X POST http://localhost:3000/pdf/parser \
  -H "Content-Type: application/json" \
  -d @test_data/parser_test_5trans.json \
  --output statement.pdf
```

**Direct Format**:
```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "new-template",
    "data": {...}
  }' \
  --output document.pdf
```

### Method 4: RabbitMQ

```bash
npm install
node test_producer.js parser_test_5trans.json
```

## File Naming Convention

```
{orgId}_{description}.{ext}

Examples:
  266_statement.txt
  266_account_001.txt
```

OrgId is extracted and used to select template.

## Templates

### Active Templates

- **266**: `new-template.json`

### Adding New Template

1. Design at https://pdfme.com/template-design
2. Export as JSON
3. Save to `./templates/{name}.json`
4. Add mapping in `pdfme/src/config/orgTemplateMapping.js`:
   ```javascript
   '123': 'your-template'
   ```
5. Restart: `docker-compose restart pdf-generator`

## Monitoring

### MinIO Console
http://localhost:9001

**Buckets**:
- `uploads` - Input files
- `pdfs` - Generated PDFs

### RabbitMQ Management
http://localhost:15672 (admin / admin123)

**Queues**:
- `parse_ready` - Files to parse
- `pdf_ready` - Data to generate
- `storage_ready` - PDFs to store

### Service Logs

```bash
docker-compose logs -f file-watcher
docker-compose logs -f parser-service
docker-compose logs -f pdf-generator
docker-compose logs -f storage-service
```

## Service Endpoints

| Service | URL | Credentials |
|---------|-----|-------------|
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |
| RabbitMQ Management | http://localhost:15672 | admin / admin123 |
| PDF Generator API | http://localhost:3000 | - |
| Parser API | http://localhost:8080 | - |

## Troubleshooting

### File Not Processed

Check each stage:
```bash
docker-compose logs file-watcher    # File detected?
docker-compose logs parser-service  # Parsing successful?
docker-compose logs pdf-generator   # PDF generated?
docker-compose logs storage-service # Uploaded to MinIO?
```

### Service Issues

```bash
docker-compose ps                      # Check status
docker-compose restart <service-name>  # Restart service
docker-compose logs <service-name>     # View logs
```

### Reset System

```bash
docker-compose down -v
docker-compose up -d --build
```

## Advanced

### Direct Queue Access

Send messages directly to queues using `test_producer.js`

### Scaling Services

```bash
docker-compose up -d --scale pdf-generator=3
docker-compose up -d --scale storage-service=2
```

### Health Checks

```bash
curl http://localhost:3000/health
```

## Technical Documentation

See `docs/` directory:
- `file-watcher.md` - File watcher specs
- `parser.md` - Parser specs
- `pdfme.md` - PDF generator specs
- `storage-service.md` - Storage specs
- `message-flow.md` - Queue message formats
- `template-mapping.md` - Field mapping rules
- `api-reference.md` - HTTP API reference
- `deployment.md` - Deployment guide
