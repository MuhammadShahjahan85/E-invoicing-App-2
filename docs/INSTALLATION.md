# Installation and Operations Guide

E-Invoicing Suite PK is a single program (`einvoice.exe` on Windows, `einvoice` on Linux). It contains:

- the web application;
- an embedded SQLite database;
- the background worker that reports invoices to FBR;
- an offline FBR simulator.

Users work in a web browser on any computer in the office network.

---

## 1. Requirements

| Item | Minimum | Recommended |
|---|---|---|
| Operating system | Windows 10/11 or Windows Server 2016+ (x64), or Linux x86-64 with systemd | Windows Server 2019+ or Ubuntu 22.04+ |
| CPU / RAM | 2 cores, 2 GB | 4 cores, 4 GB |
| Disk | 2 GB free | SSD. Allow about 1 GB per 100,000 invoices, plus backups. |
| Network | Outbound HTTPS (TCP 443) to `gw.fbr.gov.pk` | A **static public IP**, whitelisted by PRAL for production |
| Clients | Any modern browser (Chrome, Edge, Firefox) | — |
| Printers | Any A4 printer; 80 mm thermal printer for POS receipts | — |

**Time:** the server clock must be correct. Invoice dates and FBR's 72-hour window depend on it. Business dates are always calculated in Pakistan time (Asia/Karachi), whatever the server's time zone.

## 2. Windows installation

### 2.1 Using the installer (recommended)

1. Run `EInvoicingSuitePK-Setup-<version>.exe` as Administrator.
2. Keep **"Allow other computers on the office network…"** ticked if other PCs will use the system. This adds a Windows Firewall rule for TCP 8443.
3. Finish. The installer:
   - registers and starts the Windows service **EInvoicingSuitePK** (automatic start, restarts on failure);
   - opens `https://localhost:8443/`.
4. The browser warns about the certificate once (see §5). Continue, then complete the **setup wizard**:
   - administrator account;
   - business name, NTN/CNIC, province and address exactly as on IRIS;
   - business activity and sector.

Data is stored in `C:\ProgramData\EInvoicingSuitePK`.

### 2.2 Manual installation

Open Command Prompt **as Administrator**:

```bat
mkdir "C:\Program Files\E-Invoicing Suite PK"
copy einvoice.exe "C:\Program Files\E-Invoicing Suite PK\"
cd "C:\Program Files\E-Invoicing Suite PK"
einvoice.exe service install
einvoice.exe service start
netsh advfirewall firewall add rule name="E-Invoicing Suite PK" dir=in action=allow protocol=TCP localport=8443
```

`service install` accepts `--data <folder>` to use a different data folder. The default is `%ProgramData%\EInvoicingSuitePK`.

To try the program without installing a service, run `einvoice.exe serve` in a console. Its data folder is `%ProgramData%\EInvoicingSuitePK`, or the folder given with `--data`.

### 2.3 Service management

```bat
einvoice.exe service stop
einvoice.exe service start
einvoice.exe service uninstall
```

The service can also be managed from `services.msc` ("E-Invoicing Suite PK (FBR Digital Invoicing)").

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
docker build --build-arg VERSION=1.0.0 --build-arg LICENSE_PUBKEY=<base64> -t einvoice-pk .
docker run -d --name einvoice --restart unless-stopped -p 8443:8443 -v einvoice-data:/data einvoice-pk
```

All state lives in the `/data` volume. Back up that volume.

## 5. HTTPS certificate

On first start the server creates a self-signed certificate valid for 10 years. It is stored in `<data>/tls/cert.pem` and `<data>/tls/key.pem` and covers `localhost`, the computer name and its IP addresses. Browsers show a warning once per computer.

To remove the warning, either:

- **trust the certificate:** on each client PC, import `cert.pem` into "Trusted Root Certification Authorities" (Windows: double-click → Install Certificate → Local Machine); or
- **use the company's own certificate:** set `tls.certFile` and `tls.keyFile` in `config.json` to PEM files, then restart the service.

Use `--no-tls` (plain HTTP) only for local testing on `127.0.0.1`.

## 6. Configuration (`config.json`)

The file is created in the data folder on first start. Restart the service after changing it.

| Key | Default | Meaning |
|---|---|---|
| `listen` | `0.0.0.0:8443` | Address and port. `127.0.0.1:8443` restricts access to this computer. |
| `backupDir` | *(empty)* = `<data>/backups` | Backup folder, e.g. a second disk or mapped network drive |
| `tls.enabled` | `true` | Serve HTTPS |
| `tls.certFile`, `tls.keyFile` | *(empty)* = self-signed in `<data>/tls` | Own certificate (PEM) |
| `fbr.endpoints` | FBR v1.12 paths on `https://gw.fbr.gov.pk` | Override only if FBR announces new URLs |
| `fbr.timeoutSeconds` | `30` | Timeout for each FBR call |
| `fbr.cnicThreshold` | `100000` | Warn when an unregistered buyer without CNIC receives an invoice above this value (empty disables) |
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
- `einvoice version`

## 7. Data folder

| Path | Content |
|---|---|
| `einvoice.db` (+ `-wal`, `-shm`) | The database (invoices, masters, audit trail) |
| `master.key` | Key that encrypts the stored FBR tokens. **Keep a secure copy.** |
| `config.json` | Configuration |
| `tls/` | HTTPS certificate and key |
| `logs/einvoice.log` | Application log |
| `backups/` | Automatic and manual backups (`einvoice-<kind>-<YYYYMMDD-HHMMSS>.db`) |

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

- Windows: Apps & Features → E-Invoicing Suite PK → Uninstall. The service and firewall rule are removed. **The data folder is kept on purpose.**
- Linux: `sudo systemctl disable --now einvoice && sudo rm /etc/systemd/system/einvoice.service /opt/einvoice -r`. Keep `/var/lib/einvoice` until records are archived.

## 11. Troubleshooting

| Symptom | Cause and fix |
|---|---|
| Dashboard says "FBR Digital Invoicing is not reachable" | Internet or FBR outage. Invoices are queued and sent automatically. An incident is opened; report it within 24 hours if it continues (Incident register). |
| "FBR rejected the security token" / error 0401 | Wrong token for the environment, expired token, or a token issued for another NTN. Re-enter it under Settings → FBR integration and run **Test connection**. |
| Production calls fail although the token is right | The server's public IP is not whitelisted. Check the static IP with your ISP and ask PRAL to whitelist it. |
| Invoice shows **Needs reconciliation** | FBR may have recorded it, but no definite answer was received. Search IRIS, then use **Reconcile with IRIS** on the invoice. Never re-enter it as a new invoice. |
| Browser certificate warning | Expected with the self-signed certificate; see §5. |
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
