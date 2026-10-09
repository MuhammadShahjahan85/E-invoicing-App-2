# ERP / POS Integration API

ERP systems, accounting packages and POS terminals can send invoices to Veridian E-invoicing Pakistan over a REST/JSON API. The product:

- applies the tax engine and FBR validation;
- reports each invoice to FBR in real time;
- queues it if FBR is unreachable;
- returns the FBR invoice number and QR code for your own printouts.

Your ERP never handles FBR tokens.

---

## 1. Basics

| Item | Value |
|---|---|
| Base URL | `https://<server>:8443/api/v1/companies/{cid}`. `{cid}` is the company id shown on the ERP API keys page (the number in the Base URL). |
| Format | JSON, UTF-8. Amounts are numbers in rupees (2 decimals); quantities allow up to 4 decimals. |
| TLS | HTTPS with the server's certificate. Trust `<data>/tls/cert.pem` on the ERP machine, or install a company certificate. Do not disable certificate checks in production code. |
| Authentication | Header `X-API-Key: eik_…`, or `Authorization: Bearer eik_…` |
| Offline testing | Point the product at the training simulator (environment "Training simulator"), or run `einvoice mock-fbr` to test against an FBR-like API. |

### API keys

An administrator creates keys under **Settings → ERP API keys**. The key is shown once; only a hash is stored. Each key belongs to **one company** and can only access that company's `{cid}`.

| Allowed | Not allowed |
|---|---|
| Read masters and invoices; create and update customers and products; create, update, validate and submit invoices; create stock transfer notes and record their receipt; read reports and closings | Cancellations (of invoices and stock transfer notes), debit notes and reconciliation; company settings, return filing extensions and FBR tokens; users and system settings — these stay with authorised users in the web UI |

Revoking a key takes effect immediately.

## 2. Endpoints

All paths are relative to `/api/v1/companies/{cid}`.

| Method | Path | Purpose |
|---|---|---|
| POST | `/invoices` | Create an invoice (optionally submit to FBR at once) |
| POST | `/invoices/compute` | Calculate taxes and validation issues without saving |
| GET | `/invoices` | List invoices. Query: `status` (comma-separated), `docType`, `from`, `to` (YYYY-MM-DD), `q`, `env`, `limit` (default 50), `offset` |
| GET | `/invoices/{id}` | Invoice detail: `{ "invoice": {...}, "sealValid": true, "signatureValid": true?, "signingKey": "XXXX-XXXX-XXXX-XXXX"?, "original": {...}?, "debitNotes": {...}? }`. `invoice.signature` is the Base64 Ed25519 digital signature recorded on acceptance (rule 150R(4)(b)) |
| GET | `/invoice-by-ref/{externalRef}` | Find an invoice by your ERP reference. Optional `?env=` |
| PUT | `/invoices/{id}` | Replace the content of a Draft, Validated or Rejected invoice |
| DELETE | `/invoices/{id}` | Delete a draft that was never reported |
| POST | `/invoices/{id}/validate` | Validate with FBR (nothing is recorded by FBR) |
| POST | `/invoices/{id}/submit` | Submit a saved invoice to FBR |
| POST | `/invoices/{id}/retry` | Retry a queued invoice now |
| GET | `/invoices/{id}/payload` | The exact JSON sent (or to be sent) to FBR |
| GET | `/invoices/{id}/print` | Printable HTML. Query: `format=a4\|thermal`, `autoprint=1`, `preview=1` (does not count as a print), `toolbar=0` |
| GET | `/invoices/{id}/qr.png` | QR code PNG (`?scale=12`, pixels per module, maximum 40), encoding the FBR invoice number |
| GET | `/invoices/{id}/qr.svg` | QR code SVG, 1 × 1 inch, version 2 |
| GET / POST | `/customers` | List (`q`, `limit`, `offset`, `active=1`) / create customer |
| GET / PUT | `/customers/{id}` | Read / update customer |
| POST | `/buyer-check` | `{ "regNo": "1234567" }` → FBR Active Taxpayer List and registration type |
| GET / POST | `/products` | List / create product |
| GET / PUT | `/products/{id}` | Read / update product |
| GET | `/ref/provinces`, `/ref/sale-types`, `/ref/uoms` | Reference lists |
| GET | `/ref/hs-codes?q=` | HS code search |
| GET | `/ref/rates?saleType=&date=` | Rates allowed by FBR for a sale type on a date |
| GET | `/ref/sro-schedules?rateId=&date=`, `/ref/sro-items?sroId=&date=` | SRO references |
| GET | `/ref/hs-uom?hsCode=` | UoM expected by FBR for an HS code |
| GET | `/reports/{register\|tax-summary\|monthly\|customers\|annex-c}` | Reports. Query: `from`, `to`, `env`, `format=csv\|xlsx`. `annex-c` lists every document with an FBR number in the period, including cancelled ones, for matching with Annex-C (rule 150XD(2)) |
| GET | `/closings` | Day, week and month closings (rule 150R(4)(f)). Query: `kind=day\|week\|month` (all when omitted), `limit`, `env`. Response: `{ "closings": [...], "checked": 12, "chainOk": true }` |
| GET | `/return-extensions` | FBR filing-date extensions recorded for the company |
| GET / POST | `/stock-transfers` | List stock transfer notes (`from`, `to`, `status=DISPATCHED\|RECEIVED\|CANCELLED`, `q`, `limit`, `offset`) / create a note (§3.4) |
| GET | `/stock-transfers/{id}` | Note with its lines |
| POST | `/stock-transfers/{id}/receive` | `{ "receivedBy": "Name, designation", "receivedAt": "2026-10-09T16:30" }` (time optional, Pakistan time) |
| GET | `/stock-transfers/{id}/print` | Printable note (despatch and warehouse copies) |

