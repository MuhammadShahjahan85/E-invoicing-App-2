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
4. For one-click updates for your clients, also set up the update key (section 8).

Tagged releases fail on purpose if `LICENSE_PUBKEY` is not set, so a developer build is never published by mistake. The `licensegen.exe` from the vendor-tools artifact runs on Windows without Go: `licensegen.exe keygen`, `licensegen.exe issue …`.

### Building on your own computer

Embed the public key when building:

```sh
make release VERSION=1.0.0 LICENSE_PUBKEY=<your base64 public key>
# or
VERSION=1.0.0 LICENSE_PUBKEY=<key> sh scripts/build-release.sh
```

This produces:

- `dist/VeridianEInvoicingPakistan-Setup-1.0.0.exe`, the Windows installer you give to clients;
- `dist/windows-amd64/` with `einvoice.exe` (server) and `VeridianEInvoicing.exe` (desktop launcher);
- `dist/linux-amd64/einvoice` with the Linux install files;
- vendor-only `licensegen` tools in `dist/vendor-tools/`.

The installer is built with NSIS (`sudo apt install nsis` on Ubuntu, or `choco install nsis` on Windows with `makensis` on the PATH). `go install github.com/tc-hib/go-winres@v0.3.3` adds the icon and version details to the programs. The script skips a step whose tool is missing and says so. The GitHub **Release** workflow installs both tools and builds everything for you.

The release script prints a loud warning if `LICENSE_PUBKEY` is empty.

Client packages carry only the client guides: the user guide (PDF) and user manual, installation, FBR onboarding and compliance, mobile access and the ERP integration API. This guide and the architecture document stay with you.

**Code signing (recommended).** Windows SmartScreen warns about installers from an unknown publisher ("Windows protected your PC"). Buy an OV or EV code-signing certificate in your company's name from a certificate authority such as DigiCert, Sectigo or GlobalSign. Then either add it to the Release workflow (secrets `CODESIGN_PFX_BASE64` and `CODESIGN_PASSWORD`; see `.github/workflows/release.yml`), or build locally with `CODESIGN_PFX=cert.pfx CODESIGN_PASSWORD=… sh scripts/build-release.sh` (needs `osslsigncode`). The programs and the installer are then signed and time-stamped, and Windows shows your company as the publisher.

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
  - sending rule 150XA incident letters;
  - displaying the "Integrated with FBR" signboard at each outlet.
- Data ownership: all data stays on the client's premises and belongs to the client. Describe your access during support.
- Limitation of liability: tax treatment decisions remain the client's responsibility; the software implements FBR's published rules.
- Licence terms: no copying to other NTNs, no reverse engineering, transfer rules.

## 7. White-labelling

All product identity lives in `internal/brand/brand.go`:

- `ProductName`, `ShortName`, `Vendor`, `SupportContact`;
- `WindowsServiceName`, `WindowsServiceDisplay`.

Change these, then update the matching values at the top of `packaging/windows/installer.nsi` (`APPNAME`, `COMPANY`, `SUPPORT`, `SERVICE`), the version details in `packaging/windows/winres/*.json`, and the artwork (`node packaging/windows/art/render.cjs` redraws the icon and installer pictures from `web/public/favicon.svg`), then rebuild. Changing `WindowsServiceName` also changes the default data folder on Windows (`%ProgramData%\<WindowsServiceName>`). Decide on it before the first installation.

## 8. Publishing updates (one click for your clients)

Installed copies look for new versions at an **update address** built into them, and install a version only when it carries **your digital signature** (a separate Ed25519 key from the licence key). The update file (`update.json`) lists the version, what is new, and the address, size and SHA-256 of each installer. A copy installs nothing that is not signed with your update key, nothing altered after signing, and never an older version.

By default the update address is the newest **GitHub release** of this repository: `https://github.com/<owner>/<repository>/releases/latest/download/update.json`. Clients' servers need outbound HTTPS to GitHub.

### One-time setup (about 10 minutes)

1. On your own PC, with `licensegen.exe` from the **vendor-tools-keep-private** artifact (or `go run ./cmd/licensegen`):

   ```sh
   licensegen update-keygen -out update-private.key
   ```

   It prints the **update public key**. Keep `update-private.key` secret and backed up, like the licence key: whoever has it can publish updates to all your clients. If it is lost, publish a version with a new key; clients install that version once by hand.
2. GitHub → your repository → **Settings → Secrets and variables → Actions**:
   - **Variables → New repository variable**: `UPDATE_PUBKEY` = the update public key.
   - **Secrets → New repository secret**: `UPDATE_SIGNING_KEY` = the contents of `update-private.key` (open it in Notepad and copy the single line).
