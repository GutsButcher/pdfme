# 🎨 PDFme Template Designer - User Guide

## Overview

Your application now includes a **self-hosted visual template designer**! No need to use the pdfme playground website - you have your own designer running in your app.

---

## Access the Designer

Open your browser and go to:

```
http://localhost:3000/designer
```

---

## Features

### 🎨 Visual Design
- Drag and drop fields onto the canvas
- Resize and position elements visually
- Live preview of your template
- Support for text, images, QR codes, and barcodes

### 💾 Save Templates
- Save templates directly to your `./templates/` directory
- Load existing templates for editing
- Download templates as JSON files
- Templates are immediately available for PDF generation

### 📋 Template Management
- List all available templates
- Load and edit existing templates
- Create new templates from scratch
- Delete old templates (manual file deletion)

---

## How to Use

### Creating a New Template

1. **Open the designer**: http://localhost:3000/designer
2. **Start designing**:
   - Click the "+" button to add fields
   - Choose field type (Text, Image, QR Code, etc.)
   - Position and resize fields on the canvas
   - Configure field properties (font, color, alignment, etc.)
3. **Name your template**: Enter a name in the "Template Name" field (e.g., `invoice`)
4. **Save**: Click "💾 Save Template"
5. **Done!** Your template is saved to `./templates/invoice.json`

### Editing an Existing Template

1. **Open the designer**: http://localhost:3000/designer
2. **Load template**:
   - Select a template from the dropdown
   - Click "Load"
3. **Edit**: Make your changes
4. **Save**: Click "💾 Save Template"

### Downloading Templates

If you want to backup a template or use it elsewhere:
1. Click "⬇️ Download JSON"
2. The template JSON file will be downloaded to your computer

---

## Designer Interface

### Top Bar Controls

| Control | Description |
|---------|-------------|
| **Load Template** dropdown | Select from existing templates |
| **Load** button | Load the selected template |
| **Template Name** input | Name for saving the template |
| **💾 Save Template** | Save to `./templates/` directory |
| **⬇️ Download JSON** | Download template as JSON file |
| **🔄 Reset** | Clear the designer and start fresh |

### Canvas Area

- **Drag and drop** to position fields
- **Resize handles** to adjust field sizes
- **Click fields** to edit properties
- **Use toolbar** to add new fields

---

## Field Types

### Text Field
- Display static or dynamic text
- Configure: font, size, color, alignment
- Use for: labels, customer names, dates, amounts

### Image Field
- Display logos, photos, signatures
- Upload images or use base64 data
- Use for: company logos, signatures, product images

### QR Code
- Generate QR codes from data
- Configure: size, error correction
- Use for: payment codes, URLs, tracking numbers

### Barcode
- Generate various barcode types
- Configure: format, size
- Use for: product codes, shipping labels

---

## Best Practices

### 1. Use Descriptive Field Names
```
✅ Good: customer_name, invoice_date, total_amount
❌ Bad: field1, text_2, x
```

### 2. Set Default Values for Static Content
For fields that never change (like labels, headers), set a default value:
- Field property: `default` → "Invoice Total:"
- This text will always appear, even if not in request data

### 3. Organize Fields Logically
- Group related fields together
- Use consistent spacing
- Align fields properly for professional look

### 4. Test Your Template
After saving, test it immediately:
```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "your_template",
    "data": {
      "customer_name": "Test User",
      "amount": "100.00"
    }
  }' --output test.pdf
```

---

## API Endpoints

The designer uses these API endpoints (already implemented):

### List Templates
```
GET /api/templates
Response: ["template1", "template2", ...]
```

### Load Template
```
GET /api/templates/:name
Response: { template JSON }
```

### Save Template
```
POST /api/templates
Body: { "name": "template_name", "template": {...} }
Response: { "success": true, "name": "template_name" }
```

---

## Workflow Example

### Scenario: Create Invoice Template

1. **Open Designer**: http://localhost:3000/designer

2. **Add Header**:
   - Add text field: "INVOICE"
   - Font: 20pt, Bold
   - Position: Top center
   - Set as default (static text)

3. **Add Customer Info**:
   - Add text field: `customer_name`
   - Font: 12pt
   - Position: Left side

4. **Add Invoice Details**:
   - Add text fields: `invoice_number`, `invoice_date`, `due_date`
   - Position: Right side

5. **Add Items Table**:
   - Add text fields for headers: "Description", "Quantity", "Price"
   - Add fields for data: `item_1_desc`, `item_1_qty`, `item_1_price`
   - Repeat for multiple rows

6. **Add Total**:
   - Add text field: `total_amount`
   - Font: 14pt, Bold
   - Position: Bottom right

7. **Save**:
   - Template name: `invoice`
   - Click "💾 Save Template"

8. **Test**:
```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "invoice",
    "data": {
      "customer_name": "John Doe",
      "invoice_number": "INV-001",
      "invoice_date": "2024-01-15",
      "due_date": "2024-02-15",
      "item_1_desc": "Consulting Services",
      "item_1_qty": "10 hours",
      "item_1_price": "$1,000.00",
      "total_amount": "$1,000.00"
    }
  }' --output invoice.pdf
```

---

## Troubleshooting

### Designer not loading
- Check container is running: `docker ps`
- Check logs: `docker-compose logs pdf-generator`
- Restart: `docker-compose restart`

### Template not saving
- Check templates directory is mounted correctly
- Verify template name doesn't have special characters
- Check console for errors (F12 in browser)

### Fields not showing in PDF
- Field names in request must match template exactly
- Check field names are lowercase/matching case
- Verify data is in correct format

---

## Tips & Tricks

### 1. Use Grid/Guides
- Enable grid in designer settings
- Helps align fields perfectly

### 2. Copy Templates
- Load existing template
- Modify it
- Save with new name

### 3. Version Control
- Save templates with version numbers: `invoice_v1`, `invoice_v2`
- Keep old versions as backups

### 4. Reusable Components
- Create templates with common layouts
- Clone and modify for new use cases

---

## Next Steps

1. ✅ Open http://localhost:3000/designer
2. ✅ Design your first template
3. ✅ Save it
4. ✅ Generate a PDF with it
5. ✅ Iterate and improve!

Happy designing! 🎨
