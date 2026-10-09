# Architecture

## 1. Overview

```
 Browser (React SPA)        ERP / POS (API key)
        │  HTTPS :8443            │
        ▼                         ▼
 ┌──────────────────────────────────────────────────────────┐
 │ einvoice (single Go binary)                                │
 │  internal/httpapi  REST API, sessions+CSRF, API keys,      │
 │                    roles, SPA (internal/webui, embedded)   │
 │  internal/service  invoices, submit state machine, worker, │
 │                    scenarios, import/export, auth, refdata,│
 │                    alerts, e-mail, return calendar         │
 │  internal/tax      exact-decimal tax engine                │
 │  internal/validate pre-submission rules + FBR error codes  │
 │  internal/printing A4 / 80 mm invoices, QR v2 (1 inch)     │
 │  internal/store    data access (amounts in integer paisa)  │
 │  internal/db       SQLite (WAL), migrations, triggers      │
 │  internal/license  Ed25519 licences                        │
 │  internal/security bcrypt, AES-GCM vault, hash chain       │
 │  internal/fbr      DI API v1.12 client + reference APIs    │
 │  internal/fbrmock  built-in FBR simulator (training/tests) │
 │  internal/mail     SMTP client for e-mail alerts           │
 └───────────────┬───────────────────────────┬──────────────┘
                 │ HTTPS, Bearer token,      │ SMTP (optional)
                 │ static whitelisted IP     ▼
                 ▼                       company mail server
        https://gw.fbr.gov.pk  (FBR Digital Invoicing / PRAL)
```

Everything runs in one process with no external services other than FBR (and, if e-mail alerts are switched on, the company's own mail server). The database is a single SQLite file using the pure-Go `modernc.org/sqlite` driver; no cgo is needed, so cross-compiling for Windows is trivial.

## 2. Packages

| Package | Responsibility |
|---|---|
| `cmd/einvoice` | CLI: `serve`, `service`, `mock-fbr`, `backup`, `reset-password`, `version` |
| `cmd/licensegen` | Vendor tool: `keygen`, `issue`, `verify` |
| `internal/app` | Startup: config, logging, DB, vault, licence, service; HTTP(S) server and worker; self-signed TLS |
| `internal/brand` | Product, vendor and support names used everywhere (one place to change them) |
| `internal/config` | `config.json` defaults and loading; data directory selection (keeps the data folder of installations made before the rename) |
| `internal/domain` | FBR vocabulary: environments, document types, statuses, 27 sale types, 28 sandbox scenarios, provinces, UoMs |
| `internal/tax` | Rate parsing (`18%`, `Exempt`, `Rs.200`, mixed), line computation, totals, amount in words |
| `internal/validate` | Payload validation mirroring FBR's rules; catalogue of all 108 FBR error codes (FBR's wording, generated from the DI specification v1.12) with fixes |
| `internal/fbr` | Payload JSON model, HTTP client with failure classification and call logging, reference APIs |
| `internal/fbrmock` | Simulator implementing FBR's endpoints, validations and fault injection |
| `internal/store` | Typed data access for all tables, reports, audit log; dashboard and search queries (`insights.go`) |
| `internal/service` | Business logic and background worker; see below |
| `internal/mail` | Minimal SMTP client: implicit TLS (465), STARTTLS (587) or plain relay (25); AUTH PLAIN/LOGIN; multipart text + HTML |
| `internal/httpapi` | REST routes, authentication, authorisation, error mapping, security headers |
| `internal/printing` | HTML templates (A4 and 80 mm invoices, stock transfer note, "Integrated with FBR" signboard), QR generation, the official FBR DI logo (`assets/fbr-di-logo.jpg`) |
| `internal/winsvc` | Windows service integration |
| `web/` | React + TypeScript + Vite source; built into `internal/webui/dist` and embedded with `go:embed` |

Main files of `internal/service`:

