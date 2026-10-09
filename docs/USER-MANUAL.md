# User Manual

This manual is for accountants, billing staff and managers who issue sales-tax invoices with Veridian E-invoicing Pakistan.

---

## 1. Installing and opening the app

The program is installed once, on the computer that keeps the data. In a larger office your IT person may install it on a server instead; see the [Installation guide](INSTALLATION.md).

1. Double-click **VeridianEInvoicingPakistan-Setup-*version*.exe** and allow it to make changes to the computer.
2. Accept the licence.
3. Keep the recommended components:
   - **Veridian E-invoicing Pakistan** — the program and the background service that keeps it running;
   - **Trust the security certificate** — the app opens securely in Edge and Chrome, without a warning;
   - **Desktop icon**;
   - **Office network access** — other computers and phones in the office can use the app.
4. Keep the suggested folder and click **Install**.
5. On the last page keep **Open Veridian E-invoicing Pakistan now** ticked and click **Finish**.

![Setup: choose components](images/setup-components.png)

The first time the app opens, a short **setup wizard** asks for the administrator account, your business exactly as registered on IRIS (name, NTN/CNIC, STRN, province and address) and your Digital Invoicing profile (business activity and sector). You then work in the **training simulator**: nothing is sent to FBR until you connect the company to FBR (see the [FBR onboarding guide](FBR-ONBOARDING-GUIDE.md)).

### Opening the app every day

Double-click **Veridian E-invoicing Pakistan** on the desktop, or open it from the Start menu. It opens in a window of its own; there is no command window to keep open. The program keeps running in the background even when the window is closed, so invoices queued during an internet outage are still sent to FBR.

The Start menu also has **User guide** (this guide) and **Help and error codes**.

To remove the program, use **Settings → Apps** in Windows. Your data folder is kept, because sales tax records must be kept for six years; installing again picks it up.

## 2. Getting started

### Logging in

- **On the computer where it is installed:** open the app from its desktop icon.
- **On other computers and phones in the office:** open the address given by your administrator, e.g. `https://server:8443/`, in Chrome or Edge. Browsers can install it as an app with its own icon (see *Using a phone or tablet*).

Sign in. After five wrong passwords the account is locked for 15 minutes. If an administrator set or reset your password, you must choose a new one at first login: 8 or more characters, with letters and digits.

### Roles

| Role | Can do |
|---|---|
| **admin** | Everything, including users, licence, backups and API keys |
| **manager** | Company settings, FBR tokens, invoices, cancellations, scenarios, reports, audit trail |
| **accountant** | Customers, products, invoices, debit notes, cancellations, reports, incidents |
| **operator** | Create and submit invoices (counter/POS staff) |
| **auditor** | Read-only: invoices, reports, audit trail |

### Screen layout

![Dashboard](images/dashboard.png)

