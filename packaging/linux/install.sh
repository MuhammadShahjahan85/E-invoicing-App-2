#!/bin/sh
# Installs (or upgrades) E-Invoicing Suite PK as a systemd service on Linux x86-64.
#
#   sudo sh install.sh [path/to/einvoice]
#
# Binary: /opt/einvoice/einvoice   Data: /var/lib/einvoice   Service: einvoice
# Re-running the script upgrades the binary and keeps all data.
set -eu

BIN_SRC="${1:-./einvoice}"
PREFIX=/opt/einvoice
DATA=/var/lib/einvoice
UNIT=/etc/systemd/system/einvoice.service
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)

if [ "$(id -u)" -ne 0 ]; then
	echo "Please run as root, e.g.: sudo sh install.sh ./einvoice" >&2
	exit 1
fi
if [ ! -f "$BIN_SRC" ]; then
	echo "Binary not found: $BIN_SRC" >&2
	exit 1
fi
if [ ! -f "$SCRIPT_DIR/einvoice.service" ]; then
	echo "einvoice.service must be next to install.sh" >&2
	exit 1
fi

if ! id einvoice >/dev/null 2>&1; then
	NOLOGIN=/usr/sbin/nologin
	[ -x "$NOLOGIN" ] || NOLOGIN=/sbin/nologin
	useradd --system --user-group --home-dir "$DATA" --shell "$NOLOGIN" einvoice
fi

install -d -m 0755 "$PREFIX"
install -d -m 0750 -o einvoice -g einvoice "$DATA"

if systemctl is-active --quiet einvoice 2>/dev/null; then
	echo "Stopping the running service for upgrade..."
	systemctl stop einvoice
fi
install -m 0755 "$BIN_SRC" "$PREFIX/einvoice"
[ -f "$SCRIPT_DIR/README.md" ] && install -m 0644 "$SCRIPT_DIR/README.md" "$PREFIX/README.md"
install -m 0644 "$SCRIPT_DIR/einvoice.service" "$UNIT"

systemctl daemon-reload
systemctl enable einvoice >/dev/null
systemctl start einvoice

IP=$(hostname -I 2>/dev/null | awk '{print $1}')
[ -n "$IP" ] || IP=localhost
echo
echo "E-Invoicing Suite PK is running."
echo "  Open:  https://$IP:8443/   (accept the self-signed certificate once, then complete the setup wizard)"
echo "  Data:  $DATA  — back it up regularly, including master.key"
echo "  Logs:  journalctl -u einvoice  and  $DATA/logs/einvoice.log"
