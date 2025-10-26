# PDF Generator Service

**Technology**: Node.js
**Container**: `pdfme-generator`
**Port**: 3000

## Function
Generates PDFs from templates → sends to storage

## Configuration

```yaml
NODE_ENV: production
RABBITMQ_URL: amqp://admin:admin123@rabbitmq:5672
DEFAULT_BUCKET: pdfs
```

## Input

### RabbitMQ Consumer
**Queue**: `pdf_ready`

**Format 1: Parser Output** (auto-detected)
```json
{
  "orgId": "266",
  "cardNumber": "5117244499894536",
  "statementDate": "21/09/2025",
  "name": "AHMED ADEL HUSAIN ALI",
  "address": "Villa 715 Road AL NASFA",
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

**Format 2: Direct Template Data**
```json
{
  "template_name": "new-template",
  "data": {
    "Cname": "AHMED ADEL",
    "Tr1Date": "06/09/2025",
    "Tr1Debits": "149.427"
  },
  "pagination": {
    "itemPrefix": "Tr",
    "itemsPerPage": 15
  },
  "bucket_name": "pdfs",
  "filename_prefix": "statement"
}
```

### HTTP API (Alternative)

**Endpoint 1: Parser Format**
```
POST /pdf/parser
Content-Type: application/json

{parser output format}
```

**Endpoint 2: Direct Format**
```
POST /pdf
Content-Type: application/json

{direct format}
```

## Processing Logic

### OrgId → Template Mapping
Located: `src/config/orgTemplateMapping.js`

```javascript
'266': 'new-template'
```

### Parser Data Transformation
Located: `src/services/parserDataTransformer.js`

**Field Mappings**:
- `name` → `Cname`
- `address` → `Caddress`
- `cardNumber` → `CardNumber` + split to `CN1`-`CN16` (individual digits)
- `statementDate` → `StatmentDate`
- `transactions[i]` → `Tr{i+1}Date`, `Tr{i+1}Pdate`, `Tr{i+1}Details`
- `transactions[i].amountInBHD` → `Tr{i+1}Debits` (if cr=false) or `Tr{i+1}Credits` (if cr=true)
- Auto-generates: `Cpage` (current page number), `Mpage` (total pages)

**Transaction Filtering**:
- Filters out: `tx.date == null`
- Filters out: `description.includes('Account transactions')`

**Pagination**:
- `itemPrefix`: "Tr"
- `itemsPerPage`: 15
- Auto-detects rows by Y-position
- Generates multiple pages as needed

## Output

**Destination**: RabbitMQ queue `storage_ready`

**Message Format**:
```json
{
  "bucket_name": "pdfs",
  "filename": "266_4536_542462.pdf",
  "file_content": "base64_encoded_pdf"
}
```

**Fields**:
- `bucket_name` (string): MinIO bucket name
- `filename` (string): Generated filename (format: `{orgId}_{last4digits}_{random6}.pdf`)
- `file_content` (string): Base64-encoded PDF
