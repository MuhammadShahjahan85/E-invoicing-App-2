# User Manual

This manual is for accountants, billing staff and managers who issue sales-tax invoices with Veridian E-invoicing PK.

---

## 1. Getting started

### Logging in

Open the address given by your administrator, e.g. `https://server:8443/`, and sign in. After five wrong passwords the account is locked for 15 minutes. If an administrator set or reset your password, you must choose a new one at first login: 8 or more characters, with letters and digits.

### Roles

| Role | Can do |
|---|---|
| **admin** | Everything, including users, licence, backups and API keys |
| **manager** | Company settings, FBR tokens, invoices, cancellations, scenarios, reports, audit trail |
| **accountant** | Customers, products, invoices, debit notes, cancellations, reports, incidents |
| **operator** | Create and submit invoices (counter/POS staff) |
| **auditor** | Read-only: invoices, reports, audit trail |

### Screen layout

- **Left menu:** Sales, Masters, Compliance and Settings.
- **Top bar:** company selector (for multi-company installations), current environment, **+ New invoice**, Password and Log out.
- **Coloured banner:** shows when you are **not** in production.
  - *Training simulator:* nothing is sent to FBR.
  - *FBR sandbox:* test submissions for certification.
  - Invoices from either are watermarked and are not valid tax invoices.

### Dashboard

![Dashboard](images/dashboard.png)

Shows:

- invoices accepted today and this month (value and sales tax);
- items that need attention (rejected, queued, needing reconciliation);
- FBR connection health and go-live readiness;
- the most frequent FBR errors;
- warnings: licence, token expiry, unreported incidents, and invoices **not yet reported to FBR** with the age of the oldest one.

## 2. Customers (buyers)

**Masters → Customers (buyers) → + New customer**

| Field | Guidance |
|---|---|
| Name | Buyer's name as registered |
| NTN / CNIC | 7-digit NTN (without the check digit after the dash) or 13-digit CNIC, no dashes. **Check FBR** queries FBR's Active Taxpayer List and registration type. |
| Registration type | *Registered* or *Unregistered*. Taxable supplies to unregistered buyers attract **further tax (4%)**. A registered buyer must have an NTN/CNIC. |
| Province (destination of supply) | Required by FBR |
| Sales tax withholding agent | If the buyer withholds sales tax: *fraction* (default one-fifth) or *full* |

## 3. Products & services

**Masters → Products & services → + New product**

| Field | Guidance |
|---|---|
| HS code (PCT) | Format `NNNN.NNNN`; type to search the HS list |
| Unit of measure | FBR's UoM text. For some HS codes FBR expects a particular UoM; the hint shows it. |
| Sale type | FBR's sale type, e.g. *Goods at standard rate (default)*, *Goods at Reduced Rate*, *3rd Schedule Goods*, *Exempt goods*, *Goods at zero-rate*, *Services* … |
| Rate | Pick from FBR's list for the sale type, or type it, e.g. `18%`, `5%`, `Exempt`, `Rs.200`, `18% along with rupees 60 per kilogram` |
| SRO / Schedule no., SRO item serial no. | Required for reduced-rate, exempt and other SRO-based supplies. Lists come from FBR when available. |
| Printed retail price | Third Schedule goods: tax is charged on the retail price |
| Further tax | *Automatic* (unregistered buyers), *Always* or *Never* |
| Extra tax / FED rate | Only where applicable |

## 4. Creating a sales tax invoice

**+ New invoice**

![Invoice editor](images/invoice-editor.png)

1. **Document:** type (*Sale Invoice* or *Debit Note*) and invoice date (today; report at the time of supply). Optionally add your reference, such as an order or ERP number; it prevents duplicates.
2. **Buyer:** search the customer master, or type the buyer's details for a one-off sale (e.g. *Walk-in customer*, Unregistered).
3. **Lines:** type to search products, or enter a description, then set HS code, UoM, quantity, price, sale type and rate. Click **▸ SRO, retail price, further/extra tax, discount…** for the extra fields.
4. Taxes and totals are **calculated live** as you type:
   - value excluding tax, sales tax, further tax, extra tax, FED, withholding, total and amount payable;
   - warnings appear under the line.
