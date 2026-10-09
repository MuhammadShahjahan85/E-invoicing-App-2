#!/bin/sh
# Builds release binaries of Veridian E-invoicing Pakistan for Windows and Linux (x86-64).
#
# Environment:
#   VERSION         product version (default 1.0.0)
#   BUILD_DATE      build date (default: today, UTC)
#   LICENSE_PUBKEY  base64 Ed25519 public key from `go run ./cmd/licensegen keygen`
#                   (empty = developer build without licence enforcement)
#   SKIP_WEB=1      reuse the web UI already built in internal/webui/dist
#   CODESIGN_PFX, CODESIGN_PASSWORD
#                   code-signing certificate (.pfx) and its password: signs the
#                   Windows programs and installer with osslsigncode
#
# Output:
#   dist/VeridianEInvoicingPakistan-Setup-<VERSION>.exe   Windows installer
#   dist/windows-amd64/einvoice.exe             server and Windows service
#   dist/windows-amd64/VeridianEInvoicing.exe   desktop launcher (no command window)
#   dist/linux-amd64/einvoice                   (+ packaging/linux files)
#   dist/vendor-tools/<os>-amd64/licensegen     vendor only: never give to customers
#
# Tools: makensis (NSIS 3; "apt install nsis") builds the installer, and
# go-winres ("go install github.com/tc-hib/go-winres@v0.3.3") gives the Windows
# programs their icon and version details. Without them those steps are skipped.
set -eu

cd "$(dirname "$0")/.."

VERSION="${VERSION:-1.0.0}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%d)}"
LICENSE_PUBKEY="${LICENSE_PUBKEY:-}"

if [ -z "$LICENSE_PUBKEY" ]; then
	echo "********************************************************************" >&2
	echo "WARNING: LICENSE_PUBKEY is empty. This is a DEVELOPER build with NO" >&2
	echo "licence enforcement. Do not give it to customers." >&2
	echo "Create a key pair once with:  go run ./cmd/licensegen keygen" >&2
	echo "********************************************************************" >&2
fi

if [ "${SKIP_WEB:-0}" != "1" ]; then
	echo "Building web UI..."
	(cd web && npm ci && npm run build)
fi

LDFLAGS="-s -w -X einvoicing/internal/brand.Version=$VERSION -X einvoicing/internal/brand.BuildDate=$BUILD_DATE -X einvoicing/internal/license.PublicKeyB64=$LICENSE_PUBKEY"
export GOTOOLCHAIN=local CGO_ENABLED=0
# Windows wants a numeric a.b.c.d version: 1.2.3-rc1 -> 1.2.3.0.
NUMVER=$(echo "$VERSION" | sed 's/[-+].*//' | awk -F. '{printf "%d.%d.%d.%d", $1, $2, $3, $4}')

# Icon, version details and manifest of the Windows programs.
WINRES=$(command -v go-winres || true)
[ -z "$WINRES" ] && [ -x "$(go env GOPATH)/bin/go-winres" ] && WINRES="$(go env GOPATH)/bin/go-winres"
if [ -n "$WINRES" ]; then
	trap 'rm -f cmd/einvoice/rsrc_windows_amd64.syso cmd/launcher/rsrc_windows_amd64.syso' EXIT
	"$WINRES" make --in packaging/windows/winres/einvoice.json --out cmd/einvoice/rsrc --arch amd64 --product-version "$NUMVER" --file-version "$NUMVER"
	"$WINRES" make --in packaging/windows/winres/launcher.json --out cmd/launcher/rsrc --arch amd64 --product-version "$NUMVER" --file-version "$NUMVER"
else
	echo "note: go-winres not found: the Windows programs get no icon or version details" >&2
fi

# sign signs Windows programs when a code-signing certificate is given.
sign() {
	[ -n "${CODESIGN_PFX:-}" ] || return 0
	for f in "$@"; do
		osslsigncode sign -pkcs12 "$CODESIGN_PFX" -pass "${CODESIGN_PASSWORD:-}" -h sha256 \
			-n "Veridian E-invoicing Pakistan" -ts http://timestamp.digicert.com \
			-in "$f" -out "$f.signed"
		mv "$f.signed" "$f"
	done
}

for os in windows linux; do
	ext=""
	[ "$os" = "windows" ] && ext=".exe"
	out="dist/$os-amd64"
	mkdir -p "$out" "dist/vendor-tools/$os-amd64"
	echo "Building $out/einvoice$ext (version $VERSION)..."
	GOOS=$os GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$out/einvoice$ext" ./cmd/einvoice
	GOOS=$os GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "dist/vendor-tools/$os-amd64/licensegen$ext" ./cmd/licensegen
	if [ "$os" = "windows" ]; then
		echo "Building $out/VeridianEInvoicing.exe (desktop launcher)..."
		GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui" -o "$out/VeridianEInvoicing.exe" ./cmd/launcher
		sign "$out/einvoice.exe" "$out/VeridianEInvoicing.exe"
	fi
	cp LICENSE THIRD-PARTY-NOTICES.txt "$out/"
	if [ "$os" = "windows" ]; then cp packaging/windows/README-client.txt "$out/README.txt"; else cp README.md "$out/"; fi
	# Client guides only: the vendor guides (licensing, architecture) stay in the repository.
	mkdir -p "$out/docs"
	for d in USER-MANUAL INSTALLATION FBR-ONBOARDING-GUIDE FBR-COMPLIANCE-GUIDE MOBILE-AND-REMOTE-ACCESS ERP-INTEGRATION-API; do
		cp "docs/$d.md" "$out/docs/"
	done
	cp docs/Veridian-E-invoicing-Pakistan-User-Guide.pdf "$out/docs/" 2>/dev/null || true
done
cp packaging/linux/einvoice.service packaging/linux/install.sh dist/linux-amd64/

# Windows installer.
SETUP="dist/VeridianEInvoicingPakistan-Setup-$VERSION.exe"
if command -v makensis >/dev/null 2>&1; then
	echo "Building $SETUP..."
	makensis -WX -V2 -DVERSION="$VERSION" -DNUMVER="$NUMVER" packaging/windows/installer.nsi
	sign "$SETUP"
else
	echo "note: makensis not found: the Windows installer was not built (apt install nsis)" >&2
	SETUP=""
fi

# Archives (optional tools).
if command -v tar >/dev/null 2>&1; then
	tar -C dist -czf "dist/einvoice-$VERSION-linux-amd64.tar.gz" linux-amd64
fi
if command -v zip >/dev/null 2>&1; then
	(cd dist && zip -qr "einvoice-$VERSION-windows-amd64.zip" windows-amd64)
fi

echo
echo "Done."
[ -n "$SETUP" ] && echo "  Windows: give clients $SETUP (double-click to install)"
echo "  Linux:   copy dist/linux-amd64 to the server and run: sudo sh install.sh ./einvoice"
