# Licensing and Sales Guide (Vendor)

This guide is for you as the **vendor**: the practitioner or firm that sells, installs and supports Veridian E-invoicing Pakistan. It covers product licensing, building releases, issuing licences to clients, and a commercial model.

---

## 1. How licensing works

Licences are JSON files signed with an **Ed25519** private key that only you hold. Each program build embeds the matching **public key** and verifies the signature offline. No internet licence server is needed.

| State | When | Effect |
|---|---|---|
| **Developer** | The program was built **without** a public key | No licence checks at all. For your own development and demos only — never give such builds to clients. |
| **Trial** | No licence installed; first **30 days** after first start | Training simulator and FBR sandbox work (scenario testing). **Production reporting is blocked.** |
| **Trial expired** | After 30 days without a licence | Production stays blocked; install a licence |
| **Licensed** | Valid licence installed | Production allowed for the **seller NTN/CNICs listed in the licence** (or `*` for any) |
| **Grace** | Licence expiry date passed, within **15 days** | Production continues; a renewal warning is shown |
| **Expired** | More than 15 days after expiry | New production invoices cannot be submitted until renewal. Invoices already issued and queued (for example during an FBR outage) are still reported, because the law requires it. Existing data stays accessible. |
| **Invalid** | Signature check fails | Treated as unlicensed |

Limits in the licence:

- **companies:** maximum number of companies (NTNs) in the installation;
- **users:** maximum number of user accounts;
- **0** means unlimited.

A licence without an expiry date is **perpetual**. The optional **support** date records how long support and updates are covered.

## 2. One-time setup: your key pair

On your own (offline, secure) computer:

```sh
licensegen keygen -out vendor-private.key        # or: go run ./cmd/licensegen keygen -out vendor-private.key
```

It prints the **public key** (base64). Keep `vendor-private.key` secret and backed up (e.g. an encrypted USB plus a safe copy). Anyone with this file can create licences. If it is lost, you cannot issue licences that existing builds accept.

## 3. Building client releases

### Easiest: let GitHub build them (no Go or Node.js needed on your PC)

1. On GitHub, open your repository → **Settings → Secrets and variables → Actions → Variables → New repository variable**.
   - Name: `LICENSE_PUBKEY`
   - Value: the public key printed by `licensegen keygen`.
2. Open the **Actions** tab → **Release** → **Run workflow**, and enter the version (e.g. `1.0.0`). Alternatively, push a tag such as `v1.0.0`; tags also create a GitHub Release.
3. When the run finishes, download the artifacts:
   - **windows-installer** (`VeridianEInvoicingPakistan-Setup-<version>.exe`);
   - **linux-package**;
   - **vendor-tools-keep-private** (licensegen).

Tagged releases fail on purpose if `LICENSE_PUBKEY` is not set, so a developer build is never published by mistake. The `licensegen.exe` from the vendor-tools artifact runs on Windows without Go: `licensegen.exe keygen`, `licensegen.exe issue …`.

### Building on your own computer

Embed the public key when building:

```sh
make release VERSION=1.0.0 LICENSE_PUBKEY=<your base64 public key>
# or
VERSION=1.0.0 LICENSE_PUBKEY=<key> sh scripts/build-release.sh
```

This produces:

- `dist/windows-amd64/einvoice.exe`;
- `dist/linux-amd64/einvoice` with the Linux install files;
- vendor-only `licensegen` tools in `dist/vendor-tools/`.

Then build the Windows installer with Inno Setup 6:

```bat
ISCC.exe /DAppVersion=1.0.0 packaging\windows\einvoice-suite.iss
```

The release script prints a loud warning if `LICENSE_PUBKEY` is empty.

Before your first release, set your company name and support contact in `internal/brand/brand.go` (`Vendor`, `SupportContact`). They appear in the UI.

## 4. Issuing a licence to a client

```sh
go run ./cmd/licensegen issue -key vendor-private.key \
  -licensee "Indus Steel & Trading (Pvt) Ltd" \
  -ntn 1234567 \
  -expires 2027-10-31 \
  -support 2027-10-31 \
  -companies 1 -users 5 \
  -edition standard \
  -out indus-steel.lic
```