| File | Responsibility |
|---|---|
| `invoices.go`, `submit.go` | Invoice editing, payload building (plain-text cleaning of names and descriptions), submission state machine, reconciliation, duplicating |
| `worker.go`, `integrity.go`, `health.go` | Queue, outage incidents (rule 150XA), unexpected-stop detection, integrity checks (audit chain, seals, signatures, signing key, closings), backups |
| `signing.go` | Invoice digital signature (rule 150R(4)(b)): Ed25519 key per installation, private key encrypted with the vault; signing on acceptance and back-filling of earlier invoices |
| `closings.go` | Day, ISO-week and month closings (rule 150R(4)(f)), recorded hourly by the worker with catch-up |
| `transfers.go` | Stock transfer notes (STGO 25 of 2026) |
| `publicip.go` | On request, asks a public echo service for this server's internet address (for PRAL's IP whitelisting) |
| `refdata.go` | Sync and live look-ups of FBR's reference APIs (provinces, document and transaction types, UoM, HS codes, SRO item codes, SaleTypeToRate, SroSchedule, SROItem, HS_UOM, ATL status, registration type) |
| `compliance.go` | Sales tax return calendar (payment and filing days per company, FBR filing-date extensions), period-close review |
| `alerts.go` | Alerts shown under the bell and e-mailed: connection, token, unreported / rejected / unreconciled invoices, drafts, incidents, deadlines, stale reference data, sandbox progress, backups, licence |
| `notify.go` | E-mail settings (password encrypted with the vault), immediate alerts and the daily summary |
| `scenarios.go`, `importer.go`, `masters.go`, `auth.go` | Sandbox scenarios, Excel/CSV import and export, companies/customers/products, users and API keys |

The web app (`web/src`) has a sidebar layout for desktops and a bottom tab bar for phones (`App.tsx`), a header with global search, notifications and account menu (`components/Header.tsx`), light/dark themes (`theme.ts`) and hand-made SVG charts with table views (`components/Charts.tsx`). Pages are in `web/src/pages`; `ReferenceLibrary.tsx` and `Compliance.tsx` present FBR's reference data and the return calendar.

## 3. Invoice lifecycle

```
            Save draft                     Save & submit
 (new) ─────────────────► DRAFT ───────────────┐
                            ▲ edit             │ local validation (errors → stay DRAFT, 422)
                            │                  ▼
                       REJECTED ◄────── SUBMITTING ──────► ACCEPTED ──► CANCELLED
                       (FBR errors)        │   │  FBR 00      (locked)    (after IRIS
                                           │   │                           cancellation)
                         not sent / 503 /  │   │  timeout after send,
                         429 / auth fail   ▼   ▼  502 or other 5xx, bad response
                                       QUEUED  UNCERTAIN ("Needs reconciliation")
                                  (auto retry)  → accepted (FBR no. entered) / retry / draft
```

Submit sequence (`internal/service/submit.go`):

1. Lock the invoice (per-invoice mutex). Already ACCEPTED → return it (idempotent). SUBMITTING/UNCERTAIN → refuse.
2. Run local validation; any error blocks submission.
3. Check environment prerequisites: token; licence for production (not for invoices already issued and queued).
4. Build the payload, store it with its hash, and mark the invoice SUBMITTING **before** sending.
5. Detach from the caller's context, so the exchange and the recording of its outcome complete even if the browser or ERP disconnects (the FBR client's timeout bounds the calls).
6. Optionally call `validateinvoicedata`, then `postinvoicedata`.
7. Classify the outcome (`internal/fbr/client.go`):

| Outcome | Classification | Status |
|---|---|---|
| FBR status 00 | — | **ACCEPTED**: FBR numbers stored, seal computed, in one transaction |
| FBR status 01 | — | **REJECTED** with per-line errors |
| Connection refused / DNS failure (request never written) | `not_sent` | **QUEUED**, backoff 30 s × 2ⁿ up to 30 min |
| HTTP 503/429 (and 502 on read-only calls) | `unavailable` | **QUEUED** |
| HTTP 401/403 | `auth` | **QUEUED** for 15 min; auth-failure incident |
| Timeout or broken connection after the request was written; 502 or other 5xx on posting; undecodable response | `uncertain` / `decode` | **UNCERTAIN**: never retried automatically |
| Other 4xx | `client` | **REJECTED** |

Invoices queued for connectivity or token reasons record `offline_since`; they stay on the dashboard's "not yet reported" count until FBR accepts them (24-hour upload rule).

