# FBR Digital Invoicing — Research Summary and Compliance Matrix

This document summarises the Federal Board of Revenue (FBR) Digital Invoicing (DI) regime for sales-tax registered persons in Pakistan. It then maps every requirement to the feature of **Veridian E-invoicing PK** that satisfies it.

It is written for:

- the product owner (a tax practitioner who sells and installs the software);
- client management and auditors;
- implementers.

> **Verify before relying on it.** FBR updates rules frequently through SROs, Sales Tax General Orders (STGOs), Finance Acts and IRIS notices. Items marked **Verify** below should be confirmed against the latest notification on fbr.gov.pk / IRIS before you quote them to a client. This document is not legal advice.

---

## 1. Executive summary

- Every sales-tax registered person brought into the regime must issue a **real-time electronic invoice** for every supply. The invoice is reported to FBR's Digital Invoicing System through an approved integration: PRAL or a licensed integrator.
- FBR returns a unique **FBR invoice number**. The printed invoice must carry this number, a **QR code** encoding it, and the **FBR Digital Invoicing logo**.
- Before going live, each taxpayer passes **sandbox scenarios** assigned on IRIS according to business activity and sector. FBR then issues a **production security token**. Production calls are accepted only from a **whitelisted static IP**.
- After issue, an invoice may be cancelled or edited through the system only within **72 hours** (STGO 01 of 2026). After that, the Commissioner's approval is required.
- Failures, disruptions or tampering of the e-invoicing system must be reported to the Commissioner within **24 hours** (rule 150R). Invoices issued while the connection is down must be **uploaded within 24 hours of connectivity being restored**. Electronic records are kept for **six years** (rule 150S).
- A manufacturer or importer supplying an unregistered person (for example a distributor) must state the buyer's **CNIC or NTN** on the invoice (s.23(1)(b)).
- All active sales-tax filers were required to be live on Digital Invoicing by **31 July 2026** (**Verify** the latest extension for the client's category).
- Penalties for non-integration or for not issuing e-invoices are severe:
  - Rs 1 million, rising to up to Rs 5 million for a continuing default;
  - suspension of registration under s.21(2) for non-compliance with s.23(5)/(6) or s.40C, during which no input tax adjustment or refund is allowed;
  - possible blacklisting.

  These are from the Finance Act 2026 as reported (**Verify** against the enacted text).
- Buyers' input tax depends on it: Annex-A of the buyer's sales-tax return is populated from the supplier's Annex-C, which is fed by DI invoices. An invoice that never reaches FBR is an invoice the customer cannot claim.

Veridian E-invoicing PK is an on-premise system that implements the full lifecycle:

- invoice preparation with Pakistani sales-tax rules;
- validation against FBR's rules before submission;
- real-time reporting through FBR's DI API, using the client's own security token;
- compliant printing;
- reconciliation, debit notes and 72-hour cancellation control;
- incident reporting, audit trail and backups.

## 2. Legal framework

| Instrument | Date | Key provisions | Effect on the business |
|---|---|---|---|
| Sales Tax Act 1990, s.23 | — | Particulars of a tax invoice: supplier and buyer identity and NTN/CNIC, date, description and quantity, value excluding tax, rate and amount of tax, value including tax. **s.23(1)(b):** for supplies by a manufacturer or importer to an unregistered person, the buyer's CNIC or NTN must be stated. Retail supplies to end consumers up to Rs 100,000 are outside the CNIC requirement, which is also waived for card or digital payments (**Verify**). **s.23(5)/(6)** and **s.40C**: the Board may require electronic invoicing integrated with its system. | Every invoice must carry the prescribed particulars and be issued through the integrated system. |
| Sales Tax Act 1990, s.3(1) and s.3(1A) | — | Standard rate 18%. Further tax 4% on taxable supplies to persons not registered; not on exempt or zero-rated supplies; other exclusions apply (**Verify**). | Correct rate and further-tax logic per line. |
| Third, Fifth, Sixth and Eighth Schedules | — | Third Schedule: tax on printed retail price. Fifth: zero-rated. Sixth: exempt. Eighth: reduced rates with SRO/serial references. | Sale type, rate and SRO references must match the schedule. |
| SRO 69(I)/2025 — Chapter XIV, Sales Tax Rules 2006 (rules 150Q onwards) | 29 Jan 2025 | **150R**: install and integrate the e-invoicing system with the Board's system, directly through PRAL or through a licensed integrator; report any failure, disruption or tampering to the Commissioner within 24 hours. **150S**: real-time verifiable invoice for every supply, carrying the FBR invoice number and QR code; keep electronic records for 6 years. **150X**: contraventions punishable under s.33. | Core obligations implemented by this product. |
| SRO 709(I)/2025 and extensions | 2025–2026 | Phased compliance deadlines by category of registered person, extended several times. All active sales-tax filers were to be live by 31 July 2026 (**Verify** the exact phases). | All registered persons should now be live. |
| Offline-invoice rule (Chapter XIV, Sales Tax Rules 2006) | 2025 | Invoices issued while the connection to the Board's system is unavailable must be uploaded within 24 hours of connectivity being restored (**Verify** the rule reference). | Unreported invoices must be cleared quickly after an outage. |
| Sales Tax General Order 01 of 2026 | 31 Mar 2026 | Multiple licensed integrators allowed. An e-invoice may be cancelled or edited through the system within 72 hours of issue; afterwards only with the prior approval of the Commissioner Inland Revenue. | 72-hour control built into cancellation. |
| Finance Act 2026 (as reported) | 2026 | Penalty for failing to integrate or issue e-invoices: Rs 1 million, up to Rs 5 million for a continuing default. Suspension of registration under s.21(2) for non-compliance with s.23(5)/(6) or s.40C; no input tax adjustment or refund during suspension; possible blacklisting. Tier-1 retailers that are not integrated reported to be allowed only 60% of their input tax (**Verify** all). | Strong business case for a reliable system. |
| PRAL DI user manual v1.6 | 16 Apr 2026 | Adds invoice-cancellation screens on IRIS. The DI API remains v1.12; a cancellation service (`cancelinvoicedata`) is reported but its request format has not been published (**Verify**). | Cancel on IRIS, or through the API once PRAL confirms the format (see §5). |
| Sales-tax return Annex-A / Annex-C | — | The buyer's purchase annex (Annex-A) is populated from the supplier's sales annex (Annex-C), which is fed by Digital Invoicing. | Prompt, correct reporting protects customers' input tax. |
| Finance Act 2026 — s.23(1) amendment (as reported) | 2026 | Tax invoices also for exempt supplies, and a new "advance receipt invoice" with a unique FBR number. The Board is to notify the format, who issues it and from when (**Verify**; not yet notified). | Exempt supplies are already invoiced and reported through the "Exempt goods" sale type. The advance receipt document will be added when PRAL publishes it in the DI specification. |
| Draft SRO 288(I)/2026 (income tax) | 2026 (draft) | Proposed "Online Integration of Businesses" rules under the Income Tax Ordinance. This is a separate regime from sales-tax DI and is not yet in force (**Verify**). | Monitor; no product change needed now. |
| Sales Tax Act 1990, s.33 | — | General penalties. | — |

## 3. Integration models

| Model | How it works | Notes |
|---|---|---|
| **PRAL (direct)** | The taxpayer selects PRAL as integrator on IRIS. PRAL issues sandbox and production tokens and whitelists the taxpayer's static IP. The taxpayer's own software calls `gw.fbr.gov.pk`. | Free of charge. **This product uses this model.** |
| **Licensed integrator** | A licensed firm (reported: Haball, EY Ford Rhodes, WebDNAworks — **Verify** the current list) operates the integration for the taxpayer. | Licence granted by the Board. Licensing committee decides within 7 days; licence valid 5 years (**Verify** eligibility criteria). |
| **IRIS web portal** | Manual entry on FBR's portal. | Practical only for very low volumes. |

Veridian E-invoicing PK is the taxpayer's own e-invoicing system. It runs on the client's premises and calls FBR's DI API with the client's own security token, issued through PRAL. It does not route data through any third party.

## 4. Technical specification summary (DI API v1.12)

**Base URL:** `https://gw.fbr.gov.pk`. **Header:** `Authorization: Bearer <security token>`.

| Purpose | Production | Sandbox |
|---|---|---|
| Post invoice | `POST /di_data/v1/di/postinvoicedata` | `POST /di_data/v1/di/postinvoicedata_sb` |
| Validate only (nothing recorded) | `POST /di_data/v1/di/validateinvoicedata` | `POST /di_data/v1/di/validateinvoicedata_sb` |

Reference APIs used by the product:

- `GET /pdi/v1/provinces`, `doctypecode`, `itemdesccode`, `sroitemcode`, `transtypecode`, `uom`, `SroSchedule`;
- `GET /pdi/v2/SaleTypeToRate`, `HS_UOM`, `SROItem`;
- `POST /dist/v1/statl` (Active Taxpayer List);
- `POST /dist/v1/Get_Reg_Type`.

Endpoints are configurable in `config.json` (`fbr.endpoints`) in case FBR changes paths.

### 4.1 Invoice fields and how the product fills them

Payload construction: `BuildPayload` in `internal/service/invoices.go`. JSON model: `internal/fbr/models.go`.

| FBR field | Meaning | Source in the product |
|---|---|---|
| `invoiceType` | "Sale Invoice" or "Debit Note" | Document type chosen in the editor / API `docType` |
| `invoiceDate` | Date of supply, `YYYY-MM-DD` | Invoice date (Pakistan time). Future dates and stale back-dating are flagged by validation. |
| `sellerNTNCNIC`, `sellerBusinessName`, `sellerProvince`, `sellerAddress` | Seller particulars | Company profile (Settings → Company), snapshotted on the invoice |
| `buyerNTNCNIC`, `buyerBusinessName`, `buyerProvince`, `buyerAddress`, `buyerRegistrationType` | Buyer particulars | Customer master or ad-hoc buyer. NTN normalised to 7 digits, or a 13-digit CNIC. |
| `invoiceRefNo` | Debit notes: FBR number of the original invoice | Set automatically when a debit note is raised from an accepted invoice. Optionally our internal number on sale invoices (`sendInternalRef`). |
| `scenarioId` | Sandbox scenario (SN001–SN028) | Sent **only** in the sandbox; stripped in production (`TestScenarioIdStrippedInProduction`). |
| `items[].hsCode` | PCT code `NNNN.NNNN` | Product master / line entry, searchable HS list |
| `items[].productDescription` | Description | Line description |
| `items[].rate` | e.g. `18%`, `Exempt`, `Rs.200`, `18% along with rupees 60 per kilogram` | Sale type default or FBR `SaleTypeToRate` list; parsed by `internal/tax/rate.go` |
| `items[].uoM` | FBR unit of measure | UoM list; checked against `HS_UOM` |
| `items[].quantity` | Quantity (up to 4 decimals) | Line quantity |
| `items[].valueSalesExcludingST` | Value of supply excluding sales tax (s.2(46)) | Quantity × price − discount (or ERP override), plus FED charged separately |
| `items[].fixedNotifiedValueOrRetailPrice` | Retail price basis (Third Schedule), excluding sales tax (s.2(27)) | Printed retail price (which includes sales tax, s.3(2)(a)) × quantity × 100 ÷ (100 + rate) |
| `items[].salesTaxApplicable` | Sales tax | Tax engine: base × rate + quantity × per-unit rate |
| `items[].salesTaxWithheldAtSource` | Tax withheld by a withholding agent | Customer withholding mode (fraction, default 1/5, or full) |
| `items[].extraTax` | Extra tax | Sent as `""` where FBR requires it empty (reduced-rate goods, error 0091) |
| `items[].furtherTax` | Further tax (4%) | Automatic for unregistered buyers on taxable supplies |
| `items[].fedPayable` | Federal excise duty on the line | FED rate or amount per product; included in the value of supply above (except for the "FED in ST mode" sale types) |
| `items[].discount` | Discount | Amount or percentage per line |
| `items[].saleType` | FBR sale type text | 27 sale types with FBR's exact wording (`internal/domain/domain.go`) |
| `items[].sroScheduleNo`, `items[].sroItemSerialNo` | SRO / schedule references | Product master or line entry; required for SRO-based sale types |
| `items[].totalValues` | Line total | Value of supply (including FED) + sales tax + further tax + extra tax |

FBR's response contains:

- `invoiceNumber`, in the format `<seller NTN/CNIC>DI<milliseconds>`, e.g. `7000007DI1747119701593`;
- `dated`;
- `validationResponse` with `statusCode` ("00" valid / "01" invalid), error codes, and per-item `invoiceNo` values (`<invoiceNumber>-<n>`).

### 4.2 Printed invoice

The printed invoice carries:

- the FBR invoice number;
- a **QR code of version 2.0 (25×25 modules) printed at 1 × 1 inch** encoding the FBR invoice number;
- the **FBR Digital Invoicing System logo**: upload the official image under Settings → System; until then a text badge is printed.

Drafts and unreported invoices print with a "DRAFT — NOT REPORTED TO FBR" watermark and no QR code. Sandbox and training invoices are watermarked as not valid tax invoices.

### 4.3 Sandbox scenarios

Twenty-eight scenarios (SN001–SN028) are defined in `internal/domain/scenarios.go` with FBR's sample data. The scenarios page suggests the applicable set from the company's business activity and sector (`ApplicableScenarios`). It can run each scenario in the FBR sandbox (certification) or in the built-in simulator (practice).

### 4.4 Common FBR error codes

The product contains a catalogue of about 60 FBR error codes with plain-language fixes (`internal/validate/catalog.go`, shown under Help & error codes and next to every rejection). Frequent ones:

| Code | Meaning | Typical fix |
|---|---|---|
| 0401 | Unauthorised — token invalid, expired or not issued for this seller NTN | Re-enter the correct token for the environment; check the seller NTN matches the token |
| 0002 | Buyer NTN/CNIC invalid | Use a 7-digit NTN (no check digit) or a 13-digit CNIC |
| 0046 | Rate not allowed for the sale type on this date | Pick a rate from FBR's SaleTypeToRate list |
| 0052 | HS code invalid or not matching the sale type | Correct the PCT code |
| 0077 / 0078 | SRO schedule / item serial missing or invalid | Enter the SRO schedule and serial number for reduced-rate or exempt items |
| 0091 | Extra tax must be empty for reduced-rate goods | The product sends `""` automatically |

## 5. Operational obligations

| Obligation | Rule | How the product helps |
|---|---|---|
| Report each invoice in real time | 150S | "Save & submit" reports immediately. If FBR is unreachable the invoice is queued and retried automatically: backoff 30 s doubling to a 30-minute cap; 15 minutes after a token failure. |
| Upload offline invoices within 24 hours of restoration | Chapter XIV | As soon as any call to FBR succeeds again, every queued invoice of that company is resubmitted at once rather than waiting for its back-off. The dashboard shows how many invoices are not yet reported and the oldest one. A queued invoice can be handed over as a **provisional copy** watermarked "PENDING FBR REPORTING"; the final copy with the FBR number and QR code is printed after acceptance. |
| CNIC/NTN of unregistered buyers | s.23(1)(b) | Validation warns when a manufacturer or importer (business activity in Settings → Company) invoices an unregistered buyer without a CNIC/NTN, and for other sellers above Rs 100,000 (`fbr.cnicThreshold`). |
| Do not report twice | Good practice / integrity | Timeouts after sending are marked **Needs reconciliation** and never resent blindly. The user checks IRIS and records the outcome. |
| Cancel or edit only within 72 hours | STGO 01/2026 | Cancellation records the IRIS reference. After 72 hours it requires the Commissioner's approval reference. Once PRAL publishes its cancellation service, set `fbr.endpoints.cancelPath` and `cancelSandboxPath` in `config.json` (reported paths: `/di_data/v1/di/cancelinvoicedata` and `/di_data/v1/di/cancelinvoicedata_sb`) and confirm that the request format matches. The cancel dialog then offers **Cancel with FBR**. The cancellation is recorded only if FBR confirms it, and a refusal or unreadable reply leaves the invoice unchanged. |
| Report failures within 24 hours | 150R | FBR outages, token failures, unexpected stops of the system (crash or power failure) and tampering automatically open incidents. Tampering is found by a daily check of the audit chain and of every accepted invoice's seal. A ready-to-print letter to the Commissioner lists the affected invoices. |
| Keep records for 6 years | 150S | Accepted invoices cannot be edited or deleted (database triggers). Nightly backups are written to the backups folder; off-site copying is the client's responsibility. |

## 6. Compliance matrix

Status: **Implemented**, **Partial** (implemented with a manual step), or **Client** (the client's responsibility).

| # | Requirement | Source | How the product meets it | Where | Evidence (tests) | Status |
|---|---|---|---|---|---|---|
| 1 | Supplier name, address, NTN/CNIC on invoice | s.23 | Company profile snapshotted on every invoice; printed | `internal/service/invoices.go` (Build); Settings → Company; `internal/printing/templates/invoice_a4.html` | TestRenderA4 | Implemented |
| 2 | Buyer name, address, NTN/CNIC, registration type; CNIC/NTN of unregistered buyers of manufacturers and importers | s.23, s.23(1)(b); DI spec | Customer master or ad-hoc buyer; validation of NTN/CNIC format; registered buyer must have NTN/CNIC; s.23(1)(b) warning by seller activity; warning above Rs 100,000 for others | `internal/validate/validate.go`; Customers page | TestRegisteredBuyerNeedsNTN, TestNormalizeRegNo | Implemented |
| 3 | Date of issue | s.23; DI spec | Invoice date in Pakistan time; future dates rejected; sandbox scenario rules | `internal/validate/validate.go` | TestFutureDateAndSandboxScenario | Implemented |
| 4 | Description, HS code, quantity, UoM | s.23; DI spec | Product master with HS search and UoM list; HS_UOM check | Products page; `internal/service/refdata.go` | TestValidPayload, TestHeaderErrors | Implemented |
| 5 | Value excluding tax, rate, sales tax, value including tax | s.23 | Exact decimal tax engine, 2-decimal rounding, per-unit and mixed rates | `internal/tax/engine.go`, `rate.go` | TestParseRate, TestSumLines, TestScenarioArithmetic | Implemented |
| 6 | Further tax 4% to unregistered buyers, not on exempt or zero-rated | s.3(1A) | Automatic per line; overridable per product (auto/yes/no) | `internal/tax/engine.go` | TestStandardRateUnregisteredFurtherTax, TestFurtherTaxNotOnExemptOrThirdSchedule | Implemented |
| 7 | Third Schedule: tax on the retail price excluding sales tax (s.2(27)); printed price includes sales tax (s.3(2)(a)) | Third Schedule | Tax base = printed price × quantity × 100 ÷ (100 + rate); required for these sale types | `internal/tax/engine.go`; `internal/validate/validate.go` | TestThirdScheduleTaxInclusivePrintedPrice, TestThirdScheduleNeedsRetailPrice | Implemented |
| 7a | FED part of the value of supply | s.2(46)(a) | Sales tax and further tax charged on value + FED (except "FED in ST mode" sale types); FED not added to the total twice | `internal/tax/engine.go` | TestFEDIncludedInValueOfSupply | Implemented |
| 8 | Reduced rate with SRO; extraTax empty | Eighth Schedule; error 0091 | SRO fields required; `extraTax` sent as "" | `internal/domain/domain.go`; `internal/fbr/models.go` | TestReducedRateExtraTaxEmpty, TestReducedRateRules | Implemented |
| 9 | Exempt / zero-rated supplies | Fifth/Sixth Schedules | Exempt rate handling; no further tax | `internal/tax/rate.go` | TestExemptRate | Implemented |
| 10 | Withholding by withholding agents | Withholding rules | Customer withholding mode: fraction (default 1/5) or full; amount payable reduced | `internal/tax/engine.go` | TestWithholdingAndRounding | Implemented |
| 11 | Every DI payload field in FBR's format | DI v1.12 | Field order and names per spec; amounts as numbers, empty strings where required | `internal/fbr/models.go` | TestAmountJSON, TestPostInvoiceSuccess | Implemented |
| 12 | Real-time reporting and FBR invoice number | 150S | Submit on save; number stored per invoice and per line | `internal/service/submit.go` | TestInvoiceLifecycleSimulator | Implemented |
| 13 | Validate before posting | Good practice | Optional validateinvoicedata call before post (on by default) | `internal/service/submit.go` | TestValidateDoesNotRecord | Implemented |
| 14 | Rejections shown with fixes | DI error codes | Per-line FBR errors with catalogue fixes; invoice stays editable | InvoiceView page; `internal/validate/catalog.go` | TestRejectionFromFBR, TestRejectionIsNotAnError, TestCatalogue | Implemented |
| 15 | No duplicate reporting after network failures | Integrity | Failure classification: not sent → retry; ambiguous → Needs reconciliation | `internal/fbr/client.go`; `internal/service/submit.go` | TestTimeoutBecomesUncertainAndIsReconciled, TestTimeoutAfterRecordingIsUncertain, TestDroppedConnectionIsUncertain, TestConnectionRefusedIsNotSent | Implemented |
| 16 | Continuity during FBR outages | 150R/150S | Queue with automatic retry; outage incident opened | `internal/service/worker.go`, `health.go` | TestOutageQueuesAndWorkerResubmits, TestServiceUnavailableIsRetryable | Implemented |
| 16a | Offline invoices uploaded within 24 hours of restoration | Chapter XIV | Immediate resubmission on recovery; invoices issued offline stay on the dashboard's "not yet reported" count, with their age, until FBR accepts them (also when the upload after recovery is rejected); provisional printout until then; queued uploads continue after licence expiry and never block other companies' queues | `internal/service/health.go`, `worker.go`, `submit.go`; Dashboard; `internal/printing/render.go` | TestRecoveryResubmitsQueuedAtOnce, TestOfflineInvoiceTrackedUntilAccepted, TestRefusedQueuedInvoiceIsDeferred, TestRenderWatermarks | Implemented |
| 17 | Sandbox scenarios before production | IRIS onboarding | 28 scenarios with FBR samples; sandbox and practice runs; progress tracking | Scenarios page; `internal/service/scenarios.go` | TestScenariosInSandbox | Implemented |
| 18 | scenarioId only in sandbox | DI v1.12 | Stripped outside the sandbox | `internal/fbr/client.go` | TestScenarioIdStrippedInProduction | Implemented |
| 19 | Production token and IP whitelisting | IRIS/PRAL | Encrypted token storage; production switch refused without a token; connection test | Settings → FBR integration; `internal/service/masters.go` | TestProductionRequiresToken, TestUnauthorized | Partial (IP whitelisting is arranged with PRAL) |
| 20 | QR code v2, 25×25, 1 inch, encoding FBR number | FBR printing rules | Forced version 2; SVG sized 1in × 1in | `internal/printing/qr.go` | TestQRVersion2 | Implemented |
| 21 | FBR DI logo on invoice | FBR printing rules | Upload official logo once; text badge fallback | Settings → System; templates | TestRenderA4 | Partial (client uploads official logo) |
| 22 | Reported invoices cannot be altered | 150S integrity | Database triggers block updates and deletes of accepted invoices, items and audit log; tamper-evident seal | `internal/db/migrations/0001_init.sql`; `internal/service/invoices.go` (seal) | TestMigrateAndTriggers | Implemented |
| 23 | Returns and reductions by debit note | s.9 Sales Tax Act / Chapter IV Sales Tax Rules (**Verify**); DI spec (errors 0006, 0035, 0036, 0067) | Debit note raised from an accepted invoice with `invoiceRefNo`; date not before the original; value and tax capped at the original; netted off output tax in reports | InvoiceView → Debit note; `internal/validate/validate.go` | TestDebitNote | Implemented |
| 24 | Cancel/edit only within 72 hours; Commissioner approval later | STGO 01/2026; PRAL manual v1.6 | The 72 hours run from FBR's issue time ("dated"); cancellation requires a reason and IRIS reference; after 72 h a Commissioner approval reference; optional cancellation through FBR's service, recorded only on FBR's confirmation | `internal/service/invoices.go` (CancelInvoice); `internal/fbr/client.go` (CancelOutcome) | TestCancelThroughFBRAPI, TestCancelOutcome | Partial (cancel on IRIS until PRAL publishes the API format) |
| 25 | Report failures, disruptions and tampering within 24 hours | 150R | Auto-detected incidents for FBR outages, token failures, unexpected stops (heartbeat with orderly-shutdown marker) and tampering (daily audit-chain and seal check); incident register; printable letter | Incident register; `internal/service/health.go`, `integrity.go`; `internal/httpapi/h_admin.go` | TestIntegrityCheckDetectsTampering, TestUnexpectedStopIsRecorded | Partial (the client sends the letter) |
| 26 | Records for 6 years | 150S | Nightly backups with retention; manual backup; download | Settings → System; `internal/service/worker.go` | — | Partial (off-site copies are the client's job) |
| 27 | Audit trail | Good practice | Append-only, SHA-256 hash-chained log with verification | Audit trail page; `internal/store/audit.go` | TestMigrateAndTriggers | Implemented |
| 28 | Access control | Good practice | Roles: admin, manager, accountant, operator, auditor; sessions + CSRF; hashed API keys; lockout after failed logins | `internal/httpapi/server.go`, `perms.go`; `internal/store/users.go` | TestRoutesRegister | Implemented |
| 29 | Protection of FBR tokens | Good practice | AES-256-GCM encryption with `master.key`; tokens never returned by the API or logged | `internal/security/security.go`; `internal/fbr/client.go` | TestProductionRequiresToken | Implemented |
| 30 | Buyer status checks | Good practice | Active Taxpayer List and registration-type lookups | Customers → Check FBR | TestReferenceAPIs | Implemented |

## 7. Gaps and client responsibilities

- **IP whitelisting and tokens:** the client (or the practitioner on their behalf) requests these on IRIS / from PRAL.
- **Official FBR DI logo:** obtain it from FBR/PRAL and upload it once under Settings → System.
- **Off-site backups:** copy the `backups` folder and `master.key` to separate storage regularly, and keep records for six years.
- **Cancellation on IRIS:** FBR's public DI API v1.12 does not define a cancellation call, and PRAL has not published the format of the reported `cancelinvoicedata` service. Until it does, cancel on IRIS within 72 hours, then record the cancellation in the product. When the format is confirmed, enable the API option in `config.json` (§5).
- **After an outage:** check the dashboard once the connection is back, and make sure the "not yet reported" count reaches zero within 24 hours. File the Rule 150R letter for the incident.
- **Rule 150R letters:** the product prepares the letter; the client signs and sends it within 24 hours.
- **Legal updates:** rates, SROs and scenario lists change. Keep the product updated under a support contract and re-check the items marked **Verify**.

## 8. Sources and items to verify

- SRO 69(I)/2025 (29 January 2025) — Chapter XIV, Sales Tax Rules 2006.
- SRO 709(I)/2025 and subsequent extensions — compliance phases (**Verify**).
- Sales Tax General Order 01 of 2026 (31 March 2026) — integrators; 72-hour rule.
- Finance Act 2026 — penalty amounts, s.21(2) suspension, Tier-1 retailer input-tax restriction (**Verify** against the enacted text).
- Sales Tax Act 1990, s.23(1)(b), s.23(5)/(6), s.40C — invoice particulars, CNIC requirement, electronic invoicing.
- PRAL Digital Invoicing user manual v1.6 (16 April 2026) — IRIS cancellation screens.
- Draft SRO 288(I)/2026 — income-tax online integration (draft).
- PRAL Digital Invoicing technical documentation v1.12 — APIs, fields, scenarios, error codes, QR specification.
- IRIS → Digital Invoicing — onboarding screens, scenario assignment, token issue.

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing PK is proprietary software.
