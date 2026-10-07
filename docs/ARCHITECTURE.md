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
 │                    scenarios, import/export, auth, refdata │
 │  internal/tax      exact-decimal tax engine                │
 │  internal/validate pre-submission rules + FBR error codes  │
 │  internal/printing A4 / 80 mm invoices, QR v2 (1 inch)     │
 │  internal/store    data access (amounts in integer paisa)  │
 │  internal/db       SQLite (WAL), migrations, triggers      │
 │  internal/license  Ed25519 licences                        │
 │  internal/security bcrypt, AES-GCM vault, hash chain       │
 │  internal/fbr      DI API v1.12 client + reference APIs    │
 │  internal/fbrmock  built-in FBR simulator (training/tests) │
 └───────────────┬──────────────────────────────────────────┘
                 │ HTTPS, Bearer token, static whitelisted IP
                 ▼
        https://gw.fbr.gov.pk  (FBR Digital Invoicing / PRAL)
```

Everything runs in one process with no external services. The database is a single SQLite file using the pure-Go `modernc.org/sqlite` driver; no cgo is needed, so cross-compiling for Windows is trivial.

## 2. Packages

| Package | Responsibility |
|---|---|
| `cmd/einvoice` | CLI: `serve`, `service`, `mock-fbr`, `backup`, `reset-password`, `version` |
| `cmd/licensegen` | Vendor tool: `keygen`, `issue`, `verify` |
| `internal/app` | Startup: config, logging, DB, vault, licence, service; HTTP(S) server and worker; self-signed TLS |
| `internal/config` | `config.json` defaults and loading; data directory selection |
| `internal/domain` | FBR vocabulary: environments, document types, statuses, 27 sale types, 28 sandbox scenarios, provinces, UoMs |
| `internal/tax` | Rate parsing (`18%`, `Exempt`, `Rs.200`, mixed), line computation, totals, amount in words |
| `internal/validate` | Payload validation mirroring FBR's rules; catalogue of FBR error codes with fixes |
| `internal/fbr` | Payload JSON model, HTTP client with failure classification and call logging, reference APIs |
| `internal/fbrmock` | Simulator implementing FBR's endpoints, validations and fault injection |
| `internal/store` | Typed data access for all tables, reports, audit log |
| `internal/service` | Business logic and background worker |
| `internal/httpapi` | REST routes, authentication, authorisation, error mapping, security headers |
| `internal/printing` | HTML templates, QR generation |
| `internal/winsvc` | Windows service integration |
| `web/` | React + TypeScript + Vite source; built into `internal/webui/dist` and embedded with `go:embed` |

## 3. Invoice lifecycle

```
            Save draft                     Save & submit
 (new) ─────────────────► DRAFT ───────────────┐
                            ▲ edit             │ local validation (errors → stay DRAFT, 422)
                            │                  ▼
                       REJECTED ◄────── SUBMITTING ──────► ACCEPTED ──► CANCELLED
                       (FBR errors)        │   │  FBR 00      (locked)    (after IRIS
                                           │   │                           cancellation)
                         not sent / 502 /  │   │  timeout after send,
                         503 / auth fail   ▼   ▼  5xx, bad response
                                       QUEUED  UNCERTAIN ("Needs reconciliation")
                                  (auto retry)  → accepted (FBR no. entered) / retry / draft