Invoices left in SUBMITTING are released by `RecoverStuckSubmissions`: never sent → QUEUED; possibly sent → UNCERTAIN. This runs at startup for all of them, and on every worker tick for those older than twice the FBR timeout plus a minute. The worker also:

- re-submits due QUEUED invoices, and the whole queue at once when posting to FBR works again; a connection test after an outage probes with one queued invoice. Invoices refused before reaching FBR (e.g. token removed) are deferred for 15 minutes so they cannot block the queue;
- opens and closes rule 150XA incidents for production only: FBR unreachable (3+ failed calls over 10+ minutes), auth failures, unexpected stops (heartbeat without an orderly-shutdown marker, `internal/service/integrity.go`) and tampering;
- runs a daily integrity check of the audit chain, every accepted invoice's seal, chain link and digital signature, the signing key pair and the chain of closings;
- signs accepted invoices that have no signature yet (invoices accepted before version 4 of the schema);
- records due day, week and month closings once an hour;
- checks alerts every 5 minutes and sends e-mails if they are switched on (in its own goroutine, so a slow mail server never delays the queue);
- purges expired sessions;
- writes the daily backup.

## 4. Data model

The schema is in `internal/db/migrations/` (`0001_init.sql`; `0002_offline_since.sql` adds the offline marker; `0003_return_days.sql` adds each company's return payment and filing days; `0004_fbr_rules.sql` adds the Chapter XIV requirements: software registration number, invoice signature, advance receipt fields, FED particulars on lines and products, closings, stock transfer notes and return filing extensions). Migrations are applied in order at startup, each once.

**Tables**

- `companies`, `users`, `user_companies`, `sessions`, `api_keys`
- `customers`, `products`, `invoices`, `invoice_items`, `number_series`
- `fbr_calls`, `ref_cache`, `ref_hs_codes`
- `audit_log`, `scenario_runs`, `incidents`, `settings`
- `closings`, `stock_transfers`, `stock_transfer_items`, `return_extensions`

**Conventions**

- **Money** is stored as integer **paisa**; calculations use exact decimals (`shopspring/decimal`) and round to 2 decimals per FBR.
- **Uniqueness:** (company, environment, internal number), external reference, and FBR invoice number.
- **Immutability triggers:**
  - accepted or cancelled invoices accept only the transition to CANCELLED, cancellation details, print count and timestamps;
  - their items cannot change;
  - reported invoices cannot be deleted;
  - `audit_log` and `closings` rows can never be updated or deleted;
  - stock transfer notes and their lines cannot be deleted;
  - an invoice's digital signature can be set once and never changed.
- **Invoice seal:** each accepted invoice stores a SHA-256 hash of its canonical content chained to the company's previous seal. The invoice view shows "Seal verified" or "Seal mismatch".
- **Digital signature:** each accepted invoice stores an Ed25519 signature over `veridian-di-signature-v1|<FBR invoice number>|<seal>`. The public key and its fingerprint are shown under Settings → System and can be downloaded as PEM; if the vault cannot open the private key, signatures are still checked against the stored public key.
- **Closings:** each closing stores a snapshot of the period's documents and `hash = SHA-256(previous closing hash | closing)`.
- **Audit chain:** every audit entry stores `hash = SHA-256(previous hash + entry)`. **Verify integrity** (and the worker, daily) recomputes the chain and every invoice seal; a problem opens a tampering incident. The hashes are not keyed, so they detect edits made without recomputing them; protecting the server and database file remains essential.

## 5. Security

| Area | Design |
|---|---|
| Passwords | bcrypt (cost 12); policy of 8+ characters with letters and digits; lockout for 15 minutes after 5 failures. Login failures (unknown user, wrong password, locked account) return the same message and timing, so usernames cannot be probed. Changing a password signs out the user's other sessions. |
| Sessions | Random tokens in an HttpOnly, SameSite cookie with sliding 12-hour expiry; CSRF token required on every state-changing request |
| API keys | `eik_` prefix; only a SHA-256 hash stored; scoped to one company; limited permissions |
| Authorisation | Role → permission map (`internal/httpapi/perms.go`); every company-scoped route checks the user's company access; users limited to some companies see only those companies' audit entries. The FBR environment of an invoice always follows the company setting, so only users allowed to change company settings can move a company to production. |
| FBR tokens | AES-256-GCM with `master.key` (generated on first start); never returned by the API; never written to logs or the FBR call log |
| E-mail | SMTP password encrypted like the FBR tokens and never returned by the API; certificates verified; a password is never sent over an unencrypted connection to another computer |
| Web | Strict security headers and Content Security Policy (`script-src 'self'`); print and letter pages allow only a per-request nonce'd script. Uploaded images are restricted to image types, max 2 MB; SVGs with scripts, event handlers or embedded content are rejected, and images are served with a sandboxing CSP. Import uploads are capped at 20 MB with bounded XLSX decompression; QR PNG size is capped. |
| Transport | HTTPS by default (self-signed or own certificate) |
| Licence | Ed25519 signature verified offline against the embedded public key |

## 6. Simulator and testing

`internal/fbrmock` implements FBR's DI endpoints and reference APIs with FBR-like validation messages and error codes. It supports fault injection: timeout, dropped connection, 500, 503, 401. It is used by:

- the **Training simulator** environment (in-process, nothing leaves the server);
- `einvoice mock-fbr` for ERP developers;
- the automated tests.

| Test file | Covers |
|---|---|
| `internal/tax/tax_test.go` | Rates, further tax, Third Schedule, reduced rate, withholding, rounding, totals, amount in words, scenario arithmetic |
| `internal/validate/validate_test.go` | Header and line validation, NTN/CNIC normalisation, debit notes (180-day window), dates, repeated lines, error catalogue (86 sales and 22 purchase codes, FBR's wording), local checks citing documented codes |
| `internal/fbr/client_test.go` | Success, rejection, auth, scenarioId stripping, failure classification (timeout/drop/refused/503), reference APIs (with GET fallback for `statl` and `Get_Reg_Type`), JSON format |
| `internal/db/db_test.go` | Migrations and immutability triggers |
| `internal/service/service_test.go` | Full lifecycle, validation blocking, sandbox scenarios, reconciliation, outage queue and worker, rejection then fix, production token rule, idempotency, fiscal year |
| `internal/service/compliance_test.go` | Return due dates and calendar, period review, duplicating invoices, plain-text payloads |
| `internal/service/notify_test.go` | E-mail settings, immediate alerts (once a day per alert), daily summary, password encryption |
| `internal/service/integrity_test.go`, `health_test.go` | Tampering detection, unexpected stops, outage incidents |
| `internal/service/fbrrules_test.go` | Digital signature and key faults, day/week/month closings and their chain, stock transfer notes, return filing extensions, Annex-C reconciliation, FED particulars, advance receipt invoices, import of the new columns |
| `internal/domain/scenarios_test.go` | The DI specification's business activity × sector scenario matrix |
| `internal/mail/mail_test.go` | Sending through a test SMTP server, STARTTLS required, no password over plain connections, address validation |
| `internal/license/license_test.go` | Licence states and limits; developer build |
| `internal/printing/*_test.go` | QR version 2 / 25 modules; A4 and thermal rendering; copies; watermarks |
| `internal/httpapi/routes_test.go` | Route registration |
| `internal/httpapi/security_test.go` | Audit trail scoped to the user's companies, invoice environment not overridable, login failures indistinguishable, password change signs out other sessions, print-page CSP, QR and upload size limits |
| `internal/httpapi/insights_test.go` | Dashboard, search, alerts and compliance endpoints |
| `internal/config/config_test.go`, `internal/app/tls_test.go` | Data folder selection (including installations made before the rename), certificates |

Run all with `make test` (or `GOTOOLCHAIN=local go test ./...`).

## 7. Build

- `web/`: `npm ci && npm run build` writes `internal/webui/dist`. The built bundle is committed so that `go build` works without Node.js.
- Go 1.24: `go build ./cmd/einvoice`. Release builds inject the version, build date and licence public key via `-ldflags -X` (see `Makefile`, `scripts/build-release.sh`).
- Packaging:
  - `packaging/windows/einvoice-suite.iss` (Inno Setup);
  - `packaging/linux/` (systemd unit and installer);
  - `Dockerfile`.

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