| Flag | Meaning | Default |
|---|---|---|
| `-key` | Your private key file | `vendor-private.key` |
| `-licensee` | Client's legal name (shown in the product) | *(required)* |
| `-ntn` | Comma-separated seller NTN/CNICs allowed to report in production, or `*` for any | *(required)* |
| `-expires` | Expiry date `YYYY-MM-DD`; empty = perpetual | empty |
| `-support` | Support/updates valid until `YYYY-MM-DD` | empty |
| `-companies` | Maximum companies (0 = unlimited) | 1 |
| `-users` | Maximum users (0 = unlimited) | 5 |
| `-edition` | Edition name, e.g. `standard`, `professional`, `enterprise` | `standard` |
| `-id` | Licence id | generated |
| `-out` | Output file | `license.lic` |

To check a licence file:

```sh
go run ./cmd/licensegen verify -pub <base64 public key> -in indus-steel.lic
```

**Installing:** the client's administrator opens **Settings → System → Install licence file (license.lic)** and selects the file. The licence is stored in the database. **Renewal:** issue a new file with a later expiry and install it the same way.

## 5. Suggested commercial model

Prices depend on your market. Fill in this worksheet rather than quoting fixed figures.

| Component | Basis | Your price (PKR) |
|---|---|---|
| Software licence | Per NTN (company), one-time or annual | |
| Additional company (NTN) | Per NTN | |
| Additional users | Per block of 5 users | |
| Implementation | Installation, setup wizard, masters import | |
| FBR onboarding service | IRIS registration support, sandbox scenarios, production token, go-live | |
| ERP/POS integration | Per integration (API key setup, mapping, testing) | |
| Training | Per session / per user | |
| Annual support & updates (AMC) | % of licence or fixed per year; includes updates for FBR changes | |
| On-site visits | Per visit | |

Typical editions:

- **Standard:** 1 company, 5 users.
- **Professional:** up to 3 companies, 15 users, ERP API.
- **Enterprise:** unlimited companies and users, priority support.

## 6. Client contract checklist

- Scope: software licence, editions/limits, number of NTNs.
- Support SLA: response times, hours, remote access method, on-site visits.
- **Updates for FBR changes:** how quickly specification and rate changes are delivered under AMC.
- Client responsibilities:
  - IRIS registration, tokens, static IP;
  - server hardware and OS, antivirus;
  - **off-site backups and six-year retention**;
  - accuracy of master data (HS codes, rates, SROs);
  - sending Rule 150R letters.
- Data ownership: all data stays on the client's premises and belongs to the client. Describe your access during support.
- Limitation of liability: tax treatment decisions remain the client's responsibility; the software implements FBR's published rules.
- Licence terms: no copying to other NTNs, no reverse engineering, transfer rules.

## 7. White-labelling

All product identity lives in `internal/brand/brand.go`:

- `ProductName`, `ShortName`, `Vendor`, `SupportContact`;
- `WindowsServiceName`, `WindowsServiceDisplay`.

Change these, then update the matching values in `packaging/windows/einvoice-suite.iss` (`AppName`, `AppPublisher`, `ServiceName`) and rebuild. Changing `WindowsServiceName` also changes the default data folder on Windows (`%ProgramData%\<WindowsServiceName>`). Decide on it before the first installation.

## 8. Release and update process

1. Update the code (FBR changes, fixes), run `make test`, and update the docs.
2. Tag a version, e.g. `v1.1.0`. Build with `make release VERSION=1.1.0 LICENSE_PUBKEY=…` and compile the installer.
3. Test the upgrade on a copy of a client database: restore a backup on a test machine, run the new version, then check invoices, reports and **Verify integrity**.
4. Distribute the installer. Clients take a backup, then run the installer (data is kept; migrations run automatically).

## 9. Support playbook

| Request | First checks |
|---|---|
| "Invoices not going to FBR" | Dashboard connection card; Reports → FBR API log; token expiry; IP whitelist; internet |
| "FBR error XXXX" | Help & error codes; the invoice's FBR call log tab (request/response) |
| "Needs reconciliation" | Guide the client through IRIS search and **Reconcile with IRIS** |
| "Cannot log in" | Lockout (15 minutes after 5 failures); `einvoice reset-password` |
| "Server moved / restored" | Restore the procedure from backups; keep `master.key`; re-enter tokens if the key was lost |
| "Licence warning" | Settings → System; issue a renewal |

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
