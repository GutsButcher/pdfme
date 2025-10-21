# GAB Credit Card Statement - Simple Template

## What I Created

A simplified but functional pdfme JSON template (`templates/gab_statement.json`) that matches the GAB credit card statement layout.

## Key Features

### ✅ Transaction Table
- **8 transaction rows** (expandable to more if needed)
- Columns: Transaction Date, Posting Date, Transaction Details, Debits, Credits
- Table header with proper labels
- Totals row at the bottom

### ✅ Customer Information
- Customer name
- Customer address
- Card number
- Statement date

### ✅ Account Summary Box (Blue Background)
- Credit Limit
- Payment Date
- Arrears
- Current Balance
- Minimum Due
- Days Past Due

### ✅ GAB Points Section
- Table with headers: Starting Point, Points Earned, Points Redeemed, Points Balance
- Blue header background

### ✅ Additional Elements
- Statement title: "CREDIT CARD STATEMENT"
- Payment instructions
- Payment slip section
- Page number
- Footer with contact info (blue background)

---

## Template Layout

```
┌─────────────────────────────────────────────────┐
│         CREDIT CARD STATEMENT                   │
├─────────────────────────────────────────────────┤
│ CUSTOMER NAME           Card No: XXXX-XXXX-XXXX │
│ Address                 Statement Date: XX/XX/XX│
├─────────────────────────────────────────────────┤
│ Trans Date | Post Date | Details | Debit |Credit│
├─────────────────────────────────────────────────┤
│ XX/XX/XX   | XX/XX/XX  | XXXXX   | XXX   | XXX  │
│ XX/XX/XX   | XX/XX/XX  | XXXXX   | XXX   | XXX  │
│ ...        | ...       | ...     | ...   | ...  │
│                  TOTALS:           XXX     XXX   │
├─────────────────────────────────────────────────┤
│ Payment Instructions    │ Credit Limit:    XXX  │
│                         │ Payment Date:    XXX  │
│                         │ Arrears:         XXX  │
│                         │ Current Balance: XXX  │
│                         │ Minimum Due:     XXX  │
│                         │ Days Past Due:   XXX  │
├─────────────────────────────────────────────────┤
│              GAB POINTS                          │
├─────────────────────────────────────────────────┤
│ Starting | Earned | Redeemed | Balance          │
│   XXX    |  XXX   |   XXX    |   XXX            │
├─────────────────────────────────────────────────┤
│                                    Page 1 of 1   │
├─────────────────────────────────────────────────┤
│              PAYMENT SLIP                        │
│                                                  │
├─────────────────────────────────────────────────┤
│    0729 111 537 | 0711 075 000 | customercare@  │
└─────────────────────────────────────────────────┘
```

---

## How to Use

### 1. Generate PDF

```bash
curl -X POST http://localhost:3000/pdf \
  -H "Content-Type: application/json" \
  -d @sample_request_gab.json \
  --output statement.pdf
```

### 2. Sample Request Structure

See `sample_request_gab.json` for a complete example with 8 transactions.

**Key fields:**
```json
{
  "template_name": "gab_statement",
  "data": {
    "customer_name": "AHMED ADEL HUSAIN ALI",
    "customer_address": "Villa 715 Road AL NASFA",
    "card_number": "5117244499894536",
    "statement_date": "21/09/2025",

    "trans_1_date": "09/09/25",
    "trans_1_post": "09/09/25",
    "trans_1_desc": "Payment Received",
    "trans_1_debit": "",
    "trans_1_credit": "149.427",

    // ... trans_2 through trans_8 ...

    "total_debits": "18,373.55",
    "total_credits": "19,736.34",

    "credit_limit": "10,000.00",
    "payment_date": "22/10/25",
    "arrears": "0.00",
    "current_balance": "38,509.89",
    "minimum_due": "2,878.70",
    "days_past_due": "0",

    "starting_point": "0",
    "points_earned": "0",
    "points_redeemed": "0",
    "points_balance": "0"
  }
}
```

---

## Transaction Fields

For each transaction (N = 1 to 8):
- `trans_N_date`: Transaction date (e.g., "09/09/25")
- `trans_N_post`: Posting date (e.g., "09/09/25")
- `trans_N_desc`: Description (e.g., "Payment Received")
- `trans_N_debit`: Debit amount (leave empty "" if credit)
- `trans_N_credit`: Credit amount (leave empty "" if debit)

**Note**: For header rows (like "TRANSACTIONS OF CARD XXXX1234"), leave date, debit, and credit fields empty.

---

## Adding More Transactions

To support more than 8 transactions:

1. Open `templates/gab_statement.json`
2. Copy the pattern from `trans_8_*` fields
3. Add new fields: `trans_9_date`, `trans_9_post`, etc.
4. Increment the Y position by 6 for each new row:
   - trans_9: y = 101
   - trans_10: y = 107
   - trans_11: y = 113
   - etc.

---

## Visual Styling

### Blue Background Boxes
The following fields have blue backgrounds (`#5B9BD5`) with white text:
- Account summary section (Credit Limit, Payment Date, etc.)
- GAB Points header row
- Footer contact bar (`#004B87` - darker blue)

### Fonts
- **Titles**: NotoSerifJP-Regular, 12-14pt
- **Headers**: 8-10pt
- **Table content**: 7pt
- **Footer**: 7pt

---

## What's Missing (Compared to Original)

1. **VISA Logo**: Not included (can add as base64 image)
2. **GAB Bank Logo**: Not included (can add as base64 image)
3. **Background Image**: Not included (eStatement banner)
4. **Table Borders**: pdfme doesn't support borders easily
5. **Complex Payment Slip Layout**: Simplified version only

---

## Testing Results

✅ **Generated PDF**: `gab_statement_generated.pdf` (9KB)
✅ **HTTP Status**: 200
✅ **Format**: PDF document, version 1.7
✅ **Transaction Rows**: 8 rows tested successfully
✅ **Blue backgrounds**: Working
✅ **All fields**: Rendering correctly

---

## Next Steps

If you want to enhance the template:

1. **Add logos**: Convert images to base64 and add as image fields
2. **More transactions**: Extend to 15-20 rows
3. **Multi-page**: Create logic to split long statements
4. **Better styling**: Use pdfme playground to fine-tune positions
5. **Dynamic data mapper**: Service to convert parser JSON → template format

---

## Files

- **Template**: `templates/gab_statement.json`
- **Sample Request**: `sample_request_gab.json`
- **Generated PDF**: `gab_statement_generated.pdf`
- **This Guide**: `SIMPLE_TEMPLATE_GUIDE.md`
