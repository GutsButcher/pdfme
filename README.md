# PDFme Generator Service

A simple Node.js application that generates PDFs using the pdfme library. The service selects templates based on request parameters and returns generated PDFs directly in HTTP responses.

## Features

- **🎨 Visual Template Designer**: Built-in web-based designer for creating templates (no need for external tools!)
- **REST API**: Simple POST endpoint at `/pdf`
- **Template Selection**: Automatically selects templates based on `template_name` field
- **Direct PDF Response**: Returns PDF directly without storage (like Gotenberg)
- **Docker Support**: Fully containerized with volume-mounted templates
- **Easy Template Management**: Create, edit, and save templates through the web designer or by editing JSON files

## 🚀 Quick Access

- **Template Designer**: http://localhost:3000/designer
- **API Endpoint**: http://localhost:3000/pdf
- **Health Check**: http://localhost:3000/health

## Project Structure

```
pdfme/
├── src/
│   ├── index.js                 # Express server
│   └── services/
│       └── pdfGenerator.js      # PDF generation logic
├── templates/                    # Template storage (mounted as volume)
│   ├── bankX.json
│   └── bankY.json
├── Dockerfile
├── docker-compose.yml
└── package.json
```

## Quick Start

### Using Docker Compose (Recommended)

1. **Build and start the service**:
   ```bash
   docker-compose up -d --build
   ```

2. **Check if service is running**:
   ```bash
   curl http://localhost:3000/health
   ```

3. **Generate a PDF**:
   ```bash
   curl -X POST http://localhost:3000/pdf \
     -H "Content-Type: application/json" \
     -d '{
       "bank_name": "bankX",
       "data": {
         "bank_logo": "Bank X - Premium Banking",
         "customer_name": "John Doe",
         "account_number": "ACC-123456789",
         "balance": "$10,250.00",
         "statement_date": "2024-01-15"
       }
     }' \
     --output statement.pdf
   ```

### Using Node.js Directly

1. **Install dependencies**:
   ```bash
   npm install
   ```

2. **Start the server**:
   ```bash
   npm start
   ```

3. **Test the API** (same curl command as above)

## API Documentation

### POST /pdf

Generates a PDF based on the provided template and data.

**Request Body**:
```json
{
  "bank_name": "bankX",
  "data": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

**Parameters**:
- `bank_name` (required): Name of the template to use (matches filename in templates/)
- `data` (required): Object containing field values to populate in the template

**Response**:
- Content-Type: `application/pdf`
- Returns PDF file directly

**Example**:
```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "bank_name": "bankY",
    "data": {
      "bank_logo": "Bank Y Corporation",
      "customer_name": "Jane Smith",
      "account_number": "987654321",
      "balance": "$25,500.00",
      "statement_date": "2024-01-20"
    }
  }' \
  --output output.pdf
```

### GET /health

Health check endpoint.

**Response**:
```json
{
  "status": "ok"
}
```

## Adding New Templates

1. Create a new JSON template file in the `templates/` directory
2. Name it according to the `bank_name` you'll use in requests (e.g., `bankZ.json`)
3. Follow the pdfme template schema structure

**Example template** (`templates/bankZ.json`):
```json
{
  "basePdf": {
    "width": 210,
    "height": 297,
    "padding": [10, 10, 10, 10]
  },
  "schemas": [
    {
      "customer_name": {
        "type": "text",
        "position": { "x": 20, "y": 30 },
        "width": 170,
        "height": 10,
        "fontSize": 16,
        "fontColor": "#000000",
        "alignment": "left"
      }
    }
  ]
}
```

No restart needed! The app reads templates on each request.

## Template Schema

pdfme templates consist of:

- **basePdf**: Page dimensions and padding
- **schemas**: Array of field definitions with:
  - `type`: Field type (text, image, etc.)
  - `position`: X, Y coordinates
  - `width`, `height`: Field dimensions
  - `fontSize`, `fontColor`: Styling
  - `alignment`: Text alignment

## Docker Configuration

The `templates/` directory is mounted as a read-only volume:
```yaml
volumes:
  - ./templates:/app/templates:ro
```

This allows you to add/modify templates without rebuilding the container.

## Development

Run with auto-reload:
```bash
npm run dev
```

## Stopping the Service

```bash
docker-compose down
```

## Troubleshooting

**Template not found error**:
- Ensure the template file exists in `templates/` directory
- Verify the filename matches `bank_name` exactly (case-sensitive)
- Check the template JSON is valid

**PDF generation fails**:
- Verify all fields in `data` object match the template schema
- Check template schema syntax

**Container issues**:
```bash
# View logs
docker-compose logs -f

# Restart service
docker-compose restart
```

## License

MIT
# pdfme
