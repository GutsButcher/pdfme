# PDF Generator Service - User Manual

## Quick Start

### 1. Start the Service

```bash
docker-compose up -d
```

The service will be available at `http://localhost:3000`

---

## Creating Templates

### Step 1: Design Your Template

1. Go to **pdfme Playground**: https://pdfme.com/template-design
2. Design your template using the visual editor:
   - Add text fields, images, shapes, etc.
   - Position and style elements as needed
   - Define field names for dynamic data (e.g., `customer_name`, `invoice_number`)
3. **Export the template** as JSON

### Step 2: Save Template File

1. Save the exported JSON in `./templates/` directory
2. Name it descriptively (e.g., `invoice.json`, `bank_statement.json`, `contract.json`)
3. The filename (without `.json`) will be your `template_name` in API requests

**Example:**
```bash
./templates/invoice.json          → template_name: "invoice"
./templates/bank_statement.json   → template_name: "bank_statement"
```

---

## Using the API

### Endpoint

**POST** `http://localhost:3000/pdf`

### Request Format

```json
{
  "template_name": "your_template_name",
  "data": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

- **`template_name`**: Name of the template file (without `.json` extension)
- **`data`**: Object with field names matching those in your template

### Example Request

```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "invoice",
    "data": {
      "customer_name": "John Smith",
      "invoice_number": "INV-2024-001",
      "total_amount": "$1,500.00"
    }
  }' \
  --output invoice.pdf
```

### Response

- **Success (200)**: PDF file returned directly
- **Error (400)**: Missing `template_name` or `data`
- **Error (500)**: Template not found or generation failed

---

## Important Notes

### Field Names Must Match

The field names in your request `data` **must exactly match** the field names defined in your pdfme template.

**Template defines:**
```json
"customer_name": { ... }
"invoice_total": { ... }
```

**Request must use:**
```json
"data": {
  "customer_name": "John Doe",
  "invoice_total": "$500"
}
```

### Static vs Dynamic Content

Use pdfme's `default` property for static content (logos, labels, headers):

```json
{
  "company_logo": {
    "type": "text",
    "default": "ABC Corporation",
    ...
  }
}
```

Static fields don't need to be in request data.

### Multiple Pages

To generate multi-page PDFs, send `data` as an array:

```json
{
  "template_name": "invoice",
  "data": [
    { "customer_name": "Customer 1", ... },
    { "customer_name": "Customer 2", ... }
  ]
}
```

Each object creates one page.

### No Container Restart Needed

When you add/modify templates in `./templates/`, they're immediately available. No need to restart the container.

---

## Health Check

```bash
curl http://localhost:3000/health
```
Returns: `{"status":"ok"}`

---

## Example Workflow

```bash
# 1. Create template in pdfme playground
# 2. Export and save as ./templates/receipt.json
# 3. Generate PDF

curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "receipt",
    "data": {
      "date": "2024-01-15",
      "item": "Coffee",
      "price": "$4.50"
    }
  }' \
  --output receipt.pdf

# Done! receipt.pdf is generated
```
