# Message Flow Specification

## Complete Pipeline

```
File Upload (MinIO:uploads)
    ↓
[parse_ready] → Parser → [pdf_ready] → PDF Generator → [storage_ready] → Storage → MinIO:pdfs
```

## Queue Definitions

### parse_ready

**Producer**: File Watcher
**Consumer**: Parser

**Message**:
```json
{
  "filename": "string",
  "file_content": "string (base64)",
  "org_id": "string"
}
```

### pdf_ready

**Producer**: Parser
**Consumer**: PDF Generator

**Message**:
```json
{
  "orgId": "string",
  "cardNumber": "string",
  "statementDate": "string (DD/MM/YYYY)",
  "name": "string",
  "address": "string",
  "availableBalance": number,
  "openingBalance": number,
  "toatalDepits": number,
  "totalCredits": number,
  "currentBalance": number,
  "transactions": [
    {
      "date": "string|null (DD/MM/YYYY)",
      "postDate": "string|null (DD/MM/YYYY)",
      "description": "string",
      "amount": number,
      "currency": "string",
      "amountInBHD": number,
      "cr": boolean
    }
  ]
}
```

### storage_ready

**Producer**: PDF Generator
**Consumer**: Storage Service

**Message**:
```json
{
  "bucket_name": "string",
  "filename": "string",
  "file_content": "string (base64)"
}
```

## Queue Properties

All queues:
- **Durable**: true
- **Persistent messages**: true
- **Prefetch**: 1

## RabbitMQ Connection

```
URL: amqp://admin:admin123@rabbitmq:5672
Management UI: http://localhost:15672
```

## MinIO Buckets

### uploads
- **Purpose**: Input files
- **Created by**: File Watcher
- **Access**: Write (users), Read (File Watcher)

### pdfs
- **Purpose**: Generated PDFs
- **Created by**: Storage Service
- **Access**: Write (Storage Service), Read (users)

**MinIO Console**: http://localhost:9001
**Credentials**: minioadmin / minioadmin