3. Build and publish a release (below). Copies installed from it are update-ready. Copies installed from an earlier build (version 1.1.0 or older, or built without `UPDATE_PUBKEY`) cannot update themselves: install the new Setup.exe on them once, over the existing installation; the data is kept.

Optional: to keep this repository private, create a **public** repository just for releases (for example `veridian-einvoicing-updates`, with a README so that it has a commit), add the variable `UPDATES_REPO` = `<owner>/veridian-einvoicing-updates` and the secret `UPDATES_REPO_TOKEN` = a fine-grained token with **Contents: read and write** on that repository. Releases, installers and `update.json` are then published there. To host updates on your own website instead, set the variable `UPDATE_FEED_URL` to the address of your `update.json` and upload the files yourself (see "By hand" below).

### Each release

1. Write what is new in [CHANGELOG.md](../CHANGELOG.md) under a heading `## <version>` (one line per point, starting with `- `). Clients see these lines before and after updating.
2. GitHub → **Actions → Release → Run workflow**: enter the version (e.g. `1.3.0`), tick **Publish**, and optionally type the notes there instead (separate points with `;`) and tick **Required update**. Or push a tag such as `v1.3.0`.
3. The workflow tests the code, builds the Windows installer and the Linux package with your licence and update keys, signs `update.json` and publishes the GitHub release.

Within a few hours every installed copy (that has not turned updates off) shows **Version 1.3.0 is available** to its administrators, who click **Update now**; copies set to automatic install it at their update time. **Check for updates** shows it at once.

**Test first:** publish a version with a suffix, e.g. `1.3.0-rc1`. It becomes a GitHub *pre-release*, which installed copies do not see. On your own test installation, set **Settings → System → Software updates → Advanced: update address** to `https://github.com/<owner>/<repository>/releases/download/v1.3.0-rc1/update.json`, update and test. Then publish `1.3.0`. A version must be higher than the installed one to be offered.

**Required updates** (for example when FBR changes its API): tick **Required update** when running the workflow. Such an update installs at each client's update time (default 02:00 Pakistan time) even where an administrator installs updates by hand. Clients who chose **Don't look for updates** are not reached.

**Support period:** a licence issued with `-support YYYY-MM-DD` only receives updates *released* up to that date; later versions show "renew support with Veridian" instead of **Update now**. Renew by issuing a new licence file with a later support date. To give an update to every client regardless (e.g. a legal change), sign it with `-free` (the workflow does not do this; sign by hand, below).

### By hand (any web server)

```sh
licensegen sign-update -key update-private.key -version 1.3.0 -notes-file notes.txt \
    -url-base https://downloads.example.pk/einvoicing/1.3.0/ \
    -windows VeridianEInvoicingPakistan-Setup-1.3.0.exe -linux einvoice-1.3.0-linux-amd64.tar.gz \
    [-mandatory] [-free] -out update.json
licensegen verify-update -pub <update public key> -in update.json -dir .
```

Upload the installers to the `-url-base` address and `update.json` to the address built into the clients' copies (`UPDATE_FEED_URL`), replacing the previous one. Build with `make release VERSION=1.3.0 LICENSE_PUBKEY=… UPDATE_PUBKEY=… UPDATE_FEED_URL=…`.

### What happens on the client

1. The installation checks the update address every 6 hours (hourly around its update time); administrators see the new version and its notes on the System page and in the notifications bell.
2. **Update now** (or the update time): download, check size, SHA-256 and signature, back up the database, start the installer (Windows: `Setup.exe /S /UPDATE`, which keeps the client's choices of certificate trust, desktop icon and firewall rule; Linux: the update helper). The page follows the progress and reopens on the new version.
3. The new version records **Updated from version X to Y** (with the audit trail entries `system.update.start` and `system.update.done`) and shows every user **What's new** once.
4. If anything fails, the previous version keeps running (the Windows installer starts it again) and the System page says what happened; the backup made before the attempt is listed under Backups.

**Before publishing**, test the upgrade on a copy of a client database: restore a backup on a test machine, update it, then check invoices, reports and **Verify integrity**.

## 9. Support playbook

| Request | First checks |
|---|---|
| "Invoices not going to FBR" | Dashboard connection card; Reports → FBR API log; token expiry; IP whitelist; internet |
| "FBR error XXXX" | Help & error codes; the invoice's FBR call log tab (request/response) |
| "Needs reconciliation" | Guide the client through IRIS search and **Reconcile with IRIS** |
| "Cannot log in" | Lockout (15 minutes after 5 failures); `einvoice reset-password` |
| "Server moved / restored" | Settings → System → Backups → **Upload a backup** / **Restore**; keep `master.key`; re-enter tokens if the key was lost |
| "The update did not install" | Settings → System → Software updates shows why; **Server log** (download it); outbound HTTPS to GitHub; the copy must be update-ready (see section 8) |
| "Licence warning" | Settings → System; issue a renewal |

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
