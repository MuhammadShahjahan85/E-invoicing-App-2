# FBR Digital Invoicing — Research Summary and Compliance Matrix

This document summarises the Federal Board of Revenue (FBR) Digital Invoicing (DI) regime for sales-tax registered persons in Pakistan. It then maps every requirement to the feature of **Veridian E-invoicing Pakistan** that satisfies it.

It is written for:

- the product owner (a tax practitioner who sells and installs the software);
- client management and auditors;
- implementers.

> **Status: reviewed against FBR's website on 9 October 2026.** The legal sources in §2 and §8 were read from the official copies on fbr.gov.pk (links in §8). FBR still updates the rules often through SROs, Sales Tax General Orders (STGOs), Finance Acts and IRIS notices, so check for newer notifications before you quote a point to a client. Items still marked **Verify** could not be confirmed from an official source. This document is not legal advice.

---

## 1. Executive summary

- Every sales-tax registered person brought into the regime must issue a **real-time electronic invoice** for every supply. The invoice is reported to FBR's Digital Invoicing System through an approved integration: PRAL or a licensed integrator.
- FBR returns a unique **FBR invoice number**. The printed invoice must carry this number, a **QR code** encoding it, and the **FBR Digital Invoicing logo**.
- Before going live, each taxpayer passes **sandbox scenarios** assigned on IRIS according to business activity and sector. FBR then issues a **production security token**. Production calls are accepted only from a **whitelisted static IP**.
- After issue, an invoice may be cancelled or edited through the system only within **72 hours** (STGO 01 of 2026). After that, the Commissioner's approval is required.
- Operational failures, damage, disruption or tampering of the e-invoicing system must be reported to **FBR and the Commissioner within 24 hours** (rule 150XA(c)); inoperative hardware or software within 24 hours, with reasons and evidence (rule 150XA(d)). Invoices issued during a failure of the invoicing software, the internet or power must be **identified as issued in the offline mode and uploaded within 24 hours of restoration** (rule 150XC). Electronic records are kept for **six years** (rule 150S).
- The invoicing system itself must issue invoices in the prescribed format, **create and record a digital signature** on each invoice, transmit and preserve the data securely, print the QR code, **perform closings at the close of each day, week and month** and **log every adjustment, cancellation and system event** (rule 150R(4)). Each notified outlet displays an **"Integrated with FBR" signboard** with the software registration number (rule 150R(11)). The invoice carries the particulars of rule 150R(13) — since **SRO 1666(I)/2026** (29 September 2026) including six federal excise duty particulars.
- A manufacturer or importer supplying an unregistered person (for example a distributor) must state the buyer's **CNIC or NTN** on the invoice (s.23(1)(b)).
- Every registered person should now be live: under **SRO 1852(I)/2025** the last dates for issuing electronic invoices ran from **1 November 2025** (public companies, importers, companies with turnover above Rs 1 billion, individuals and AOPs above Rs 100 million) to **31 December 2025** (all other registered persons).
- Since the **Finance Act 2026** (explained in FBR's Circular 01 of 2026, 11 September 2026):
  - an invoice bearing an FBR number is required for **exempt supplies and advance receipts** as well (s.23(1));
  - failure to integrate or record sales costs **Rs 1 million**, and a second penalty of **up to Rs 5 million** if the default continues a month after the first; the premises may be sealed (s.33(25));
  - registration may be **suspended** for failure to integrate (s.21(2));
  - the Board may reduce the **input tax ratio** of persons not complying with digital invoicing and other electronic systems (s.8B);
  - issuers of simulated invoices are listed on a public **Simulated Invoice Issuers Register**, and buyers lose the input tax on their invoices (s.33(29)–(31)).
- Goods moved from a factory to the registered person's **own warehouse under the same STRN** are not a supply: no digital invoice is issued, but each consignment travels with a **stock transfer note** in the prescribed format (STGO 25 of 2026).
- An invoice reported with an FBR number but not accounted for in **Annex-C** or the return is taxed by the Officer of Inland Revenue unless it was cancelled through the approved mechanism (rule 150XD(2), as substituted by SRO 1666(I)/2026).
- Buyers' input tax depends on it: Annex-A of the buyer's sales-tax return is populated from the supplier's Annex-C, which is fed by DI invoices. An invoice that never reaches FBR is an invoice the customer cannot claim.

Veridian E-invoicing Pakistan is an on-premise system that implements the full lifecycle:

- invoice preparation with Pakistani sales-tax rules;
- validation against FBR's rules before submission;
- real-time reporting through FBR's DI API, using the client's own security token;
- compliant printing;
- reconciliation, debit notes and 72-hour cancellation control;
- the system functions of rule 150R(4): digital signature, day/week/month closings and complete logs;
- the signboard, stock transfer notes, advance receipt invoices and federal excise duty particulars;
- incident reporting, Annex-C reconciliation, audit trail and backups.

## 2. Legal framework

| Instrument | Date | Key provisions | Effect on the business |
|---|---|---|---|
| Sales Tax Act 1990, s.23 | — | Particulars of a tax invoice: supplier and buyer identity and NTN/CNIC, date, description and quantity, value excluding tax, rate and amount of tax, value including tax. **s.23(1)(b):** for supplies by a manufacturer or importer to an unregistered person, the buyer's CNIC or NTN must be stated. Retail supplies to end consumers up to Rs 100,000 are outside the CNIC requirement, which is also waived for card or digital payments (**Verify**). **s.23(5)/(6)** and **s.40C**: the Board may require electronic invoicing integrated with its system. | Every invoice must carry the prescribed particulars and be issued through the integrated system. |
| Sales Tax Act 1990, s.3(1) and s.3(1A) | — | Standard rate 18%. Further tax 4% on taxable supplies to persons not registered; not on exempt or zero-rated supplies; other exclusions apply (**Verify**). | Correct rate and further-tax logic per line. |
| Third, Fifth, Sixth and Eighth Schedules | — | Third Schedule: tax on printed retail price. Fifth: zero-rated. Sixth: exempt. Eighth: reduced rates with SRO/serial references. | Sale type, rate and SRO references must match the schedule. |
| SRO 69(I)/2025 — Chapter XIV, Sales Tax Rules 2006 (rules 150Q–150XQ) | 29 Jan 2025 | **150R(1)–(3)**: register, install and integrate the invoicing system with the Board's system; supplies only through integrated outlets. **150R(4)**: the system must (a) generate, record and store invoice data, (b) issue invoices in the prescribed format and create and record a **digital signature**, (c) transmit securely and receive the FBR number, (d) preserve the data irrevocably, (e) print the QR code, (f) perform **closings at the close of the day, week and month**, (g) log every adjustment, modification, cancellation and system event. **150R(5)**: Annexure-C auto-filled from e-invoices. **150R(6)**: alert messages and a log of malpractice or errors. **150R(9)**: exempt supplies also invoiced electronically. **150R(11)**: **"Integrated with FBR" signboard** with the software registration number. **150R(13)**: invoice particulars (a)–(z). **150S**: real-time verifiable invoice for every supply; records for **6 years**. **150V**: the Commissioner may extend the integration date. **150X**: contraventions punishable under s.33. **150XA(c)/(d)**: report failures, damage, disruption, tampering or inoperative systems to FBR and the Commissioner **within 24 hours**. **150XB**: buyers can verify invoices on FBR's website. **150XC**: offline invoices identified as such and **uploaded within 24 hours of restoration**. **150XD**: monitoring; tax on unaccounted invoices. **150XE–150XQ**: licensed integrators; under **150XF** PRAL acts as a licensed integrator free of cost. | Core obligations implemented by this product (see §6). |
| SRO 709(I)/2025 → SRO 1413(I)/2025 → **SRO 1852(I)/2025** | 22 Apr 2025 → 1 Aug 2025 → 24 Sep 2025 | Integration made mandatory for all registered persons; SRO 1852(I)/2025 superseded the earlier notifications and set the final dates for registration, testing and issuing e-invoices: **1 Nov 2025** (public companies; companies with turnover above Rs 1 billion; importers; individuals and AOPs above Rs 100 million), **15 Nov 2025** (companies Rs 100 million – 1 billion), **1 Dec 2025** (companies up to Rs 100 million), **31 Dec 2025** (all others). | All registered persons should now be live. |
| Sales Tax General Order 01 of 2026 | 30 Mar 2026 | A registered person may engage one or more licensed integrators. A valid e-invoice generated by bona fide mistake may be cancelled, deleted or edited through the Board's system only within **72 hours** of its generation; afterwards only with the prior approval of the Commissioner Inland Revenue. | 72-hour control built into cancellation. |
| Finance Act 2026, explained in **Circular 01 of 2026** | Act: Jun 2026; circular: 11 Sep 2026 | **s.23(1)**: invoices also for exempt supplies and **advance receipts**, bearing a verifiable FBR number; "advance receipt invoice" defined in s.2(1AA) as an invoice in the format the Board notifies. **s.33(25)**: Rs 1 million for failure to integrate or record sales, second penalty up to Rs 5 million if the default continues a month later; premises may be sealed. **s.21(2)**: suspension extended to failure to integrate. **s.8B**: input tax ratio may be reduced for non-compliance with digital systems. **s.33(29)–(31)**: Simulated Invoice Issuers Register; 20% penalties on unmatched or unreversed input tax. **s.9**: the Board may prescribe an electronic mechanism for debit and credit notes. | Strong business case for a reliable system; advance receipt invoices supported (§5). |
| **SRO 1655(I)/2026** — Chapter XII-A (electronic scrutiny) | 25 Sep 2026 | FBR's system cross-matches returns with other data and sends **intimations through IRIS**, allowing at least seven days to explain or correct before action. | The Annex-C reconciliation and sales register answer such intimations. |
| **Sales Tax General Order 25 of 2026** | 28 Sep 2026 | Movement from a factory to the registered person's own warehouse under the same STRN is not a supply and needs no digital invoice, but must travel with a prescribed **stock transfer note** (Annexure-A format), be acknowledged by the warehouse, reconciled and kept for six years. A warehouse with a separate STRN receives a taxable supply. | Stock transfer notes module (§5). |
| **SRO 1666(I)/2026** — amendments to Chapter XIV | 29 Sep 2026 | Chapter XIV also applies to **federal excise duty** and Islamabad Capital Territory services. Rule 150R(13) gains the FED particulars **(aa)** FED type, **(bb)** rate, **(cc)** price per unit, **(dd)** amount payable otherwise than in sales tax mode, **(ee)** SRO/Schedule reference and **(ff)** serial number. Rule 150S(2): debit notes, credit notes and **advance receipt invoices** issued electronically and kept six years. Rule 150XD(2): tax recovered on invoices transmitted with an FBR number but not accounted for in **Annex-C** or the return. | FED particulars printed; Annex-C reconciliation export. |
| PRAL DI user manual | v1.5 (linked on FBR's site, Oct 2026); v1.6 reported 16 Apr 2026 | Registration on IRIS: technical details (ERP/system provider, software type, version, CRM user), IP whitelisting (1–3 addresses, approved by PRAL within about two working hours), sandbox and production. v1.6 is reported to add invoice-cancellation screens; the DI API remains v1.12 and the request format of the reported `cancelinvoicedata` service has not been published (**Verify**). | IRIS details shown in the product; cancel on IRIS, or through the API once PRAL confirms the format (see §5). |
| Sales-tax return Annex-A / Annex-C | — | The buyer's purchase annex (Annex-A) is populated from the supplier's sales annex (Annex-C), which is fed by Digital Invoicing (rule 150R(5)). | Prompt, correct reporting protects customers' input tax. |
| Draft SRO 288(I)/2026 (income tax) | 18 Feb 2026 (draft) | Proposed "Online Integration of Businesses" rules under the Income Tax Ordinance. This is a separate regime from sales-tax DI and is not yet in force (**Verify**). | Monitor; no product change needed now. |
| Sales Tax Act 1990, s.33 | — | General penalties. | — |

## 3. Integration models

| Model | How it works | Notes |
|---|---|---|
| **PRAL (direct)** | The taxpayer selects PRAL as integrator on IRIS. PRAL issues sandbox and production tokens and whitelists the taxpayer's static IP. The taxpayer's own software calls `gw.fbr.gov.pk`. | Free of charge. **This product uses this model.** |
| **Licensed integrator** | A firm licensed by the Board operates the integration for the taxpayer. FBR's list (October 2026): Haball (Pvt) Ltd, WebDNAworks (Pvt) Ltd, EY Ford Rhodes, PRAL, OpenPort Pakistan (Pvt) Ltd, TMR Consulting (Pvt) Ltd, NatureTech (Pvt) Ltd and Dynamic Resources (Pvt) Ltd. | Licensing under rules 150XE–150XQ; fees within the limit the Board specifies. The list is shown under Help → Documents & integrators. |
| **IRIS web portal** | Manual entry on FBR's portal. | Practical only for very low volumes. |

Veridian E-invoicing Pakistan is the taxpayer's own e-invoicing system. It runs on the client's premises and calls FBR's DI API with the client's own security token, issued through PRAL as the licensed integrator (FBR's FAQ 12: only a licensed integrator configures a registered person's software for transmission to FBR; under rule 150XF PRAL does so free of cost). It does not route data through any third party.

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
| `items[].fedPayable` | Federal excise duty on the line | FED rate or amount per product; included in the value of supply above (except for the "FED in ST mode" sale types). The FED particulars of rule 150R(13)(aa)–(ff) have no field in the API and are printed on the invoice. |
| `items[].discount` | Discount | Amount or percentage per line |
| `items[].saleType` | FBR sale type text | 27 sale types with FBR's exact wording (`internal/domain/domain.go`) |
| `items[].sroScheduleNo`, `items[].sroItemSerialNo` | SRO / schedule references | Product master or line entry; required for SRO-based sale types |
| `items[].totalValues` | Line total | Value of supply (including FED) + sales tax + further tax + extra tax |

FBR's response contains:

- `invoiceNumber`, in the format `<seller NTN/CNIC>DI<milliseconds>`, e.g. `7000007DI1747119701593`;
- `dated`;
- `validationResponse` with `statusCode` ("00" valid / "01" invalid), error codes, and per-item `invoiceNo` values (`<invoiceNumber>-<n>`).

### 4.2 Printed invoice

The printed invoice carries every particular of rule 150R(13) and section 6 of the DI specification:

- the FBR invoice number and FBR's date and time;
- a **QR code of version 2.0 (25×25 modules) printed at 1 × 1 inch** encoding the FBR invoice number (the DI specification's size; rule 150R(13)(b) gives 7 × 7 mm, which the 1-inch code exceeds);
- the official **FBR Digital Invoicing System logo** from the DI specification, built in (a newer image can be uploaded under Settings → System);
- the **software registration number** (150R(13)(c)) and the **tax period** (150R(13)(l));
- seller and buyer particulars, description, **HS code** (always printed), quantity, unit, value excluding tax, rate, sales tax, sales tax withheld, extra tax, further tax, FED in sales tax mode, **total discount**, invoice reference and SRO/serial (150R(13)(e)–(z));
- the **federal excise duty particulars** (aa)–(ff) under each line subject to FED (SRO 1666(I)/2026);
- the **digital signature** and the signing key's fingerprint (150R(4)(b));
- for invoices issued while FBR could not be reached, the note "Issued in offline mode … reported to FBR on …" (150XC);
- for advance receipt invoices, the title "ADVANCE RECEIPT INVOICE" and an explanatory note; on final invoices, the advance receipt invoices adjusted.

Drafts and unreported invoices print with a "DRAFT — NOT REPORTED TO FBR" watermark and no QR code. Sandbox and training invoices are watermarked as not valid tax invoices.

### 4.3 Sandbox scenarios

Twenty-eight scenarios (SN001–SN028) are defined in `internal/domain/scenarios.go` with FBR's sample data. The scenarios page suggests the applicable set from the company's business activity and sector, using the business activity × sector matrix of section 10 of the DI specification v1.12, transcribed in full (`ApplicableScenarios`, `TestScenarioMatrixComplete`). It can run each scenario in the FBR sandbox (certification) or in the built-in simulator (practice).

### 4.4 Common FBR error codes

The product contains all **108 error codes** of the DI specification v1.12 — 86 for sales (0001–0113, 0300, 0401, 0402) and 22 for purchases (0156–0177) — with FBR's own wording and a plain-language fix (`internal/validate/catalog.go`, generated from the specification; shown under Help & error codes and next to every rejection). The local checks cite the same codes FBR would return (`TestLocalChecksCiteDocumentedCodes`). Frequent ones:

| Code | FBR's message | Typical fix |
|---|---|---|
| 0401 | The provided seller NTN/CNIC does not have a valid or authorized access token | Re-enter the correct token for the environment; check the seller NTN matches the token |
| 0002 | Invalid Buyer Registration No or NTN | Use a 7- or 9-digit NTN or a 13-digit CNIC |
| 0053 | Provided buyer registration type is invalid | Registered or Unregistered |
| 0052 | Please provide valid HS Code against invoice no | Use an HS code that matches the sale type |
| 0077 / 0078 | Provide SRO/Schedule No. / Provide Item Sr. No. | Enter the SRO schedule and serial number for reduced-rate or exempt items |
| 0091 | Extra tax must be empty. | The product sends `""` automatically |
| 0104 | The calculated percentage sales tax does not match. | Let the tax engine compute the tax, or correct the override |
| 0026 / 0057 | Invoice Reference No. is required. / Reference Invoice does not exist. | A debit note must quote the FBR number of an accepted invoice (22 characters for an NTN seller, 28 for a CNIC seller) |

## 5. Operational obligations

| Obligation | Rule | How the product helps |
|---|---|---|
| Report each invoice in real time | 150S | "Save & submit" reports immediately. If FBR is unreachable the invoice is queued and retried automatically: backoff 30 s doubling to a 30-minute cap; 15 minutes after a token failure. |
| Upload offline invoices within 24 hours of restoration, identified as offline | 150XC | As soon as any call to FBR succeeds again, every queued invoice of that company is resubmitted at once rather than waiting for its back-off. The dashboard shows how many invoices are not yet reported and the oldest one. A queued invoice can be handed over as a **provisional copy** watermarked "PENDING FBR REPORTING"; the final copy with the FBR number and QR code is printed after acceptance and states that it was issued in offline mode and when FBR received it. |
| CNIC/NTN of unregistered buyers | s.23(1)(b) | Validation warns when a manufacturer or importer (business activity in Settings → Company) invoices an unregistered buyer without a CNIC/NTN, and for other sellers above Rs 100,000 (`fbr.cnicThreshold`). |
| Do not report twice | Good practice / integrity | Timeouts after sending are marked **Needs reconciliation** and never resent blindly. The user checks IRIS and records the outcome. |
| Cancel or edit only within 72 hours | STGO 01/2026 | Cancellation records the IRIS reference. After 72 hours it requires the Commissioner's approval reference. Once PRAL publishes its cancellation service, set `fbr.endpoints.cancelPath` and `cancelSandboxPath` in `config.json` (reported paths: `/di_data/v1/di/cancelinvoicedata` and `/di_data/v1/di/cancelinvoicedata_sb`) and confirm that the request format matches. The cancel dialog then offers **Cancel with FBR**. The cancellation is recorded only if FBR confirms it, and a refusal or unreadable reply leaves the invoice unchanged. |
| Report failures within 24 hours | 150XA(c)/(d) | FBR outages, token failures, unexpected stops of the system (crash or power failure) and tampering automatically open incidents. Tampering is found by a daily check of the audit chain, every accepted invoice's seal and digital signature, the signing key and the chain of closings. A ready-to-print letter to FBR and the Commissioner lists the affected invoices. |
| Keep records for 6 years | 150S | Accepted invoices cannot be edited or deleted (database triggers). Nightly backups are written to the backups folder; off-site copying is the client's responsibility. |
| Digital signature on each invoice | 150R(4)(b) | Each invoice accepted by FBR is signed with the installation's Ed25519 key over its FBR number and seal. The signature and key fingerprint are printed; the public key can be downloaded for auditors (Settings → System). |
| Closing at the close of each day, week and month | 150R(4)(f) | The worker records closings automatically within an hour of each period ending (ISO weeks, Monday to Sunday), catching up after downtime. Each closing counts the period's documents by status and totals what was reported; closings are hash-chained and cannot be changed (Tax periods & returns). |
| Logs of every adjustment, cancellation and system event | 150R(4)(g), 150R(6) | Append-only, hash-chained audit trail; FBR API log; incident register. |
| Signboard with the software registration number | 150R(11) | The registration number is recorded under Settings → FBR integration and printed on every invoice; the "Integrated with FBR" signboard is printed from the same page for each outlet. |
| Stock transfer notes for own warehouses | STGO 25/2026 | Serially numbered notes (STN-000001…) in the Annexure-A particulars, despatch and warehouse copies, receipt acknowledgement, cancellation with reason, register export for the monthly reconciliation. The same-STRN condition must be confirmed; otherwise the user is told to issue a sales tax invoice. |
| Advance receipt invoices | s.23(1), 150S(2) | An invoice can be marked as an advance receipt invoice: it is reported through DI like any sale invoice, printed as "ADVANCE RECEIPT INVOICE", and the final invoice records the advance invoices it adjusts. FBR has not yet notified a separate format (s.2(1AA)); update when it does. |
| Federal excise duty particulars | 150R(13)(aa)–(ff) | FED type, rate, price per unit, amount and the FED Schedule/SRO reference and serial are kept per line (defaults from the product), printed under the line and checked before submission. |
| Account for every reported invoice in Annex-C | 150XD(2) | Annex-C reconciliation export (Excel/CSV): every document with an FBR number in the period, including those cancelled, with buyer and tax amounts, to match against Annex-C before filing. |
| Answer FBR's electronic scrutiny | 150HB (SRO 1655(I)/2026) | Sales register, tax summary and Annex-C reconciliation give the figures to explain a discrepancy within the time allowed. |

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
| 16 | Continuity during FBR outages | 150S, 150XC | Queue with automatic retry; outage incident opened | `internal/service/worker.go`, `health.go` | TestOutageQueuesAndWorkerResubmits, TestServiceUnavailableIsRetryable | Implemented |
| 16a | Offline invoices identified and uploaded within 24 hours of restoration | 150XC | Immediate resubmission on recovery; printed note "Issued in offline mode … reported to FBR on …"; invoices issued offline stay on the dashboard's "not yet reported" count, with their age, until FBR accepts them (also when the upload after recovery is rejected); provisional printout until then; queued uploads continue after licence expiry and never block other companies' queues | `internal/service/health.go`, `worker.go`, `submit.go`; Dashboard; `internal/printing/render.go` | TestRecoveryResubmitsQueuedAtOnce, TestOfflineInvoiceTrackedUntilAccepted, TestRefusedQueuedInvoiceIsDeferred, TestRenderWatermarks | Implemented |
| 17 | Sandbox scenarios before production | IRIS onboarding | 28 scenarios with FBR samples; sandbox and practice runs; progress tracking | Scenarios page; `internal/service/scenarios.go` | TestScenariosInSandbox | Implemented |
| 18 | scenarioId only in sandbox | DI v1.12 | Stripped outside the sandbox | `internal/fbr/client.go` | TestScenarioIdStrippedInProduction | Implemented |
| 19 | Production token and IP whitelisting | IRIS/PRAL | Encrypted token storage; production switch refused without a token; connection test | Settings → FBR integration; `internal/service/masters.go` | TestProductionRequiresToken, TestUnauthorized | Partial (IP whitelisting is arranged with PRAL) |
| 20 | QR code v2, 25×25, 1 inch, encoding FBR number | DI v1.12 §6; 150R(4)(e), 150R(13)(b) | Forced version 2; SVG sized 1in × 1in | `internal/printing/qr.go` | TestQRVersion2 | Implemented |
| 21 | FBR DI logo on invoice | DI v1.12 §6; 150R(13)(d) | Official logo from the specification built in; a newer image can be uploaded | `internal/printing/assets/fbr-di-logo.jpg`; Settings → System | TestRenderA4 | Implemented |
| 22 | Reported invoices cannot be altered | 150S integrity | Database triggers block updates and deletes of accepted invoices, items and audit log; tamper-evident seal | `internal/db/migrations/0001_init.sql`; `internal/service/invoices.go` (seal) | TestMigrateAndTriggers | Implemented |
| 23 | Returns and reductions by debit note | s.9 Sales Tax Act (FBR FAQ 17); rule 150S(2); DI spec errors 0026, 0034, 0035, 0036, 0057, 0067 | Debit note raised from an accepted invoice with `invoiceRefNo` (22 characters for an NTN seller, 28 for a CNIC seller); dated on or after the original and within 180 days of it; value and tax capped at the original; netted off output tax in reports | InvoiceView → Debit note; `internal/validate/validate.go` | TestDebitNote, TestNoteWindowAndElectricityRate | Implemented |
| 24 | Cancel/edit only within 72 hours; Commissioner approval later | STGO 01/2026 (30 Mar 2026); PRAL manual v1.6 | The 72 hours run from FBR's issue time ("dated"); cancellation requires a reason and IRIS reference; after 72 h a Commissioner approval reference; optional cancellation through FBR's service, recorded only on FBR's confirmation | `internal/service/invoices.go` (CancelInvoice); `internal/fbr/client.go` (CancelOutcome) | TestCancelThroughFBRAPI, TestCancelOutcome | Partial (cancel on IRIS until PRAL publishes the API format) |
| 25 | Report failures, disruptions and tampering within 24 hours | 150XA(c)/(d) | Auto-detected incidents for FBR outages, token failures, unexpected stops (heartbeat with orderly-shutdown marker) and tampering (daily check of the audit chain, seals, digital signatures, signing key and closings); incident register; printable letter to FBR and the Commissioner citing rules 150XA and 150XC | Incident register; `internal/service/health.go`, `integrity.go`; `internal/httpapi/h_admin.go` | TestIntegrityCheckDetectsTampering, TestUnexpectedStopIsRecorded | Partial (the client sends the letter) |
| 26 | Records for 6 years | 150S | Nightly backups with retention; manual backup; download | Settings → System; `internal/service/worker.go` | — | Partial (off-site copies are the client's job) |
| 27 | Audit trail; logs of adjustments, cancellations and system events | 150R(4)(g) | Append-only, SHA-256 hash-chained log with verification | Audit trail page; `internal/store/audit.go` | TestMigrateAndTriggers | Implemented |
| 28 | Access control | Good practice | Roles: admin, manager, accountant, operator, auditor; sessions + CSRF; hashed API keys; lockout after failed logins | `internal/httpapi/server.go`, `perms.go`; `internal/store/users.go` | TestRoutesRegister | Implemented |
| 29 | Protection of FBR tokens | Good practice | AES-256-GCM encryption with `master.key`; tokens never returned by the API or logged | `internal/security/security.go`; `internal/fbr/client.go` | TestProductionRequiresToken | Implemented |
| 30 | Buyer status checks | Good practice | Active Taxpayer List and registration-type lookups, single or for all buyers; master records differing from FBR are flagged; inline check in the invoice editor | Customers → Check FBR / Check all with FBR; FBR reference library → Buyer check | TestReferenceAPIs | Implemented |
| 31 | Monthly return: payment and filing on time; Annexure-C complete | s.26; return due dates (generally 15th / 18th of the following month); FBR's extension orders (e.g. C.No.9(11)ST-LP&E/Misc/2016, 2 Oct 2026, for the August 2026 period) | Compliance calendar with per-company due days and countdowns; FBR extensions of the filing date recorded with their reference (payment date unchanged); period-close checklist (unreported, rejected, unreconciled, drafts, incidents); supplies by sale type and rate as they feed Annexure-C | Tax periods & returns; `internal/service/compliance.go` | TestUpcomingDeadlines, TestReturnDueDateClampsToMonthEnd, TestDuplicateInvoiceAndPeriodReview, TestReturnFilingExtension | Implemented |
| 32 | Use of FBR's reference data | DI v1.12 reference APIs | Synced lists (provinces, document and transaction types, UoM, SRO item codes, HS codes); live SaleTypeToRate, SroSchedule, SROItem and HS_UOM; reference library; FBR unit and SRO pickers in the editor | FBR reference library; `internal/service/refdata.go` | TestReferenceAPIs, TestDashboardSearchAlertsAndCompliance | Implemented |
| 33 | Prompt action on outages, rejections and the 24-hour upload | 150XA, 150XC | Notifications and e-mail alerts checked every 5 minutes (FBR unreachable, token rejected, invoices not reported, rejected or unreconciled invoices, unreported incidents, deadlines); optional daily summary | Bell; Settings → System & licence → E-mail notifications; `internal/service/alerts.go`, `notify.go`; `internal/mail` | TestEmailNotifications, TestNotifyPasswordIsEncrypted, TestSendPlainRelay | Implemented |
| 34 | extraTax left empty where FBR refuses a value | DI v1.12 error 0091; reports from other integrators | Sent empty for reduced-rate, exempt, zero-rated and cotton ginner supplies | `internal/domain/domain.go`; `internal/tax/engine.go` | TestReducedRateExtraTaxEmpty, TestExtraTaxEmptyForExemptZeroRatedAndCottonGinners | Implemented |
| 35 | Payloads FBR's parser accepts | Reports from other integrators | Plain text in names, addresses and descriptions (no control characters, double quotes or backslashes); warning when the same product is on two lines | `internal/service/invoices.go` (fbrText); `internal/validate/validate.go` | TestPayloadTextIsPlain, TestRepeatedLinesWarned | Implemented |
| 36 | Digital signature created and recorded on each invoice | 150R(4)(b) | Ed25519 key per installation (private key encrypted with `master.key`); each accepted invoice signed over its FBR number and seal; signature and key fingerprint printed; public key downloadable; earlier invoices signed by the worker; signature and key checked daily | `internal/service/signing.go`, `integrity.go`; Settings → System | TestInvoiceDigitalSignature, TestSigningKeyFaults | Implemented |
| 37 | Closing at the close of each day, week and month | 150R(4)(f) | Automatic day, ISO-week and month closings with document counts and reported totals; catch-up after downtime; hash-chained and immutable (triggers); chain verified daily | `internal/service/closings.go`, `internal/store/closings.go`; Tax periods & returns | TestClosingPeriods, TestDayWeekMonthClosings | Implemented |
| 38 | "Integrated with FBR" signboard; software registration number on invoices | 150R(11), 150R(13)(c) | Registration number recorded per company and printed on invoices and the incident letter; A4 signboard with FBR's logo per outlet | Settings → FBR integration; `internal/printing/templates/signboard.html` | TestRenderA4 | Partial (FBR issues the number; the client displays the signboard) |
| 39 | Tax period, HS code, total discount and sales tax withheld always printed | 150R(13)(l), (r), (v), (x) | Printed on every invoice whatever the print settings | `internal/printing/templates/invoice_a4.html`, `invoice_thermal.html` | TestRenderA4 | Implemented |
| 40 | Federal excise duty particulars | 150R(13)(aa)–(ff) (SRO 1666(I)/2026) | FED type, rate, price per unit, amount (in or otherwise than in sales tax mode) and FED Schedule/SRO and serial per line, defaults from the product, printed under each line; warning when missing | Invoice editor; Products; `internal/printing/render.go` (fedNote) | TestFEDParticularsAndAdvanceReceipts | Implemented |
| 41 | Advance receipt invoices | s.23(1), s.2(1AA) (Finance Act 2026); 150S(2) | Sale invoice marked as advance receipt, reported through DI, printed as "ADVANCE RECEIPT INVOICE"; final invoices record the advance invoices adjusted; both protected after acceptance | Invoice editor; import column `advance_receipt` | TestFEDParticularsAndAdvanceReceipts, TestImportAdvanceReceiptAndFEDColumns | Partial (FBR has not yet notified a separate format) |
| 42 | Stock transfer notes for goods moved to own warehouses | STGO 25/2026 | Numbered notes with the Annexure-A particulars, same-STRN confirmation, despatch and warehouse copies, receipt, cancellation with reason, no deletion (triggers), register export | Stock transfer notes page; `internal/service/transfers.go` | TestStockTransferNotes | Implemented |
| 43 | Every reported invoice accounted for in Annex-C | 150XD(2) (SRO 1666(I)/2026) | Annex-C reconciliation report and Excel/CSV export, including cancelled documents and offline flags | Reports → Annex-C reconciliation; Tax periods & returns | TestAnnexCReconciliation | Implemented |
| 44 | FBR's error wording and scenario matrix | DI v1.12 §7, §8, §10 | Catalogue of all 108 codes generated from the specification; scenario matrix transcribed in full | `internal/validate/catalog.go`; `internal/domain/scenarios.go` | TestCatalogue, TestScenarioMatrixComplete, TestScenarioMatrixRows | Implemented |
| 45 | Registration details for IRIS | PRAL DI user manual v1.5 | The exact technical details to enter (system provider, software type, version, business nature, sector, CRM user) and this server's public IP for whitelisting | Settings → FBR integration | — | Implemented |

## 7. Gaps and client responsibilities

- **IP whitelisting and tokens:** the client (or the practitioner on their behalf) requests these on IRIS / from PRAL.
- **Software registration number and signboard:** record the number FBR issues for the software under Settings → FBR integration, then print and display the signboard at each outlet (rule 150R(11)).
- **Off-site backups:** copy the `backups` folder and `master.key` to separate storage regularly, and keep records for six years.
- **Cancellation on IRIS:** FBR's public DI API v1.12 does not define a cancellation call, and PRAL has not published the format of the reported `cancelinvoicedata` service. Until it does, cancel on IRIS within 72 hours, then record the cancellation in the product. When the format is confirmed, enable the API option in `config.json` (§5).
- **After an outage:** check the dashboard once the connection is back, and make sure the "not yet reported" count reaches zero within 24 hours (rule 150XC). File the rule 150XA letter for the incident.
- **Rule 150XA letters:** the product prepares the letter; the client signs and sends it to FBR and the Commissioner within 24 hours.
- **Advance receipt invoice format:** FBR is to notify the format (s.2(1AA)). Until it does, the product reports the advance as a sale invoice marked "advance receipt"; review when PRAL updates the DI specification.
- **FED particulars:** enter the FED type and the Schedule/SRO reference and serial on FED-liable products; the product warns when they are missing.
- **Stock transfer notes:** have the warehouse sign the warehouse copy and record the receipt; reconcile the notes monthly with stock records.
- **Master key:** keep `master.key` with the backups — it also protects the invoice signing key.
- **Legal updates:** rates, SROs and scenario lists change. Keep the product updated under a support contract and re-check the items marked **Verify**.

## 8. Sources and items to verify

Official copies on fbr.gov.pk, read on 9 October 2026:

- [SRO 69(I)/2025](https://download1.fbr.gov.pk/SROs/2025129141598258SRO69(I)2025.pdf) (29 January 2025) — Chapter XIV, Sales Tax Rules 2006.
- [SRO 709(I)/2025](https://download1.fbr.gov.pk/SROs/2025423124414622SRO709dated22April,2025.pdf) (22 April 2025), [SRO 1413(I)/2025](https://download1.fbr.gov.pk/SROs/2025811681810559SRO1413.pdf) (1 August 2025) and [SRO 1852(I)/2025](https://download1.fbr.gov.pk/SROs/2025924149054920SRO1852.pdf) (24 September 2025) — integration deadlines.
- [Sales Tax General Order 01 of 2026](https://download1.fbr.gov.pk/Docs/2026331133557466STGO01of2026.pdf) (30 March 2026) — one or more integrators; 72-hour rule.
- [Finance Act 2026](https://download1.fbr.gov.pk/Docs/20266291261044366FinanceAct2026.pdf) and [Circular 01 of 2026](https://download1.fbr.gov.pk/Docs/20269111791418742Circular01of2026.pdf) (11 September 2026) — s.23(1), s.21(2), s.33(25), s.33(29)–(31), s.8B, s.9.
- [SRO 1655(I)/2026](https://download1.fbr.gov.pk/SROs/20269251892143851SRO1655.pdf) (25 September 2026) — electronic scrutiny, Chapter XII-A.
- [Sales Tax General Order 25 of 2026](https://download1.fbr.gov.pk/Docs/20269281593856844STGO25of2026.pdf) (28 September 2026) — stock transfers to own warehouses.
- [SRO 1666(I)/2026](https://download1.fbr.gov.pk/SROs/202693089244653SRO1666dated29-09-2026.pdf) (29 September 2026) — FED particulars, advance receipt invoices, rule 150XD(2).
- [PRAL Technical Specification for DI API v1.12](https://download1.fbr.gov.pk/Docs/20257301172130815TechnicalDocumentationforDIAPIV1.12.pdf) (24 July 2025) — APIs, fields, scenarios, error codes, QR and logo. No later version is published on FBR's site.
- [PRAL Digital Invoicing User Manual v1.5](https://download1.fbr.gov.pk/Docs/20257301171649798DIUserManualV1.5.pdf) — IRIS registration, technical details, IP whitelisting.
- FBR's [Digital Invoicing FAQs](https://fbr.gov.pk/faqs/173967/173969) and [list of licensed integrators](https://fbr.gov.pk/list-of-license-interprator/173967/173971).

Other sources:

- Sales Tax Act 1990, s.23(1)(b), s.23(5)/(6), s.40C — invoice particulars, CNIC requirement, electronic invoicing.
- PRAL Digital Invoicing user manual v1.6 (16 April 2026, as reported) — IRIS cancellation screens (**Verify**: FBR's DI page still links v1.5).
- Draft SRO 288(I)/2026 (18 February 2026) — income-tax online integration (separate regime, draft).
- Reports from other DI integrations in use (reviewed October 2026): FBR refusing a numeric zero extraTax for scenarios SN005–SN007, SN009 and SN028; FBR flagging repeated lines; FBR checking the invoice date against a UTC clock just after midnight (error 0043) (**Verify** with PRAL).
- IRIS → Digital Invoicing — onboarding screens, scenario assignment, token issue.

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