## 3. Creating an invoice

`POST /invoices`

```json
{
  "externalRef": "ERP-1001",
  "docType": "Sale Invoice",
  "invoiceDate": "2026-10-07",
  "buyer": {
    "ntnCnic": "2046004",
    "name": "ABC Traders",
    "province": "PUNJAB",
    "address": "Lahore",
    "registrationType": "Registered"
  },
  "items": [
    {
      "hsCode": "0101.2100",
      "description": "Product A",
      "uom": "Numbers, pieces, units",
      "quantity": 10,
      "unitPrice": 150,
      "saleType": "Goods at standard rate (default)",
      "rate": "18%"
    }
  ],
  "submit": true
}
```

### 3.1 Invoice fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `externalRef` | string | Strongly recommended | Your ERP invoice number. **Idempotency key**: sending the same value again returns the existing invoice instead of creating a duplicate. |
| `docType` | string | No | `"Sale Invoice"` (default) or `"Debit Note"`. A debit note records a return or reduction against an accepted invoice: positive amounts, capped at the original. |
| `invoiceDate` | string | No | `YYYY-MM-DD`; defaults to today (Pakistan time) |
| `customerId` | number | One of `customerId` / `buyer` | Use a customer from the master |
| `buyer` | object | One of `customerId` / `buyer` | Ad-hoc buyer: `ntnCnic`, `name`, `province`, `address`, `registrationType` (`Registered` / `Unregistered`) |
| `withholdingMode` | string | No | `""` (none), `"fraction"` (default 1/5 of sales tax) or `"full"`; defaults from the customer |
| `invoiceRefNo` | string | Debit notes | FBR invoice number of the original invoice |
| `refInvoiceId` | number | No | Internal id of the original invoice (debit notes) |
| `scenarioId` | string | Sandbox only | `SN001`–`SN028`; ignored outside the sandbox |
| `notes` | string | No | Printed on the invoice |
| `advanceReceipt` | boolean | No | `true` for an **advance receipt invoice**: payment received before the supply (s.23(1) and s.2(44)). Reported like a sale invoice and printed as "ADVANCE RECEIPT INVOICE". Sale invoices only. |
| `advanceRef` | string | No | On a final invoice: the FBR numbers of the advance receipt invoices it adjusts (printed) |
| `submit` | boolean | No | `true` = report to FBR immediately after saving |
| `environment` | string | No | Normally omitted. Invoices always use the company's working environment (Settings → FBR integration). If sent, it must equal that environment, otherwise 422. |
| `items` | array | Yes | At least one line |

### 3.2 Line fields

Optional override fields replace the tax engine's calculation. Use them when your ERP has already computed the amounts.