5. Click **Save & submit to FBR**. **Save draft** keeps it unreported.

After submission:

- **Accepted by FBR:** the FBR invoice number and QR code appear. Print it.
- **Rejected:** the FBR errors are shown per line with *How to fix*. Click **Edit**, correct it, and submit again.
- **Errors before sending:** if the product finds problems itself (e.g. missing NTN for a registered buyer, invalid HS code), the invoice is not sent and the issues are listed.

### Special cases

| Case | How |
|---|---|
| Unregistered buyer | Further tax at 4% is added automatically to taxable lines (not to exempt or zero-rated lines). If your company is a **manufacturer or importer**, record the buyer's CNIC or NTN: section 23(1)(b) requires it and the system warns when it is missing. Other sellers are warned above Rs 100,000. |
| Third Schedule goods | Enter the printed retail price per unit; sales tax is charged on the retail value |
| Reduced rate / exempt / zero-rated | Choose the sale type and rate, and enter the SRO schedule and serial number |
| Withholding agent buyer | Set on the customer; *Amount payable* is reduced by the withheld tax |
| Discounts | Enter an amount or percentage in the line details |

## 5. Invoice statuses

![Invoice list](images/invoices.png)

| Status | Meaning | What to do |
|---|---|---|
| **Draft** | Saved, not reported | Edit or submit |
| **Validated** | FBR validation passed, not yet reported | Submit |
| **Queued** | FBR could not be reached; the system retries automatically and resends every queued invoice as soon as the connection is back | Nothing — it is sent when FBR is reachable (**Retry now** to try at once). FBR requires invoices issued offline to be uploaded within 24 hours of the connection being restored, so check the dashboard after an outage. **Print provisional copy** gives the customer a copy marked "PENDING FBR REPORTING"; print the final copy once it is accepted. |
| **Submitting** | Being sent | Wait |
| **Accepted by FBR** | Reported; FBR number issued; locked | Print, deliver |
| **Rejected** | FBR refused it | Edit and resubmit |
| **Needs reconciliation** | No definite answer received; FBR may have recorded it | A manager/accountant checks IRIS and uses **Reconcile with IRIS** |
| **Cancelled** | Cancelled after FBR cancellation | — |

Accepted invoices **cannot be edited or deleted**. Corrections are made by debit note or by cancellation.

## 6. Printing

![Accepted invoice](images/invoice-view.png)

On an accepted invoice, click **Print A4** or **Print receipt (80 mm)**. The print shows:

- seller and buyer particulars;
- the lines with HS codes, rates and taxes;
- amount in words;
- the **FBR invoice number**, **QR code** and **FBR Digital Invoicing logo**.

![Printed invoice](images/print-a4.png)

The first print is the original. Later prints are marked **DUPLICATE**.

Copies per print (buyer / seller / office copy), logo, terms and footer are set under **Settings → Invoice printing**.

## 7. Debit notes

Use a debit note when goods are **returned** or the **value of a reported sale is reduced** after supply, for example a post-sale discount or short supply.

1. Open the accepted sale invoice and click **Debit note**. A draft debit note opens with the original lines and the original FBR invoice number.
2. Reduce the quantities and values to the returned or reduced amount. A note cannot exceed the original invoice's value or sales tax.
3. Click **Save & submit to FBR**.

The original invoice shows the debit notes accepted against it. Reports list debit notes separately and deduct them from output tax ("Net sales tax" in the monthly summary).

For an upward price revision, issue a supplementary sale invoice for the difference instead.

## 8. Cancelling an invoice

FBR allows cancellation or editing through its system only **within 72 hours** of issue. After that, the **Commissioner's prior approval** is required (STGO 01 of 2026).

1. Cancel the invoice on **IRIS → Digital Invoicing**.
2. In the product, open the invoice → **Cancel invoice**. Enter the reason and the IRIS reference. After 72 hours, also enter the Commissioner's approval reference.

