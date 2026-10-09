# Installation and Operations Guide

Veridian E-invoicing Pakistan is a single program (`einvoice.exe` on Windows, `einvoice` on Linux). It contains:

- the web application;
- an embedded SQLite database;
- the background worker that reports invoices to FBR;
- an offline FBR simulator.

On Windows it runs in the background as a Windows service. The desktop icon opens it in a window of its own (`VeridianEInvoicing.exe`, the desktop launcher), so nobody needs a command window. Other computers and phones in the office use it in a web browser.

---

## 1. Requirements

| Item | Minimum | Recommended |
|---|---|---|
| Operating system | Windows 10/11 or Windows Server 2016+ (x64), or Linux x86-64 with systemd | Windows Server 2019+ or Ubuntu 22.04+ |
| CPU / RAM | 2 cores, 2 GB | 4 cores, 4 GB |
| Disk | 2 GB free | SSD. Allow about 1 GB per 100,000 invoices, plus backups. |
| Network | Outbound HTTPS (TCP 443) to `gw.fbr.gov.pk` | A **static public IP**, whitelisted by PRAL for production. For e-mail notifications, outbound access to the mail server (usually TCP 587 or 465). |
| Clients | Any modern browser (Chrome, Edge, Firefox) | — |
| Printers | Any A4 printer; 80 mm thermal printer for POS receipts | — |

**Time:** the server clock must be correct. Invoice dates and FBR's 72-hour window depend on it. Business dates are always calculated in Pakistan time (Asia/Karachi), whatever the server's time zone.

## 2. Windows installation

### 2.1 Using the installer (recommended)

1. Double-click `VeridianEInvoicingPakistan-Setup-<version>.exe` and allow it to make changes (it needs administrator rights).
   If Windows shows **"Windows protected your PC"**, click **More info → Run anyway**. This appears for installers that are not code-signed; see §11.
2. Read and accept the licence.
3. Choose the components. The recommended choices suit most offices:

   | Component | What it does |
   |---|---|
   | **Veridian E-invoicing Pakistan** (always) | The program, the background service and the guides |
   | **Trust the security certificate** | Edge and Chrome on this computer open the app without a certificate warning (§5) |
   | **Desktop icon** | An icon that opens the app in its own window |
   | **Office network access** | A Windows Firewall rule that lets computers and phones on the local network reach the app (TCP 8443, local subnet only) |

4. Keep the suggested folder (`C:\Program Files\Veridian E-invoicing Pakistan`) and click **Install**. Setup:
   - prepares the data folder `C:\ProgramData\VeridianEInvoicingPakistan` and its HTTPS certificates;
   - registers and starts the Windows service **VeridianEInvoicingPakistan** (automatic start, restarts on failure);
   - adds the Start menu entries **Veridian E-invoicing Pakistan** and **Help and error codes**, and the entry in **Settings → Apps**.
5. On the last page, keep **Open Veridian E-invoicing Pakistan now** ticked and click **Finish**. Complete the **setup wizard**:
   - administrator account;
   - business name, NTN/CNIC, province and address exactly as on IRIS;
   - business activity and sector.

From then on, open the app with the **desktop icon** or from the **Start menu**. The icon starts the service if it is not running and opens the app in a window of its own (Microsoft Edge or Google Chrome "app" window, without tabs or an address bar). If neither browser is installed, it opens in the default browser.

**Silent installation** (for IT staff): `VeridianEInvoicingPakistan-Setup-<version>.exe /S` installs all recommended components. Add `/D=<folder>` (last, without quotes) to choose the program folder.

### 2.2 Manual installation

Open Command Prompt **as Administrator**:

```bat
mkdir "C:\Program Files\Veridian E-invoicing Pakistan"
copy einvoice.exe VeridianEInvoicing.exe "C:\Program Files\Veridian E-invoicing Pakistan\"
cd "C:\Program Files\Veridian E-invoicing Pakistan"
einvoice.exe tls init
einvoice.exe tls trust
einvoice.exe service install
einvoice.exe service start
einvoice.exe firewall allow
```

- `tls init` creates the data folder and HTTPS certificates; `tls trust` makes this computer's browsers trust them.
- `service install` registers the service (or updates it after an upgrade) and lets signed-in users start it from the desktop icon.
- `firewall allow` lets computers on the local network reach the configured port.
- Each command accepts `--data <folder>` to use a different data folder. The default is `%ProgramData%\VeridianEInvoicingPakistan`.

Create a shortcut to `VeridianEInvoicing.exe` on the desktop to open the app without a command window.

To try the program without installing a service, run `einvoice.exe serve` in a console. Its data folder is `%ProgramData%\VeridianEInvoicingPakistan`, or the folder given with `--data`.

### 2.3 Service management

```bat
einvoice.exe service stop
einvoice.exe service start
einvoice.exe service uninstall
```