| Field | Type | Notes |
|---|---|---|
| `productId` / `productCode` | number / string | Take HS code, UoM, sale type, rate and SRO from the product master |
| `hsCode` | string | PCT code `NNNN.NNNN` |
| `description` | string | Product description |
| `uom` | string | FBR UoM text, e.g. `"Numbers, pieces, units"`, `"KG"` |
| `quantity` | number | |
| `unitPrice` | number | Price excluding sales tax |
| `discountPercent` / `discountAmount` | number | Line discount |
| `value` | number | *Override*: value excluding sales tax **and excluding FED**. Any FED (`fedRate`/`fed`) is added to form the value of supply (s.2(46)), which is reported as `valueSalesExcludingST` |
| `saleType` | string | FBR sale type text, e.g. `"Goods at standard rate (default)"`, `"Goods at Reduced Rate"`, `"3rd Schedule Goods"`, `"Exempt goods"` (see `/ref/sale-types`) |
| `rate` | string | e.g. `"18%"`, `"5%"`, `"Exempt"`, `"Rs.200"`, `"18% along with rupees 60 per kilogram"`. Defaults from the sale type. |
| `retailPrice` / `retailValue` | number | Third Schedule: retail price printed on the pack per unit, **including** sales tax (the system derives the retail value excluding sales tax) / *override*: the line's retail value **excluding** sales tax, exactly as FBR's `fixedNotifiedValueOrRetailPrice` |
| `furtherTaxMode` | string | `"auto"` (unregistered buyers), `"yes"` or `"no"` |
| `furtherTax`, `salesTax`, `extraTax`, `fed`, `stWithheld` | number | *Overrides* |
| `extraTaxRate`, `fedRate` | number | Percentages |
| `sroScheduleNo`, `sroItemSerialNo` | string | Required for SRO-based sale types (reduced rate, exempt, etc.) |
| `fedType`, `fedRateText`, `fedUnitPrice`, `fedSro`, `fedSroSerial` | string / number | Federal excise duty particulars of rule 150R(13)(aa)–(ff) (SRO 1666(I)/2026): FED type (e.g. `"Ad valorem"`, `"Specific (per unit)"`), the rate as printed when it is not a percentage (e.g. `"Rs 4 per kg"`), the price per unit for FED, and the FED Schedule/SRO reference and serial number. Printed on the invoice, not sent to FBR (the DI API has no fields for them). Blanks are taken from the product |

### 3.3 Sending an FBR-format payload unchanged

If your ERP already produces FBR's DI JSON, send it in `fbrPayload`. The amounts you supply are kept as overrides:

```json
{ "externalRef": "ERP-1001", "submit": true, "fbrPayload": { "invoiceType": "Sale Invoice", "invoiceDate": "2026-10-07", "...": "..." } }
```

The seller fields are always taken from the company profile.

### 3.4 Stock transfer notes

Goods moved to the company's own warehouse under the **same STRN** are not a supply and are not reported to FBR; Sales Tax General Order 25 of 2026 requires a stock transfer note instead.

```http
POST /api/v1/companies/1/stock-transfers
X-API-Key: eik_…
Content-Type: application/json

{
  "dispatchedAt": "2026-10-09T10:15",
  "toName": "Own warehouse, Port Qasim",
  "toAddress": "Plot 7, Port Qasim, Karachi",
  "vehicleNo": "JX-1234",
  "driverCnic": "4210112345671",
  "authorisedBy": "Store manager",
  "sameStrn": true,
  "items": [
    { "productId": 12, "quantity": 500, "valueAtCost": 100000 },
    { "description": "Empty bags", "quantity": 1000, "uom": "Numbers, pieces, units", "valueAtCost": 2500 }
  ]
}
```

`sameStrn` must be `true` (otherwise 422: issue a sales tax invoice). `fromName` / `fromAddress` default to the company. Lines take `productId` (or describe the goods with `description`, `hsCode`, `uom`); `quantity` must be positive. The response is the note with its number (`STN-000001`, …) and status `DISPATCHED`.

## 4. Responses

`201 Created` returns the invoice:

```json
{
  "id": 42,
  "internalNo": "INV-2627-000042",
  "externalRef": "ERP-1001",
  "status": "ACCEPTED",
  "fbrInvoiceNumber": "1234567DI1791353838222",
  "fbrDated": "2026-10-07 11:17:18",
  "totals": { "valueExclST": 1500, "salesTax": 270, "furtherTax": 0, "extraTax": 0, "fed": 0,
              "stWithheld": 0, "totalValue": 1770, "amountPayable": 1770, "gross": 1500, "discount": 0, "retailValue": 0 },
  "fbrErrors": null,
  "lastError": "",
  "items": [ { "lineNo": 1, "hsCode": "0101.2100", "salesTax": 270, "fbrItemInvoiceNo": "1234567DI1791353838222-1", "...": "..." } ],
  "...": "..."
}
```

**Always check `status`:**