If your administrator has enabled FBR's cancellation service (Installation guide, `fbr.endpoints.cancelPath`), the dialog offers **Cancel with FBR** instead. The request goes straight to FBR, and the invoice is marked cancelled only when FBR confirms it.

The invoice is then marked cancelled and prints with a CANCELLED watermark.

## 9. Needs reconciliation

This appears when the connection broke after the invoice was sent, so FBR may or may not have recorded it.

1. Search IRIS for the invoice (date, buyer, value).
2. Open the invoice → **Reconcile with IRIS** and choose one:
   - **IRIS shows this invoice** → enter the FBR invoice number;
   - **IRIS does not show it** → **resubmit automatically**, or **return to draft for correction**.

Never re-enter the sale as a new invoice; it could be reported twice.

## 10. Importing invoices (CSV / Excel)

**Import (CSV / Excel)**

1. Download the Excel or CSV template.
2. Fill one row per invoice line. Rows with the same `invoice_ref` form one invoice.
3. Choose the file and click **2. Preview & check**. Every invoice is calculated and validated, and nothing is saved.
4. Click **3. Import as drafts**, or tick *Submit each imported invoice to FBR immediately*.

Re-importing a file never duplicates invoices whose `invoice_ref` already exists.

## 11. Reports

![Reports](images/reports.png)

**Reports** includes only documents accepted by FBR:

- **Sales register (line level):** reconcile with Annexure-C of the sales tax return;
- **Tax summary by sale type & rate**;
- **Monthly summary (tax periods):** sale invoices, debit notes, net sales tax, tax withheld by buyers;
- **Buyer-wise summary**;
- **FBR API log:** every request and response exchanged with FBR (security tokens are never logged).

Choose the period and environment, then **Download CSV** or **Download Excel**.

## 12. Sandbox scenarios

![Scenarios](images/scenarios.png)

Used once per company before going live (see [FBR-ONBOARDING-GUIDE.md](FBR-ONBOARDING-GUIDE.md)):

- set the scenarios assigned on IRIS;
- practise them in the simulator;
- run them in the FBR sandbox until all show **passed**.

## 13. Incident register

Rule 150R requires reporting failures, disruptions or tampering of the e-invoicing system to the Commissioner **within 24 hours**.

- Loss of FBR connectivity and token failures are recorded automatically.
- Record other incidents with **+ Record incident** (power failure, hardware/software failure, suspected tampering).
- **Letter** prints the intimation to the Commissioner with the list of affected invoices.
- After sending it, edit the incident and enter the date and reference. Incidents not reported within 24 hours are flagged **overdue**.

## 14. Audit trail

Every action is logged with user, time and details: logins, invoices, changes, cancellations, settings. Entries are chained by SHA-256 hashes. **Verify integrity** proves that no entry was altered or removed.

## 15. Using a phone or tablet

Open **Mobile app & access** in the menu. It has a QR code to open the system on your phone, an **Install** button (or instructions for Android and iPhone) and the certificate the phone needs to trust the server. On a phone, the menu opens from the **☰** button. See [MOBILE-AND-REMOTE-ACCESS.md](MOBILE-AND-REMOTE-ACCESS.md).

## 16. Settings

| Page | Purpose |
|---|---|
| Company | Seller particulars, business activity and sector, invoice prefixes, further-tax rate, withholding fraction, validate-before-post. **+ Add another company** for multi-company installations. |
| FBR integration | Working environment (training simulator / FBR sandbox / FBR production), sandbox and production tokens with expiry, **Test connection**, reference data download, go-live requirements |
| Invoice printing | Default format, amount-in-words style, columns, copies, terms, footer, company logo |
| Users & roles | Create users, assign roles and companies, reset passwords, deactivate leavers |
| ERP API keys | Keys for ERP/POS integration ([ERP-INTEGRATION-API.md](ERP-INTEGRATION-API.md)) |
| System | Licence status and installation, backups (create/download), FBR Digital Invoicing logo, server information |
| Help & error codes | FBR error codes with fixes, compliance rules, sale types and rates |

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing PK is proprietary software.
