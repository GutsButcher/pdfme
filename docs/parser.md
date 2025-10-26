# Parser Service

**Technology**: Spring Boot (Java 21)
**Container**: `pdfme-parser`
**Port**: 8080

## Function
Parses pipe-delimited statement files → extracts structured data

## Configuration

```yaml
RABBITMQ_HOST: rabbitmq
RABBITMQ_PORT: 5672
RABBITMQ_USERNAME: admin
RABBITMQ_PASSWORD: admin123
```

## Input

### RabbitMQ Consumer
**Queue**: `parse_ready`

**Message Format**:
```json
{
  "filename": "266_statement.txt",
  "file_content": "base64_encoded_content",
  "org_id": "266"
}
```

### HTTP API (Alternative)
```
POST /api/statement/upload
Content-Type: multipart/form-data

file: <pipe-delimited text file>
```

## Output

**Destination**: RabbitMQ queue `pdf_ready`

**Message Format**:
```json
{
  "orgId": "266",
  "cardNumber": "5117244499894536",
  "statementDate": "21/09/2025",
  "name": "AHMED ADEL HUSAIN ALI",
  "address": "Villa 715 Road AL NASFA",
  "availableBalance": 1026.248,
  "openingBalance": 149.427,
  "toatalDepits": 1159.819,
  "totalCredits": 187.72,
  "currentBalance": 1121.526,
  "transactions": [
    {
      "date": "06/09/2025",
      "postDate": "06/09/2025",
      "description": "Payment Received",
      "amount": 0.0,
      "currency": "",
      "amountInBHD": 149.427,
      "cr": true
    }
  ]
}
```

**Fields**:
- `orgId` (string): Organization ID
- `cardNumber` (string): Card number
- `statementDate` (string): Statement date (DD/MM/YYYY)
- `name` (string): Customer name
- `address` (string): Customer address
- `availableBalance` (number): Available balance
- `openingBalance` (number): Opening balance
- `toatalDepits` (number): Total deposits
- `totalCredits` (number): Total credits
- `currentBalance` (number): Current balance
- `transactions` (array): Transaction list

**Transaction Object**:
- `date` (string|null): Transaction date (DD/MM/YYYY)
- `postDate` (string|null): Post date (DD/MM/YYYY)
- `description` (string): Transaction description
- `amount` (number): Original amount in foreign currency
- `currency` (string): Currency code (GBP, EUR, PLN, etc.)
- `amountInBHD` (number): Amount in BHD
- `cr` (boolean): true=credit, false=debit