The service can also be managed from `services.msc` ("Veridian E-invoicing Pakistan (FBR Digital Invoicing)"). Signed-in users may start it (the desktop icon does so when needed); stopping, changing or removing it needs an administrator.

## 3. Linux installation (systemd)

Copy the Linux release folder (`einvoice`, `einvoice.service`, `install.sh`) to the server, then run:

```sh
sudo sh install.sh ./einvoice
```

The script:

- creates the system user `einvoice`;
- installs the binary to `/opt/einvoice/einvoice` and stores data in `/var/lib/einvoice`;
- installs and starts the `einvoice` systemd unit.

Re-run it to upgrade. Data is kept.

```sh
sudo systemctl status einvoice
sudo journalctl -u einvoice -f
sudo systemctl restart einvoice
```

Open TCP 8443 in the server firewall if other computers need access (e.g. `sudo ufw allow 8443/tcp`).

## 4. Docker

```sh
docker build --build-arg VERSION=1.0.0 --build-arg LICENSE_PUBKEY=<base64> -t veridian-einvoicing .
docker run -d --name einvoice --restart unless-stopped -p 8443:8443 -v einvoice-data:/data veridian-einvoicing
```

All state lives in the `/data` volume. Back up that volume.

## 5. HTTPS certificate

On first start the server creates its own **certificate authority** (`<data>/tls/ca.pem`, valid 10 years). It uses it to issue the server certificate (`<data>/tls/cert.pem`). That certificate covers `localhost`, the computer name, every IP address of the server and any names listed in `tls.hosts`. It lasts 800 days, is renewed automatically before expiry, and is re-issued when the server's addresses change.

Browsers warn until a device trusts the CA. On the server computer the installer trusts it for you (component **Trust the security certificate**; manually: `einvoice tls trust`). For the other devices (and to install the app on phones):

- **Trust the CA on each device (recommended):** download it from **Mobile app & access → Download certificate** (or `https://<server>:8443/api/v1/system/ca.crt`) and install it as a trusted root. Steps for Windows, macOS, Android and iPhone are in [MOBILE-AND-REMOTE-ACCESS.md](MOBILE-AND-REMOTE-ACCESS.md). Distribute only `ca.pem`; keep `ca-key.pem` secret.
- **Use the company's own certificate:** set `tls.certFile` and `tls.keyFile` in `config.json` to PEM files, then restart the service.

Use `--no-tls` (plain HTTP) only for local testing on `127.0.0.1`.

## 6. Configuration (`config.json`)

The file is created in the data folder on first start. Restart the service after changing it.

| Key | Default | Meaning |
|---|---|---|
| `listen` | `0.0.0.0:8443` | Address and port. `127.0.0.1:8443` restricts access to this computer. |
| `backupDir` | *(empty)* = `<data>/backups` | Backup folder, e.g. a second disk or mapped network drive |
| `tls.enabled` | `true` | Serve HTTPS |
| `tls.certFile`, `tls.keyFile` | *(empty)* = issued by the installation's local CA in `<data>/tls` | Own certificate (PEM) |
| `tls.hosts` | *(empty)* | Extra DNS names or IP addresses users reach the server by (VPN address, office DNS name); added to the local certificate |
| `fbr.endpoints` | FBR v1.12 paths on `https://gw.fbr.gov.pk` | Override only if FBR announces new URLs |
| `fbr.endpoints.cancelPath`, `fbr.endpoints.cancelSandboxPath` | *(empty: cancel on IRIS)* | FBR cancellation service, e.g. `/di_data/v1/di/cancelinvoicedata` and `/di_data/v1/di/cancelinvoicedata_sb`. Set these only after PRAL confirms the request format; the cancel dialog then offers **Cancel with FBR** |
| `fbr.timeoutSeconds` | `30` | Timeout for each FBR call |
| `fbr.cnicThreshold` | `100000` | Warn when an unregistered buyer without CNIC receives an invoice above this value (empty disables). Manufacturers and importers are always warned, whatever the value (s.23(1)(b)) |
| `worker.intervalSeconds` | `20` | How often the background worker processes the queue |
| `worker.backupHour` | `23` | Hour (0–23, Pakistan time) for the automatic daily backup; `-1` disables it. The server must be running at that hour. |
| `worker.backupRetention` | `30` | Number of automatic backups kept |
| `logLevel` | `info` | Log detail |

**Environment variables**

- `EINV_DATA_DIR` sets the data folder.
- `EINV_LISTEN` overrides `listen`.

**Command-line flags**

- `einvoice serve [--data DIR] [--listen ADDR] [--no-tls]`
- `einvoice backup [--data DIR] [--out FILE]`
- `einvoice reset-password --user NAME --password NEW [--data DIR]`
- `einvoice mock-fbr [--listen 127.0.0.1:9090]` — offline FBR simulator for ERP developers
- `einvoice service install|uninstall|start|stop [--data DIR]` — Windows service (as Administrator)
- `einvoice tls init|trust|untrust [--data DIR]` — create the HTTPS certificates; add or remove the local CA in the Windows trusted root store
- `einvoice firewall allow|remove [--data DIR]` — Windows Firewall rule for the local network
- `einvoice version`

