# Template Field Mapping

## Parser Output → Template Fields

### Header Fields

| Parser Field | Template Field | Transform |
|-------------|----------------|-----------|
| `orgId` | - | Determines template via mapping |
| `name` | `Cname` | Direct |
| `address` | `Caddress` | Direct |
| `cardNumber` | `CardNumber` | Direct |
| `cardNumber` | `CN1` - `CN16` | Split into individual digits |
| `statementDate` | `StatmentDate` | Direct |
| `availableBalance` | `AvailableBalance` | toString() |
| `openingBalance` | `OpeningBalance` | toString() |
| `currentBalance` | `CurrentBalance` | toString() |
| `toatalDepits` | `TotalDepits` | toString() |
| `totalCredits` | `TotalCredits` | toString() |
| - | `Cpage` | Auto-generated (1, 2, 3...) |
| - | `Mpage` | Auto-generated (total pages) |

### Transaction Fields

| Parser Field | Template Field | Transform |
|-------------|----------------|-----------|
| `transactions[i].date` | `Tr{i+1}Date` | Direct |
| `transactions[i].postDate` | `Tr{i+1}Pdate` | Direct |
| `transactions[i].description` | `Tr{i+1}Details` | Direct |
| `transactions[i].amountInBHD` (if cr=false) | `Tr{i+1}Debits` | toString() |
| `transactions[i].amountInBHD` (if cr=true) | `Tr{i+1}Credits` | toString() |
| `transactions[i].currency` | `Tr{i+1}Currency` | Direct |
| `transactions[i].amount` | `Tr{i+1}Amount` | toString() |

## Card Number Digit Mapping

**Example**: `5117244499894536`

```
CN1  = 5
CN2  = 1
CN3  = 1
CN4  = 7
CN5  = 2
CN6  = 4
CN7  = 4
CN8  = 4
CN9  = 9
CN10 = 9
CN11 = 8
CN12 = 9
CN13 = 4
CN14 = 5
CN15 = 3
CN16 = 6
```

## OrgId to Template Mapping

**Location**: `pdfme/src/config/orgTemplateMapping.js`

```javascript
{
  '266': 'new-template'
}
```

## Pagination Logic

**Input**: Array of transactions
**Output**: Multiple pages

**Parameters**:
- `itemPrefix`: "Tr"
- `itemsPerPage`: 15

**Process**:
1. Filter valid transactions (has date, not header row)
2. Detect template rows by Y-position
3. Calculate pages needed: `ceil(transactions / itemsPerPage)`
4. Map transactions to template positions
5. Add page numbers to each page

**Example**:
- 419 valid transactions
- 15 items per page
- Result: 28 pages
- Page 1: Tr1-Tr15 (Cpage=1, Mpage=28)
- Page 2: Tr16-Tr30 (Cpage=2, Mpage=28)
- ...
- Page 28: Tr406-Tr419 (Cpage=28, Mpage=28)