```

Submit sequence (`internal/service/submit.go`):

1. Lock the invoice (per-invoice mutex). Already ACCEPTED → return it (idempotent). SUBMITTING/UNCERTAIN → refuse.
2. Run local validation; any error blocks submission.
3. Check environment prerequisites: token; licence for production.
4. Build the payload, store it with its hash, and mark the invoice SUBMITTING **before** sending.
5. Optionally call `validateinvoicedata`, then `postinvoicedata`.
6. Classify the outcome (`internal/fbr/client.go`):

| Outcome | Classification | Status |
|---|---|---|
| FBR status 00 | — | **ACCEPTED**: FBR numbers stored, seal computed, in one transaction |
| FBR status 01 | — | **REJECTED** with per-line errors |
| Connection refused / DNS failure (request never written) | `not_sent` | **QUEUED**, backoff 30 s × 2ⁿ up to 30 min |
| HTTP 502/503/429 | `unavailable` | **QUEUED** |
| HTTP 401/403 | `auth` | **QUEUED** for 15 min; auth-failure incident |
| Timeout or broken connection after the request was written; other 5xx; undecodable response | `uncertain` / `decode` | **UNCERTAIN**: never retried automatically |
| Other 4xx | `client` | **REJECTED** |

On startup, invoices left in SUBMITTING by a crash are moved to UNCERTAIN (`RecoverStuckSubmissions`). The worker:

- re-submits due QUEUED invoices;
- opens and closes incidents (FBR unreachable for 10+ minutes; auth failures);
- purges expired sessions;
- writes the daily backup.

## 4. Data model

The schema is in `internal/db/migrations/0001_init.sql`.

**Tables**

- `companies`, `users`, `user_companies`, `sessions`, `api_keys`
- `customers`, `products`, `invoices`, `invoice_items`, `number_series`
- `fbr_calls`, `ref_cache`, `ref_hs_codes`
- `audit_log`, `scenario_runs`, `incidents`, `settings`

**Conventions**

- **Money** is stored as integer **paisa**; calculations use exact decimals (`shopspring/decimal`) and round to 2 decimals per FBR.
- **Uniqueness:** (company, environment, internal number), external reference, and FBR invoice number.
- **Immutability triggers:**
  - accepted or cancelled invoices accept only the transition to CANCELLED, cancellation details, print count and timestamps;
  - their items cannot change;
  - reported invoices cannot be deleted;
  - `audit_log` rows can never be updated or deleted.
- **Invoice seal:** each accepted invoice stores a SHA-256 hash of its canonical content chained to the company's previous seal. The invoice view shows "Seal verified" or "Seal mismatch".
- **Audit chain:** every audit entry stores `hash = SHA-256(previous hash + entry)`. **Verify integrity** recomputes the chain.

## 5. Security

| Area | Design |
|---|---|
| Passwords | bcrypt (cost 12); policy of 8+ characters with letters and digits; lockout for 15 minutes after 5 failures |
| Sessions | Random tokens in an HttpOnly, SameSite cookie with sliding 12-hour expiry; CSRF token required on every state-changing request |
| API keys | `eik_` prefix; only a SHA-256 hash stored; scoped to one company; limited permissions |
| Authorisation | Role → permission map (`internal/httpapi/perms.go`); every company-scoped route checks the user's company access |
| FBR tokens | AES-256-GCM with `master.key` (generated on first start); never returned by the API; never written to logs or the FBR call log |
| Web | Strict security headers and Content Security Policy; uploaded images restricted to image types, SVG without scripts, max 2 MB |
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
| `internal/validate/validate_test.go` | Header and line validation, NTN/CNIC normalisation, debit notes, dates, error catalogue |
| `internal/fbr/client_test.go` | Success, rejection, auth, scenarioId stripping, failure classification (timeout/drop/refused/503), reference APIs, JSON format |
| `internal/db/db_test.go` | Migrations and immutability triggers |
| `internal/service/service_test.go` | Full lifecycle, validation blocking, sandbox scenarios, reconciliation, outage queue and worker, rejection then fix, production token rule, idempotency, fiscal year |
| `internal/license/license_test.go` | Licence states and limits; developer build |
| `internal/printing/*_test.go` | QR version 2 / 25 modules; A4 and thermal rendering; copies; watermarks |
| `internal/httpapi/routes_test.go` | Route registration |

Run all with `make test` (or `GOTOOLCHAIN=local go test ./...`).

## 7. Build

- `web/`: `npm ci && npm run build` writes `internal/webui/dist`. The built bundle is committed so that `go build` works without Node.js.
- Go 1.24: `go build ./cmd/einvoice`. Release builds inject the version, build date and licence public key via `-ldflags -X` (see `Makefile`, `scripts/build-release.sh`).
- Packaging:
  - `packaging/windows/einvoice-suite.iss` (Inno Setup);
  - `packaging/linux/` (systemd unit and installer);
  - `Dockerfile`.
