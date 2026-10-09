# FBR Onboarding Guide — Taking a Client Live

This is the step-by-step procedure a tax practitioner follows for **each client** to start issuing FBR Digital Invoices with Veridian E-invoicing Pakistan. Section 4 explains the approval and certification paths for you as the **software vendor**.

> Screens on IRIS change from time to time. Items marked **Verify** should be checked against the current IRIS / FBR notices.

---

## 1. Prerequisites

| Item | Notes |
|---|---|
| Active sales-tax registration | The client's NTN (7 digits) or CNIC (13 digits for individuals) must be active on IRIS. |
| IRIS credentials | Needed for Digital Invoicing registration, tokens and cancellations. |
| Static public IP | Ask the client's ISP for a static IP. FBR accepts production calls only from a whitelisted IP. |
| Server | Veridian E-invoicing Pakistan installed ([INSTALLATION.md](INSTALLATION.md)). The setup wizard is completed with the business details exactly as on IRIS. |
| Licence | A licence covering the client's NTN is installed (Settings → System) before switching to production. Sandbox and simulator work during the 30-day evaluation. |

## 2. Go-live procedure

### Step 1 — Register for Digital Invoicing on IRIS and choose the integrator

On IRIS, open **Digital Invoicing** and submit the integration request:

1. Choose **PRAL** as the integrator. PRAL integration is free.
2. Veridian E-invoicing Pakistan is the client's own e-invoicing system. It calls FBR's DI API directly with the client's security token, so no third party sees the data.
3. If the client prefers another licensed integrator, that integrator handles the request and the IP whitelisting instead. FBR's list (October 2026) is under **Help → Documents & integrators** and on fbr.gov.pk.

Then enter the technical details and the IP addresses (PRAL Digital Invoicing User Manual v1.5). **Settings → FBR integration → Registration on IRIS** shows exactly what to type, with copy buttons:

| IRIS field | Value |
|---|---|
| Technical contact person, mobile, e-mail | The client's or your own contact |
| ERP / System Provider | Veridian Partners Consultancy Private Limited — Veridian E-invoicing Pakistan |
| Software Type | On Premises |
| Software Version | The version shown on the page (Help → Support) |
| CRM user ID and password | The e-mail to be used on PRAL's support portal, dicrm.pral.com.pk |
| Business Nature / Sector | As in Settings → Company; IRIS takes one sector for sandbox testing |
| IP whitelisting | Hosting server company (ISP or data centre), country, and 1–3 static public IPs. **Find this server's public IP** on the same page shows the address the server's traffic comes from. PRAL accepts or rejects it within about two working hours. |

### Step 2 — Business activity and sector

On IRIS, record the client's business activity (Manufacturer, Importer, Distributor, Wholesaler, Exporter, Retailer, Service Provider, Other) and sector. IRIS assigns the **sandbox scenarios** from these.

Record the same values in the product under **Settings → Company → Digital Invoicing profile**. The Scenarios page then suggests the matching scenarios.

### Step 3 — Sandbox token

IRIS issues a sandbox security token.

1. In the product open **Settings → FBR integration → Sandbox token**.
2. Paste the token, enter its expiry date and click **Save token**. Tokens are stored encrypted and are never displayed again.
3. Click **Test connection**.

### Step 4 — Run the assigned scenarios

Open **Sandbox scenarios**:

1. Click **Set assigned scenarios** and tick exactly the scenarios IRIS lists, or use **Use suggested**.
2. Optionally press **Practice** on each scenario. This runs it in the built-in training simulator; nothing is sent to FBR.
3. Press **Run in sandbox** for each scenario, or **Run pending in FBR sandbox** for all of them.
   - The product creates the scenario invoice with the client's NTN as seller and FBR's sample buyer and items.
   - It sends the invoice to the sandbox API with the `scenarioId`.
   - A scenario is **passed** when FBR returns status 00 and an invoice number.
4. "Amounts" lets you choose between amounts **computed by the tax engine** (default, proves the calculation) and **FBR sample values verbatim**.
5. If a scenario fails, open the linked invoice. FBR's error code is shown with the fix (see also **Help & error codes**). Correct the master data and run it again.

When all assigned scenarios have passed, the readiness card shows **Ready**. Confirm on IRIS that the scenarios show as completed.

### Step 5 — IP whitelisting

Confirm with PRAL that the server's static public IP is whitelisted. A missing whitelist usually shows as connection or authorisation failures in production while the sandbox works.

### Step 6 — Production token

After the scenarios pass, generate the **production token** on IRIS. It is valid for five years.

1. Save it under **Settings → FBR integration → Production token** with its expiry date.
2. Click **Test connection**. The product validates a sample invoice with FBR without recording it.

The dashboard warns 60 days before the production token expires.

### Step 7 — Switch to production

Under **Settings → FBR integration → Working environment**, click **Switch to fbr production**. The product refuses the switch unless:

- a production token is saved, and
- the licence covers the company's NTN.

From now on, every invoice saved with **Save & submit to FBR** is reported in real time and printed with the FBR number and QR code.

### Step 8 — Go-live checklist

- [ ] Invoice prefixes and print settings (logo, copies, terms) configured.
- [ ] The FBR Digital Invoicing logo prints on invoices (built in; replace it under Settings → System only if FBR issues a new one).
- [ ] Software registration number recorded (Settings → FBR integration) and the **"Integrated with FBR" signboard** printed and displayed at each outlet (rule 150R(11)).
- [ ] FED-liable products carry the FED particulars (type, rate as printed, Schedule/SRO and serial) — rule 150R(13)(aa)–(ff).
- [ ] If goods move to the client's own warehouses, staff trained on **Stock transfer notes** (STGO 25 of 2026).
- [ ] Customers (with FBR check) and products (HS code, UoM, sale type, rate, SRO) set up or imported.
- [ ] User accounts with the right roles; no shared logins.
- [ ] Printers tested (A4 and/or 80 mm thermal).
- [ ] Backups verified; off-site copy procedure agreed; `master.key` copied to safe storage.
- [ ] Staff trained on statuses, debit notes, cancellations and the incident procedure.
- [ ] ERP/POS integration (if any) tested against the simulator and sandbox ([ERP-INTEGRATION-API.md](ERP-INTEGRATION-API.md)).

