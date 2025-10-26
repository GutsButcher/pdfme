# Message Flow

## Pipeline

```
S3 uploads → File-Watcher → [parse_ready] → Parser → [pdf_ready] → PDF Generator → [storage_ready] → Storage → S3 pdfs
                ↓                                                                            ↓
           PostgreSQL                                                                   PostgreSQL
              Redis                                                                        Redis
```

## Queue: parse_ready

**Producer**: File-Watcher
**Consumer**: Parser

```json
{
  "job_id": "1a4d1ecf-5319-4bef-abcf-9ec4f212496c",
  "file_hash": "3afbe4f0cec23d2079926ad2c28207cd",
  "filename": "266003.txt",
  "file_content": "base64-encoded-content..."
}
```

**Fields:**
- `job_id`: UUID from database (for tracking)
- `file_hash`: S3 ETag (MD5 hash for deduplication)
- `filename`: Original filename
- `file_content`: Base64-encoded file content

## Queue: pdf_ready

**Producer**: Parser
**Consumer**: PDF Generator

```json
{
  "job_id": "1a4d1ecf-5319-4bef-abcf-9ec4f212496c",
  "file_hash": "3afbe4f0cec23d2079926ad2c28207cd",
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

**Fields:**
- `job_id`, `file_hash`: Passed through from parse_ready
- Parsed statement data
- Transactions array

## Queue: storage_ready

**Producer**: PDF Generator
**Consumer**: Storage

```json
{
  "job_id": "1a4d1ecf-5319-4bef-abcf-9ec4f212496c",
  "file_hash": "3afbe4f0cec23d2079926ad2c28207cd",
  "bucket_name": "pdfs",
  "filename": "statement_266_1761480965145.pdf",
  "file_content": "base64-encoded-pdf..."
}
```

**Fields:**
- `job_id`: For DB update
- `file_hash`: For Redis cache update
- `bucket_name`: Target S3 bucket
- `filename`: Generated PDF filename
- `file_content`: Base64-encoded PDF

## Message Properties

**All queues:**
- Durable: true (survive broker restart)
- Manual ACK: true (reliability)
- Prefetch: 1 (process one at a time)
- Persistent: true (messages survive restart)
