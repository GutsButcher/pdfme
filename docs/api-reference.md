# API Reference

## PDF Generator HTTP API

**Base URL**: http://localhost:3000

### POST /pdf/parser

Accepts parser output format.

**Request**:
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
  "transactions": [...]
}
```

**Response**:
- **Content-Type**: `application/pdf`
- **Body**: PDF binary
- **Filename**: `{template_name}_{orgId}_{timestamp}.pdf`

**Status Codes**:
- 200: Success
- 400: Missing orgId or transactions
- 500: Generation error

### POST /pdf

Accepts direct template data format.

**Request**:
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
  }
}
```

**Response**:
- **Content-Type**: `application/pdf`
- **Body**: PDF binary
- **Filename**: `{template_name}_document.pdf`

**Status Codes**:
- 200: Success
- 400: Missing template_name or data
- 500: Generation error

### GET /api/templates

List available templates.

**Response**:
```json
["new-template", "template_form_OLD"]
```

### GET /api/templates/:name

Get template definition.

**Response**:
```json
{
  "schemas": [...],
  "basePdf": "...",
  "fonts": {...}
}
```

### POST /api/templates

Save template.

**Request**:
```json
{
  "name": "my-template",
  "template": {...}
}
```

**Response**:
```json
{
  "success": true,
  "name": "my-template"
}
```

### GET /health

Health check.

**Response**:
```json
{
  "status": "ok"
}
```

## Parser HTTP API

**Base URL**: http://localhost:8080

### POST /api/statement/upload

Parse statement file.

**Request**:
```
Content-Type: multipart/form-data

file: <pipe-delimited text file>
```

**Response**:
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
  "transactions": [...]
}
```