| Status | Meaning | ERP action |
|---|---|---|
| `DRAFT` | Saved, not submitted | Submit later (`/submit`) |
| `VALIDATED` | FBR validation passed, not yet reported | Submit |
| `QUEUED` | FBR unreachable; the product retries automatically (`nextAttemptAt`) | Poll `/invoice-by-ref/{externalRef}` until `ACCEPTED` |
| `SUBMITTING` | In progress | Poll again shortly |
| `ACCEPTED` | Reported; `fbrInvoiceNumber` is final | Print with the FBR number and QR code |
| `REJECTED` | FBR rejected; see `fbrErrors` (line, code, message) | Correct and `PUT` the invoice, then submit again |
| `UNCERTAIN` | Outcome unknown (e.g. timeout after sending). Shown as "Needs reconciliation". | An authorised user reconciles it with IRIS in the web UI. **Do not** create a new invoice. |
| `CANCELLED` | Cancelled after FBR cancellation | Reverse in the ERP |

### Errors

| HTTP | Body | When |
|---|---|---|
| 422 | `{ "error": "...", "issues": [ { "line": 1, "field": "hsCode", "code": "0052", "severity": "error", "message": "..." } ] }` | Validation failed. If `submit` was true and `externalRef` was given, the response also contains `"invoice"`: it was saved as a draft. |
| 422 | `{ "error": "..." }` | Invalid input (bad JSON, unknown sale type, missing fields) |
| 401 | `{ "error": "..." }` | Missing, invalid or revoked API key |
| 403 | `{ "error": "..." }` | Key not allowed for this operation or company |
| 404 | `{ "error": "not found" }` | Unknown id or reference |
| 409 | `{ "error": "already exists" }` | Conflict (e.g. duplicate code) |
| 500 | `{ "error": "internal error: ..." }` | Unexpected server error; retry later and contact support |

A rejection by FBR is **not** an HTTP error. The call succeeds and the invoice has `status: "REJECTED"`.

## 5. Recommended integration flow

1. Sync customers and products (`/customers`, `/products`) or send full line details each time.
2. `POST /invoices` with `externalRef` and `"submit": true`.
3. On a network error or timeout between the ERP and this server, simply **repeat the same request**. The `externalRef` guarantees no duplicate.
4. Store `id`, `status` and `fbrInvoiceNumber`. Print only `ACCEPTED` invoices, using `/print` or your own layout with `/qr.png`.
5. Poll `QUEUED`/`SUBMITTING` invoices every minute or so via `/invoice-by-ref/{externalRef}`.

## 6. Code samples

### curl

```sh
curl --cacert cert.pem -X POST "https://einvoice-server:8443/api/v1/companies/1/invoices" \
  -H "X-API-Key: $EINV_KEY" -H "Content-Type: application/json" \
  -d @invoice.json
```

### Python (requests)

```python
import requests

BASE = "https://einvoice-server:8443/api/v1/companies/1"
HEADERS = {"X-API-Key": "eik_..."}
invoice = {
    "externalRef": "ERP-1001",
    "buyer": {"ntnCnic": "2046004", "name": "ABC Traders", "province": "PUNJAB",
              "address": "Lahore", "registrationType": "Registered"},
    "items": [{"hsCode": "0101.2100", "description": "Product A", "uom": "Numbers, pieces, units",
               "quantity": 10, "unitPrice": 150, "saleType": "Goods at standard rate (default)", "rate": "18%"}],
    "submit": True,
}
r = requests.post(f"{BASE}/invoices", json=invoice, headers=HEADERS, verify="cert.pem", timeout=60)
if r.status_code == 422:
    for issue in r.json().get("issues", []):
        print(issue["line"], issue.get("code"), issue["message"])
else:
    r.raise_for_status()
    inv = r.json()
    print(inv["status"], inv["fbrInvoiceNumber"])
```

### C# (HttpClient)

```csharp
using System.Net.Http.Json;

var http = new HttpClient { BaseAddress = new Uri("https://einvoice-server:8443/api/v1/companies/1/") };
http.DefaultRequestHeaders.Add("X-API-Key", "eik_...");
var invoice = new {
    externalRef = "ERP-1001",
    buyer = new { ntnCnic = "2046004", name = "ABC Traders", province = "PUNJAB", address = "Lahore", registrationType = "Registered" },
    items = new[] { new { hsCode = "0101.2100", description = "Product A", uom = "Numbers, pieces, units",
                          quantity = 10, unitPrice = 150, saleType = "Goods at standard rate (default)", rate = "18%" } },
    submit = true
};
var resp = await http.PostAsJsonAsync("invoices", invoice);
var body = await resp.Content.ReadFromJsonAsync<System.Text.Json.JsonElement>();
Console.WriteLine($"{(int)resp.StatusCode} {body.GetProperty("status")} {body.GetProperty("fbrInvoiceNumber")}");
```

(Install the server certificate in the Windows certificate store so that `HttpClient` trusts it. On a 422 response, read `error` and `issues` instead of `status`.)

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