- **Left menu:** your company, with its environment and NTN (click it to switch between companies), then *Sales*, *Masters*, *FBR & compliance*, *Settings* and *Support*. **Collapse** at the bottom folds the menu into icons for more room. On a phone the menu opens from the **☰** button.
- **Top bar:**
  - the page name and the **status chip**, which says at a glance where you stand with FBR: *Simulator · not sent to FBR*, *All reported · 0 pending*, *To review · n* (invoices rejected, queued or awaiting reconciliation) or *FBR · not reachable*. Click it to go where action is needed;
  - **As at** — today's date in Pakistan;
  - **Search** (or press **Ctrl K** or **/**): finds invoices by number, FBR invoice number, buyer or NTN; buyers; products; FBR HS codes; and pages such as *Verify a buyer* or *Sales tax calculator*. Use the arrow keys and **Enter** to open a result;
  - the **moon / sun** button switches between the dark and light theme;
  - the **bell** (notifications, see below);
  - your **initials**: appearance (light, dark or automatic), **Amounts** (see below), **Urdu captions**, **Keyboard shortcuts**, change password, mobile app, help and **Log out**.
- A thin line under the top bar moves while the system is working, for example while an invoice is sent to FBR.
- **Coloured banner:** shows when you are **not** in production.
  - *Training simulator:* nothing is sent to FBR.
  - *FBR sandbox:* test submissions for certification.
  - Invoices from either are watermarked and are not valid tax invoices.

![Search](images/search.png)

### Made for Pakistan

| Feature | Where | What it does |
|---|---|---|
| **Lakh and crore** | Initials → **Amounts** | Shows amounts as 12,34,567.00 instead of 1,234,567.00. On the dashboard, point at a large amount to read it in words, e.g. *about Rs 43.44 lakh*. |
| **Urdu captions** | Initials → **Urdu captions** | Adds Urdu captions to the menu sections, page titles and the tax tip, e.g. *فروخت* under Sales. The app itself stays in English, as on FBR's systems. |
| **Tax tip** | Dashboard | A short tip on Pakistani sales tax and FBR's rules every day, in English and Urdu. Use the arrows for more. |
| **Cities and provinces** | Customers, invoice buyer | Type a city (Karachi, Lahore, Faisalabad, Peshawar, Quetta …) and the province FBR needs is filled in for you; the buyer's address on an invoice does the same. |
| **NTN or CNIC** | Customers, invoice buyer | As you type, the number is shown in its usual form, e.g. *CNIC 35202-1234567-1*, so mistakes are easy to spot. |
| **Greeting** | Dashboard | *Assalam-o-Alaikum* and today's date in Pakistan time. |

![Lakh and crore amounts and Urdu captions](images/made-for-pakistan.png)

### Keyboard shortcuts

Press **?** anywhere (or choose **Keyboard shortcuts** under your initials) to see this list.

| Keys | Opens |
|---|---|
| **Ctrl K** or **/** | Search |
| **Alt N** | New invoice |
| **Alt I** | Invoices |
| **Alt D** | Dashboard |
| **Alt U** | Import data |
| **Alt B** | Customers (buyers) |
| **Alt P** | Products & services |
| **Alt T** | Tax periods & returns |
| **Alt R** | Reports |
| **Alt L** | FBR reference library |

### Notifications (the bell)

The red number counts what needs action now; a blue dot means there is only information. Notifications are worked out from the current state, so each disappears by itself once its cause is dealt with. Click one to go to the page where you can fix it:

| Notification | What to do |
|---|---|
| FBR not reachable / token rejected | Check the internet connection or the token under **FBR integration**. Invoices are queued meanwhile. |
| Invoices not yet reported to FBR | Shown in red after 24 hours: invoices issued during an outage are marked as issued in the offline mode and must be uploaded within 24 hours of the connection being restored (rule 150XC). |
| Invoices rejected by FBR | Correct the errors and resubmit. |
| Submissions need reconciliation | Check the invoice on IRIS and record the outcome. |
| Draft invoices not yet issued | Issue them when the supply is made, or delete them. |
| Incidents not reported to FBR | Rule 150XA(c): report to FBR and the Commissioner within 24 hours and record the reference. |
| Pay sales tax / file the return in *n* days | Review the period on **Tax periods & returns**. |
| FBR reference data not downloaded / out of date | Download it under **FBR reference library → Data sync**. |
| Token or licence expiring, no recent backup | Renew, or check the backups (administrators). |

![Notifications](images/notifications.png)

The same items can be **e-mailed** to the people responsible, so problems are seen even when nobody has the system open — see *E-mail notifications* in section 16.

### Dashboard

The dashboard, **Sales tax overview**, answers three questions: how much have we sold, has FBR received everything, and is the month ready to close?

- **Sales this month:** the value of sale invoices accepted by FBR, excluding sales tax, with the sales tax, today's sales and the number of invoices that **need attention** (rejected, queued or awaiting reconciliation).
- **Closing *month*:** the four steps to close the tax period whose return is due next — invoices issued, reported to FBR, period checks and filing the return, with its due date. A tick means the step is done; click a step to go to it.
- **Books, FBR and Annexure-C tie:** a triangle that shows whether your invoices (the books), FBR's records and Annexure-C of the return agree. A green tick on a side means it ties; an amber mark shows the documents to review, with a link to them.
- **Report pack:** one PDF for the period, to print or send to your tax adviser (see *Tax periods & returns*).
- **Tax tip**, sales by month for the last 12 months (point at a column for its figures, or click **Show table**), the sales tax return due dates, invoices by status, the top buyers and HS codes of the month and the latest invoices.
- Go-live readiness, the state of FBR reference data, the most frequent FBR errors, and warnings: licence, token expiry, unreported incidents, and invoices **not yet reported to FBR** with the age of the oldest one.

Sales figures by month, buyer and HS code are shown to roles that can see reports.

![Dashboard in the dark theme](images/dashboard-dark.png)

## 3. Customers (buyers)

**Masters → Customers (buyers) → + New customer**

| Field | Guidance |
|---|---|
| Name | Buyer's name as registered |
| NTN / CNIC | 7-digit NTN (without the check digit after the dash) or 13-digit CNIC, no dashes. **Check FBR** queries FBR's Active Taxpayer List and registration type. |
| Registration type | *Registered* or *Unregistered*. Taxable supplies to unregistered buyers attract **further tax (4%)**. A registered buyer must have an NTN/CNIC. |
| City | Pick or type the city; the province is filled in for it |
| Province (destination of supply) | Required by FBR |
| Sales tax withholding agent | If the buyer withholds sales tax: *fraction* (default one-fifth) or *full* |

**Check all with FBR** (on the Customers page) checks every buyer that has an NTN or CNIC, one after another, and records the result with the date. A buyer whose registration type in your master differs from FBR's record is flagged **FBR says …** so that further tax is charged correctly.

## 4. Products & services

**Masters → Products & services → + New product**

| Field | Guidance |
|---|---|
| HS code (PCT) | Format `NNNN.NNNN`; type to search the HS list |
| Unit of measure | FBR's UoM text. For some HS codes FBR expects a particular UoM; the hint shows it. |
| Sale type | FBR's sale type, e.g. *Goods at standard rate (default)*, *Goods at Reduced Rate*, *3rd Schedule Goods*, *Exempt goods*, *Goods at zero-rate*, *Services* … |
| Rate | Pick from FBR's list for the sale type, or type it, e.g. `18%`, `5%`, `Exempt`, `Rs.200`, `18% along with rupees 60 per kilogram` |
| SRO / Schedule no., SRO item serial no. | Required for reduced-rate, exempt and other SRO-based supplies. Lists come from FBR when available. |
| Printed retail price | Third Schedule goods: the price printed on the pack, which includes sales tax. Tax = printed price × rate ÷ (100 + rate), e.g. Rs 118 → Rs 18 at 18% |
| Further tax | *Automatic* (unregistered buyers), *Always* or *Never* |
| Extra tax / FED rate | Only where applicable |
| FED type, FED rate as printed, FED Schedule / SRO, FED serial no. | For goods or services subject to federal excise duty. They are copied to each invoice line and printed, as rule 150R(13)(aa)–(ff) requires since SRO 1666(I)/2026. For a specific duty (rupees per unit), enter the rate as it should be printed, e.g. *Rs 4 per kg*, and the amount on the invoice line. |

## 5. Creating a sales tax invoice

**+ New invoice**

![Invoice editor](images/invoice-editor.png)

1. **Document:** type (*Sale Invoice* or *Debit Note*) and invoice date (today; report at the time of supply). Optionally add your reference, such as an order or ERP number; it prevents duplicates. Tick **Advance receipt invoice** when you have received payment before supplying (see *Special cases*).
2. **Buyer:** search the customer master, or type the buyer's details for a one-off sale (e.g. *Walk-in customer*, Unregistered). **Check with FBR** next to the NTN/CNIC asks FBR's Active Taxpayer List and registration type and sets *Registered* or *Unregistered* for you; a buyer picked from the master shows the status on file.
3. **Lines:** type to search products, or enter a description, then set HS code, UoM, quantity, price, sale type and rate. If FBR prescribes a different unit for the HS code, the unit is flagged with **Use** to apply FBR's unit. Click **▸ SRO, discount, FED and more** for the extra fields (SRO, printed retail price, discount, further and extra tax, FED), including **Pick SRO / schedule from FBR** and **Pick serial no. from FBR**.
4. Taxes and totals are **calculated live** as you type:
   - value excluding tax, sales tax, further tax, extra tax, FED, withholding, total and amount payable;
   - warnings appear under the line; FBR's checks are listed at the top once you start entering the invoice. Put each product on one line: FBR may treat the same product on two lines as a duplicate, and the system warns about it.
5. Click **Save & submit to FBR**. **Save draft** keeps it unreported.

After submission:

- **Accepted by FBR:** the FBR invoice number and QR code appear. Print it.
- **Rejected:** the FBR errors are shown per line with *How to fix*. Click **Edit**, correct it, and submit again.
- **Errors before sending:** if the product finds problems itself (e.g. missing NTN for a registered buyer, invalid HS code), the invoice is not sent and the issues are listed.

### Special cases

| Case | How |
|---|---|
| Unregistered buyer | Further tax at 4% is added automatically to taxable lines (not to exempt or zero-rated lines). If your company is a **manufacturer or importer**, record the buyer's CNIC or NTN: section 23(1)(b) requires it and the system warns when it is missing. Other sellers are warned above Rs 100,000. |
| Third Schedule goods | Enter the retail price printed on the pack per unit (it includes sales tax). The system reports the retail value excluding sales tax to FBR and charges 18% on it, i.e. printed price × 18 ÷ 118 |
| Goods with federal excise duty | Set the FED rate (or, for a specific duty, the FED amount on the line). FED is part of the value of supply, so sales tax and further tax are charged on value + FED (section 2(46)). Enter the **FED particulars** — type, rate as printed, price per unit, FED Schedule/SRO and serial no. — in the line details or once on the product; they are printed under the line (rule 150R(13)(aa)–(ff)). The system warns when they are missing. |
| Advance received before supply | Sales tax is due at the time of supply — delivery or payment, whichever is earlier (section 2(44)) — and since the Finance Act 2026 section 23(1) requires an invoice with an FBR number for an advance receipt. Tick **Advance receipt invoice** and enter the advance (excluding sales tax) as the value. It is reported to FBR and printed as an **ADVANCE RECEIPT INVOICE**. On delivery, invoice only the balance and enter the advance invoice's FBR number under **Advance receipt invoices adjusted**. |
| Reduced rate / exempt / zero-rated | Choose the sale type and rate, and enter the SRO schedule and serial number |
| Withholding agent buyer | Set on the customer; *Amount payable* is reduced by the withheld tax |
| Discounts | Enter an amount or percentage in the line details |

### Duplicating an invoice

For repeat sales, open an earlier sale invoice and click **Duplicate**. A new draft dated today is created with the same buyer and lines (taxes are recalculated); check it and submit it.

## 6. Invoice statuses

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

## 7. Printing, PDF and sharing

![Accepted invoice](images/invoice-view.png)

On an accepted invoice, click **Print A4** or **Print receipt (80 mm)**, or **PDF** to download the invoice as a PDF document. The print and the PDF show:

- seller and buyer particulars;
- the lines with HS codes, rates and taxes;
- amount in words;
- the **FBR invoice number**, **QR code** and **FBR Digital Invoicing logo** (the official logo is built in);
- the tax period, the HS code of each line, the total discount and the tax withheld;
- the **software registration number** (Settings → FBR integration) and the **digital signature** with the signing key's fingerprint (rule 150R(4)(b));
- FED particulars under each line subject to federal excise duty;
- for invoices issued while FBR was unreachable, a note that the invoice was issued in offline mode and when FBR received it (rule 150XC).

![Advance receipt invoice with FED particulars](images/advance-receipt-invoice.png)

![Printed invoice](images/print-a4.png)

The first print is the original. Later prints are marked **DUPLICATE**.

Copies per print (buyer / seller / office copy), logo, terms and footer are set under **Settings → Invoice printing**.

### Sharing an invoice

**Share** on an accepted invoice offers:

- **Share the PDF** — on a phone, opens the share sheet with the invoice PDF attached (WhatsApp, e-mail and so on); on a computer, downloads the PDF to attach yourself;
- **Send details on WhatsApp** — the invoice number, date, buyer, amounts and **FBR invoice number** as a message;
- **Copy details** — the same details, ready to paste into an e-mail.

The printed or PDF copy remains the tax invoice.

## 8. Debit notes

Use a debit note when goods are **returned** or the **value of a reported sale is reduced** after supply, for example a post-sale discount or short supply.

1. Open the accepted sale invoice and click **Debit note**. A draft debit note opens with the original lines and the original FBR invoice number.
2. Reduce the quantities and values to the returned or reduced amount. A note cannot exceed the original invoice's value or sales tax.
3. Click **Save & submit to FBR**.

The original invoice shows the debit notes accepted against it. Reports list debit notes separately and deduct them from output tax ("Net sales tax" in the monthly summary).

For an upward price revision, issue a supplementary sale invoice for the difference instead.

## 9. Cancelling an invoice

FBR allows cancellation or editing through its system only **within 72 hours** of issue. After that, the **Commissioner's prior approval** is required (STGO 01 of 2026).

1. Cancel the invoice on **IRIS → Digital Invoicing**.
2. In the product, open the invoice → **Cancel invoice**. Enter the reason and the IRIS reference. After 72 hours, also enter the Commissioner's approval reference.

If your administrator has enabled FBR's cancellation service (Installation guide, `fbr.endpoints.cancelPath`), the dialog offers **Cancel with FBR** instead. The request goes straight to FBR, and the invoice is marked cancelled only when FBR confirms it.

The invoice is then marked cancelled and prints with a CANCELLED watermark.

## 10. Needs reconciliation

This appears when the connection broke after the invoice was sent, so FBR may or may not have recorded it.

1. Search IRIS for the invoice (date, buyer, value).
2. Open the invoice → **Reconcile with IRIS** and choose one:
   - **IRIS shows this invoice** → enter the FBR invoice number;
   - **IRIS does not show it** → **resubmit automatically**, or **return to draft for correction**.

Never re-enter the sale as a new invoice; it could be reported twice.

## 11. Importing invoices from any file

**Sales → Import data** brings in invoices from your accounting software, ERP or spreadsheets. Upload the file **as it is**: there is no template to fill in.

![Import: matching columns](images/import-match.png)

1. **Choose file.** Drop the file on the page or click to choose it. Excel (.xlsx, .xls), OpenDocument (.ods), CSV and other separated text, fixed-width text reports, JSON, XML, Word (.docx), web pages (.html) and PDF reports are read. Titles, blank lines and totals rows are skipped. Up to 20 MB and 20,000 lines.
2. **Match columns.** Each column of your file is shown with sample values and the field it was matched to, such as *Invoice number*, *Date*, *Buyer NTN/CNIC*, *HS code*, *Quantity* or *Value excl. sales tax*. Change any match under **Use as**, or choose *Don't import* for a column you do not need. When a file has several sheets or tables, pick the one to use; if the headings are not on the first row, set **Headings are on row**.
![Import: what FBR needs that the file lacks](images/import-prompt.png)

3. **Missing details.** A message lists what FBR needs that your file does not hold — for example the buyer's province, the sale type and rate, or FBR's spelling of a unit (*Cartons* → *Packs*). Supply each one **once for the whole file**: pick a value, take it from another column, or map each value in your file to FBR's.
4. **Check & import.** Every invoice is calculated and checked against FBR's rules, and nothing is saved yet. The table shows each invoice with its total and any problems. Then click **Import** to save them as drafts, or tick **Submit each invoice to FBR right away**.

![Import: missing details](images/import-missing.png)

Invoices already imported (same invoice number) are never created twice, so a corrected file can simply be uploaded again. **Excel template** gives a sheet with every column the import understands, if you prefer to prepare a file yourself; it includes the optional columns for advance receipt invoices and FED particulars.

## 12. Reports

![Reports](images/reports.png)

**Reports** includes only documents accepted by FBR:

- **Sales register (line level):** reconcile with Annexure-C of the sales tax return;
- **Tax summary by sale type & rate**;
- **Monthly summary (tax periods):** sale invoices, debit notes, net sales tax, tax withheld by buyers;
- **Buyer-wise summary**;
- **Annexure-C reconciliation:** every document that received an FBR invoice number in the period, including those cancelled later, with the buyer, the tax amounts and whether it was issued offline. Since SRO 1666(I)/2026 (rule 150XD(2)), tax is recovered on any invoice reported with an FBR number but not accounted for in Annexure-C or the return, so match each line before filing. The same file answers FBR's electronic scrutiny intimations (SRO 1655(I)/2026);
- **FBR API log:** every request and response exchanged with FBR (security tokens are never logged).

Choose the period and environment, then download the report as **PDF** (formatted, ready to print or send), **Excel** or **CSV**. PDF reports carry the company's particulars, the period and page numbers; reports that mix sale invoices and debit notes are totalled net of the debit notes.

## 13. Tax periods & returns

![Tax periods & returns](images/compliance.png)

**FBR & compliance → Tax periods & returns** (roles that can see reports). Invoices reported through Digital Invoicing populate Annexure-C of the monthly sales tax return on IRIS (shown there as *Annex-C*), so review each tax period before filing:

- **Due dates** with a countdown: by default tax is paid by the 15th and the return filed by the 18th of the following month. Change the days under **Settings → Company → Sales tax return calendar** if your sector has other dates. When FBR extends the filing date for a period (by notification or circular), click **Record an FBR extension**, enter the new date and FBR's reference: reminders and the checklist then use it, and the original date is shown alongside. The payment date does not move — FBR's extensions are normally conditional on the tax being paid on time.
- **Period-close checklist:** every issued invoice reported; no rejected invoices left uncorrected; no submissions awaiting reconciliation; no unissued drafts dated in the period; incidents reported. **Review** opens the matching invoices.
- The period's value of supplies, sales tax, further tax, debit notes, net sales tax and tax withheld by buyers.
- **Supplies by sale type and rate** — the figures that feed Annexure-C — and the largest buyers and HS codes.
- Incidents that overlapped the period.
- **Annexure-C reconciliation** for the period, as Excel or PDF.
- **Report pack (PDF):** one document with the period's figures, the books–FBR–Annexure-C tie, the checklist, tax by sale type and rate, every document reported to FBR and a sign-off block — for your files, your tax adviser or an FBR enquiry.
- **Day, week and month closings** (rule 150R(4)(f)): the system records a closing automatically within an hour after each day, week (Monday to Sunday) and month ends — documents issued, reported, not reported and cancelled, the values and tax reported, and the first and last FBR numbers. Closings are chained by hash and cannot be changed; **Chain intact** shows they have not been tampered with.

![Day, week and month closings](images/closings.png)

## 14. Stock transfer notes

**Sales → Stock transfer notes.** Goods moved from your factory to your own warehouse registered under the **same STRN** are not a supply, so no invoice is reported to FBR. Sales Tax General Order 25 of 2026 requires each consignment to travel with a serially numbered stock transfer note instead.

1. Click **New transfer note**. Enter the dispatch date and time, the receiving warehouse and its address, the vehicle number, the driver's CNIC and who authorised the dispatch.
2. Add the goods: search products (the HS code and unit are filled in) or describe them, with quantity and value at cost.
3. Tick the confirmation that the warehouse is registered under your own STRN — if it has a separate STRN, the movement is a taxable supply and needs a sales tax invoice.
4. **Save and print** numbers the note (STN-000001, STN-000002 …) and prints a despatch copy and a warehouse copy.
5. When the goods arrive, click **Received** and record who signed the warehouse copy, and when. A note issued in error can be **cancelled** with a reason; it stays in the register so the series has no gaps.

**Register (CSV)** downloads the notes for the monthly reconciliation with your stock records. Notes are kept for six years like invoices.

![Stock transfer notes](images/stock-transfers.png)

## 15. FBR reference library

![FBR reference library](images/library-rates.png)

**FBR & compliance → FBR reference library** shows the lists FBR maintains for Digital Invoicing, downloaded with the company's security token:

| Tab | Use it to |
|---|---|
| HS codes | Search FBR's HS (PCT) codes by number or description; click a code for the unit of measure FBR prescribes for it (HS_UOM). |
| Rates & SROs | Pick a sale type and date: the rates FBR accepts (SaleTypeToRate) → the SROs or schedules linked to a rate → their serial numbers. These are the values to put on the invoice. |
| Units | FBR's units of measure, spelt as FBR expects them. |
| Provinces & types | Province codes, document types and transaction (sale) types. |
| Buyer check (ATL) | Check an NTN or CNIC against FBR's Active Taxpayer List and registration type, with what it means for the invoice (registered or unregistered buyer, further tax). |
| Tax calculator | Work out sales tax, further tax, extra tax, FED, withholding and the invoice total for a sale, using the same engine as invoices. |
| Data sync | See what is stored on the server and **Download from FBR now** (company settings permission). Refresh after FBR announces new sale types, rates or SROs. |

![Tax calculator](images/library-calculator.png)

The invoice editor uses the same data: it shows FBR's unit for the HS code you enter (with **Use** to apply it), offers **Pick SRO / schedule from FBR** and **Pick serial no. from FBR** in the line details, and has **Check with FBR** next to the buyer's NTN/CNIC.

## 16. Sandbox scenarios

![Scenarios](images/scenarios.png)

Used once per company before going live (see [FBR-ONBOARDING-GUIDE.md](FBR-ONBOARDING-GUIDE.md)):

- set the scenarios assigned on IRIS;
- practise them in the simulator;
- run them in the FBR sandbox until all show **passed**.

## 17. Incident register

Rule 150XA(c) and (d) require reporting any operational failure, damage, disruption or tampering of the e-invoicing system — and any inoperative hardware or software, with reasons and evidence — to FBR and the Commissioner **within 24 hours**.

- Recorded automatically:
  - loss of FBR connectivity and token failures;
  - **unexpected stops** of the system itself (crash or power failure longer than 10 minutes) for companies in production. A normal shutdown of the server or Windows service is not reported;
  - **tampering**: every day, and whenever **Verify** is clicked on the Audit trail page, the system checks the audit trail, the seal and digital signature of every accepted invoice, the signing key and the chain of closings. Any change made outside the application opens a tampering incident.
- Record other incidents with **+ Record incident** (hardware/software failure, suspected tampering).
- If an automatically recorded stop was planned (for example the server was switched off without stopping the service), edit the incident and enter a reference such as "Planned shutdown — not reportable" so it no longer shows as unreported.
- **Letter** prints the intimation to the Commissioner, copied to FBR and PRAL, with the list of affected invoices and the software registration number.
- After sending it, edit the incident and enter the date and reference. Incidents not reported within 24 hours are flagged **overdue**.

## 18. Audit trail

Every action is logged with user, time and details: logins, invoices, changes, cancellations, settings. Entries are chained by SHA-256 hashes. **Verify integrity** proves that no entry was altered or removed.

## 19. Using a phone or tablet

Open **Mobile app & access** in the menu. It has a QR code to open the system on your phone, an **Install** button (or instructions for Android and iPhone) and the certificate the phone needs to trust the server. See [MOBILE-AND-REMOTE-ACCESS.md](MOBILE-AND-REMOTE-ACCESS.md).

On a phone, the bar at the bottom of the screen opens **Home**, **Invoices**, a new invoice (the round **+** button), the **FBR data** library and the full **Menu**; the magnifier at the top opens search.

![Mobile app & access](images/mobile-access.png)

<p>
<img src="images/phone-dashboard.png" alt="Dashboard on a phone" width="240">
<img src="images/phone-menu.png" alt="Menu on a phone" width="240">
<img src="images/phone-invoice.png" alt="Accepted invoice on a phone" width="240">
</p>

## 20. Settings

![FBR integration settings](images/fbr-settings.png)

| Page | Purpose |
|---|---|
| Company | Seller particulars, business activity and sector, invoice prefixes, further-tax rate, withholding fraction, validate-before-post, sales tax return due days. **+ Add another company** for multi-company installations. |
| FBR integration | Working environment (training simulator / FBR sandbox / FBR production), sandbox and production tokens with expiry, **Test connection**, reference data download, go-live requirements. **Registration on IRIS** lists the exact technical details to enter on IRIS (system provider, software type, version, business nature, sector), finds this server's public IP for PRAL's IP whitelisting and keeps the **software registration number**. **“Integrated with FBR” signboard** prints the signboard each outlet must display (rule 150R(11)). |
| Invoice printing | Default format, amount-in-words style, columns, copies, terms, footer, company logo |
| Users & roles | Create users, assign roles and companies, reset passwords, deactivate leavers |
| ERP API keys | Keys for ERP/POS integration ([ERP-INTEGRATION-API.md](ERP-INTEGRATION-API.md)) |
| System & licence | Licence status and installation, backups (create/download), FBR Digital Invoicing logo, server information, the **digital signature key** (fingerprint and public key for auditors), **e-mail notifications** |
| Help & error codes | All 108 FBR error codes in FBR's words with fixes, compliance rules, a summary of FBR's FAQs, the official documents and the licensed integrators, sale types and rates |

### Registration on IRIS and the signboard

**Settings → FBR integration** shows the technical details to enter on IRIS, finds this server's public IP for whitelisting, keeps the software registration number and prints the "Integrated with FBR" signboard.

![Registration on IRIS](images/iris-registration.png)

![Integrated with FBR signboard](images/signboard.png)

**Help & error codes → Documents & integrators** links FBR's official notifications and technical documents and lists the licensed integrators.

![Help — documents and integrators](images/help-documents.png)

### E-mail notifications

![E-mail notifications](images/email-notifications.png)

**Settings → System & licence → E-mail notifications** (administrators). The e-mails go through your own mail server; nothing passes through Veridian.

1. Click **Microsoft 365 / Outlook** or **Gmail / Google Workspace** to fill in the server, port and security, or enter your mail server's details.
2. Enter the user name (usually the full e-mail address) and password. Microsoft 365 and Gmail may require an *app password* when two-step sign-in is on. The password is stored encrypted.
3. Enter the sender, and the recipients separated by commas.
4. Choose whether problems are e-mailed as soon as they are found (checked every 5 minutes), and whether to receive a **daily summary** at a chosen hour (yesterday's and this month's invoices and tax, the return due dates and open items).
5. Optionally enter the address of the system (for example `https://192.168.1.10:8443`) so that the e-mails link to it.
6. Tick **Send e-mails** and click **Save & send test e-mail**. The result of the last e-mail is shown below the form.

Each problem is e-mailed at most once a day while it lasts, and again if it recurs after being fixed. Companies working in the training simulator do not raise e-mails about invoices. The server must be able to reach the mail server (usually TCP port 587 or 465).

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
