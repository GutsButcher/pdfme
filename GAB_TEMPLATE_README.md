# GAB Credit Card Statement Template

## Overview

This template replicates the Gulf African Bank (GAB) credit card statement format using pdfme JSON templates, replacing the legacy HTML-based PDF generation system.

## Files Created

### 1. Template File
**Location**: `templates/gab_credit_card_statement.json`

A pdfme JSON template that defines the layout and structure of the GAB credit card statement.

### 2. Sample Request
**Location**: `sample_request_gab_statement.json`

Example API request showing how to populate the template with parser data.

### 3. Generated Output
**Location**: `generated_gab_statement.pdf`

Sample PDF output generated from the template.

---

## Template Structure

The template includes the following sections:

### Header Section
- **statement_title**: "CREDIT CARD STATEMENT" (static default)
- Positioned centrally at top of page

### Customer Information
- **customer_name**: Customer's full name
- **customer_address**: Customer's address
- **card_number**: 16-digit card number
- **statement_date**: Statement generation date

### Transaction Table
The template supports **3 transaction rows** per page:
- **trans_date_N**: Transaction date
- **trans_post_date_N**: Posting date
- **trans_desc_N**: Transaction description
- **trans_debit_N**: Debit amount (if any)
- **trans_credit_N**: Credit amount (if any)

Where N = 1, 2, 3

### Summary Section
- **total_debits**: Sum of all debit transactions
- **total_credits**: Sum of all credit transactions
- **payment_instructions**: Static text with payment info (default)

### Account Details
- **credit_limit**: Card credit limit
- **payment_date**: Next payment due date
- **arrears**: Outstanding arrears amount
- **current_balance**: Current account balance
- **minimum_due**: Minimum payment amount

### GAB Points Section
- **starting_point**: Points at statement start
- **points_earned**: Points earned this period
- **points_redeemed**: Points redeemed this period
- **points_balance**: Current points balance

### Footer
- **page_number**: Page numbering (default: "Page 1 of 1")
- **payment_slip_title**: "PAYMENT SLIP" (static default)
- **footer_contact**: Contact information (static default)

---

## Mapping from Parser Output

The parser outputs JSON in this format:

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

### Field Mapping

| Parser Field | Template Field | Notes |
|--------------|----------------|-------|
| `name` | `customer_name` | Direct mapping |
| `address` | `customer_address` | Direct mapping |
| `cardNumber` | `card_number` | Direct mapping |
| `statementDate` | `statement_date` | Direct mapping |
| `toatalDepits` | `total_debits` | Note: typo in parser ("toatal") |
| `totalCredits` | `total_credits` | Direct mapping |
| `currentBalance` | `current_balance` | Direct mapping |
| `transactions[n].date` | `trans_date_N` | Per transaction |
| `transactions[n].postDate` | `trans_post_date_N` | Per transaction |
| `transactions[n].description` | `trans_desc_N` | Per transaction |
| `transactions[n].cr == false` | `trans_debit_N` | If debit, show `amountInBHD` |
| `transactions[n].cr == true` | `trans_credit_N` | If credit, show `amountInBHD` |

### Additional Fields (Not in Parser)

These need to be calculated or provided separately:
- `credit_limit`: Card's credit limit
- `payment_date`: Next payment due date
- `arrears`: Calculated arrears
- `minimum_due`: Calculated minimum payment
- `starting_point`, `points_earned`, `points_redeemed`, `points_balance`: Loyalty points data

---

## Usage Example

### API Request

```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d '{
    "template_name": "gab_credit_card_statement",
    "data": {
      "customer_name": "AHMED ADEL HUSAIN ALI",
      "customer_address": "Villa 715 Road AL NASFA",
      "card_number": "5117244499894536",
      "statement_date": "21/09/2025",
      "trans_date_1": "06/09/2025",
      "trans_post_date_1": "06/09/2025",
      "trans_desc_1": "Payment Received",
      "trans_credit_1": "149.427",
      "total_debits": "1,159.819",
      "total_credits": "187.72",
      "current_balance": "1,121.526",
      ...
    }
  }' \
  --output statement.pdf
```

---

## Limitations

### 1. Fixed Transaction Rows
The template supports only **3 transactions** per page. For statements with more transactions:
- Create multiple pages
- OR extend the template with more transaction fields (trans_date_4, trans_date_5, etc.)

### 2. No Dynamic Tables
pdfme doesn't support dynamic tables. Each transaction row must be explicitly defined in the template.

### 3. Images/Logos
The template uses text for branding. To add the VISA logo and GAB logo:
- Convert logos to base64
- Add as `image` type fields in the template
- Reference in the template schema

### 4. Styling Limitations
- No background colors for sections (blue payment info box in original)
- Limited font support
- No borders/lines for tables

---

## Extending the Template

### Adding More Transaction Rows

To support 10 transactions, add fields in the template:

```json
{
  "trans_date_4": { ... },
  "trans_post_date_4": { ... },
  "trans_desc_4": { ... },
  "trans_debit_4": { ... },
  "trans_credit_4": { ... }
}
```

And increment Y positions accordingly.

### Adding Images

For the VISA and GAB logos:

```json
{
  "visa_logo": {
    "type": "image",
    "position": { "x": 15, "y": 15 },
    "width": 30,
    "height": 18,
    "default": "data:image/png;base64,..."
  },
  "gab_logo": {
    "type": "image",
    "position": { "x": 165, "y": 15 },
    "width": 30,
    "height": 18,
    "default": "data:image/png;base64,..."
  }
}
```

---

## Comparison with Legacy System

### Legacy System (HTML-based)
- **Templater**: Spring Boot with Thymeleaf
- **Format**: HTML templates converted to PDF
- **Pros**: Dynamic tables, complex styling, SVG support
- **Cons**: Heavy dependency stack, requires Java runtime

### New System (pdfme-based)
- **Templater**: Node.js with pdfme
- **Format**: JSON templates
- **Pros**: Lightweight, simple API, fast generation
- **Cons**: Limited styling, fixed fields, no dynamic tables

---

## Testing

Generate a test PDF:

```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d @sample_request_gab_statement.json \
  --output test_statement.pdf
```

Output: `test_statement.pdf` (7.7KB)

---

## Next Steps

1. **Add More Transaction Rows**: Extend template to support 10-15 transactions
2. **Add Logos**: Convert VISA and GAB logos to base64 and add to template
3. **Multi-page Support**: Create logic to split statements across multiple pages
4. **Data Transformation**: Build a service to transform parser JSON to template format
5. **Styling Enhancements**: Add background colors and borders using pdfme features