## 7. Data folder

| Path | Content |
|---|---|
| `einvoice.db` (+ `-wal`, `-shm`) | The database (invoices, masters, audit trail) |
| `master.key` | Key that encrypts the stored FBR tokens. **Keep a secure copy.** |
| `config.json` | Configuration |
| `tls/` | Local certificate authority (`ca.pem`, `ca-key.pem` — keep the key secret) and the server certificate (`cert.pem`, `key.pem`) |
| `logs/einvoice.log` | Application log |
| `backups/` | Automatic and manual backups (`einvoice-<kind>-<YYYYMMDD-HHMMSS>.db`) |

On Windows the installer restricts the folder to Windows itself (SYSTEM) and administrators; other users can read only `config.json` and `tls\ca.pem`. Open it as an administrator (Explorer asks for permission the first time).

## 8. Backups and restore

- **Automatic:** every day at `worker.backupHour` (default 23:00). The latest `worker.backupRetention` copies are kept.
- **Manual:** Settings → System → **Back up now** (with download), or `einvoice backup` on the command line.
- **Off-site:** copy the backups folder and `master.key` to another disk or cloud storage at least weekly. Records must be kept for six years (rule 150S).

**Restore**

1. Stop the service (`einvoice service stop` or `systemctl stop einvoice`).
2. In the data folder, rename `einvoice.db` and delete `einvoice.db-wal` and `einvoice.db-shm`, if present.
3. Copy the backup file to `einvoice.db`.
4. Make sure the original `master.key` is in the data folder. Without it, FBR tokens must be re-entered; invoices remain readable.
5. Start the service. Check Settings → System and the Audit trail (**Verify integrity**).

## 9. Upgrades

1. Take a manual backup.
2. Windows: run the new installer; it stops the service, replaces the program and starts it again. Linux: re-run `install.sh` with the new binary. Docker: rebuild the image and recreate the container with the same volume.
3. Database migrations run automatically on start.

## 10. Uninstall

- Windows: Settings → Apps → Veridian E-invoicing Pakistan → Uninstall. The service, firewall rule, trusted certificate and shortcuts are removed. **The data folder is kept on purpose**; installing again picks it up automatically.
- Linux: `sudo systemctl disable --now einvoice && sudo rm /etc/systemd/system/einvoice.service /opt/einvoice -r`. Keep `/var/lib/einvoice` until records are archived.

## 11. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Dashboard says "FBR Digital Invoicing is not reachable" | Internet or FBR outage. Invoices are queued and sent automatically. An incident is opened; report it within 24 hours if it continues (Incident register). |
| "FBR rejected the security token" / error 0401 | Wrong token for the environment, expired token, or a token issued for another NTN. Re-enter it under Settings → FBR integration and run **Test connection**. |
| Production calls fail although the token is right | The server's public IP is not whitelisted. Check the static IP with your ISP and ask PRAL to whitelist it. |
| Invoice shows **Needs reconciliation** | FBR may have recorded it, but no definite answer was received. Search IRIS, then use **Reconcile with IRIS** on the invoice. Never re-enter it as a new invoice. |
| Browser certificate warning | The device does not trust the server's certificate yet; see §5. On the server computer, run `einvoice tls trust` as Administrator. |
| "Windows protected your PC" when running Setup | Windows SmartScreen shows this for installers that are not code-signed. Click **More info → Run anyway**. Release builds signed with the company's code-signing certificate show the company as publisher instead (see LICENSING-AND-SALES.md). |
| Desktop icon: "Windows could not start the … service" | Open `services.msc`, start **Veridian E-invoicing Pakistan (FBR Digital Invoicing)** and read `logs\einvoice.log` in the data folder. Re-running Setup repairs the installation. |
| Desktop icon opens a normal browser tab | Neither Microsoft Edge nor Google Chrome was found; the default browser is used instead. |
| "address already in use" in the log | Another program uses port 8443. Change `listen` in `config.json`, and the firewall rule. |
| Wrong dates on invoices | Correct the server clock (enable automatic time synchronisation). |
| Forgotten admin password | `einvoice reset-password --user admin --password NewPass123`. Add `--data` if the data folder is not the default. The user must change the password at next login. |
| Licence messages | Settings → System shows the licence state. Production needs a valid licence for the company's NTN. |

## 12. Security checklist

- Install on a dedicated, patched server with antivirus and disk encryption.
- Allow only office computers to reach port 8443 (firewall); do not expose it to the internet.
- Give each person their own account with the least role needed. Remove leavers promptly.
- Store copies of `master.key` and backups securely, separately from the server.
- Replace the self-signed certificate with a trusted one where possible.
- Review the Audit trail periodically and run **Verify integrity**.

---

© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software.