## 3. Day-to-day compliance

| Situation | What to do in the product |
|---|---|
| Normal sale | **Save & submit to FBR**. Print only after the status is **Accepted by FBR**. |
| FBR rejected the invoice | Open it. The FBR errors are listed with fixes. Edit, then submit again. The invoice is not reported until accepted. |
| Internet/FBR outage | Invoices are **Queued** and sent automatically when FBR is reachable. An incident is opened automatically. |
| **Needs reconciliation** | FBR may have recorded the invoice, but no definite answer was received. Search for it on IRIS, then use **Reconcile with IRIS**: record the FBR number if found, otherwise resubmit or return it to draft. Never re-enter it as a new invoice — that could report the sale twice. |
| Goods returned, or value reduced after supply (e.g. post-sale discount) | Raise a **Debit note** from the accepted invoice. It carries the original FBR number and cannot exceed the original invoice's value or sales tax (FBR errors 0036/0067). It reduces output tax in the period. For an upward price revision, issue a supplementary sale invoice for the difference. |
| Invoice issued in error | Within **72 hours**: cancel it on IRIS, then **Cancel invoice** in the product with the reason and IRIS reference. After 72 hours: obtain the Commissioner's prior approval and enter its reference (STGO 01 of 2026). |
| System failure, power failure, tampering, prolonged outage | Report to FBR and the Commissioner within **24 hours** (rule 150XA(c)/(d)). Open **Incident register**, print the letter (it lists invoices issued during the incident), send it, and record the date and reference. Invoices issued offline must reach FBR within 24 hours of restoration (rule 150XC). |
| Payment received before supply | Issue an **advance receipt invoice** (tick the box on the invoice); on delivery invoice the balance and note the advance invoice (section 23(1), Finance Act 2026). |
| Goods sent to the client's own warehouse (same STRN) | **Stock transfer note**, not an invoice (STGO 25 of 2026); record the receipt when the warehouse signs. |
| FBR extends the return filing date | **Tax periods & returns → Record an FBR extension** with the notification's reference. |
| Before filing the return | Download the **Annex-C reconciliation** and match every FBR number with Annex-C (rule 150XD(2)); check the period's closings show **Chain intact**. |
| Token expiring | Generate a new production token on IRIS and save it before expiry. |
| Monthly return | **Reports → Sales register** reconciles with Annexure-C. The tax summary and monthly summary support the return. |

## 4. Approval and certification paths for the software vendor

### 4.1 What FBR actually certifies

FBR does not publish a general "approved software" list for taxpayer-side systems. Each **taxpayer's integration** is certified when:

1. the taxpayer registers for Digital Invoicing on IRIS and selects an integrator (PRAL or licensed);
2. the taxpayer's system successfully submits **all assigned sandbox scenarios**;
3. FBR issues the **production token**, and production traffic comes from the **whitelisted IP**.

Veridian E-invoicing Pakistan supports this path completely (Scenarios page, sandbox runs, token management). Each client you install is certified individually through it. Do not describe the product as "FBR approved". Describe it as "built to FBR DI technical specification v1.12; each installation is certified through FBR's sandbox scenarios".

### 4.2 Becoming an FBR licensed integrator (optional)

If you want to operate as an integrator yourself, apply under rules 150XE–150XQ of the Sales Tax Rules (Chapter XIV, SRO 69(I)/2025) through FBR's licensing committee; FBR publishes expressions of interest on its Digital Invoicing pages. The rules cover the application (150XH), the procedure for grant (150XI), the licensee's rights (150XJ), technical support (150XL), cancellation (150XN) and fees (150XO). Eight integrators are licensed as of October 2026, including PRAL, which provides integration free of cost under rule 150XF.

The product can be used either way: as the client's own system under PRAL integration (the default), or as part of your services as a licensed integrator.

### 4.3 Evidence pack to keep

Keep these ready for clients, auditors or FBR:

- [FBR-COMPLIANCE-GUIDE.md](FBR-COMPLIANCE-GUIDE.md): legal and technical requirement → feature → test evidence.
- [ARCHITECTURE.md](ARCHITECTURE.md): security design (encryption, audit chain, immutability).
- Automated test results (`make test`) and screenshots of passed sandbox scenarios per client.
- The client's IRIS confirmation of scenario completion and production token issue.

## 5. Troubleshooting common FBR errors

The complete list with fixes is under **Help & error codes** in the product.

| Error | Likely cause | Fix |
|---|---|---|
| 0401 | Token missing, expired, wrong environment, or seller NTN not matching the token | Re-enter the token for the active environment; check the company NTN |
| 0002 | Buyer NTN/CNIC invalid | 7-digit NTN without check digit, or 13-digit CNIC |
| 0046 | Rate not valid for the sale type on the invoice date | Choose from the rate suggestions (FBR SaleTypeToRate) |
| 0052 | HS code invalid / not matching the sale type | Correct the PCT code in the product master |
| 0077 / 0078 | SRO schedule or serial missing/invalid | Fill both SRO fields for reduced-rate or exempt items |
| 0091 | Extra tax must be empty for reduced-rate goods | Handled automatically; remove manual extra-tax overrides |
| Connection failures in production only | IP not whitelisted | Ask PRAL to whitelist the server's static IP |

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
