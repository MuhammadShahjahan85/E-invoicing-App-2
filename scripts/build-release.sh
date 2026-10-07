#!/bin/sh
# Builds release binaries of Veridian E-invoicing Pakistan for Windows and Linux (x86-64).
#
# Environment:
#   VERSION         product version (default 1.0.0)
#   BUILD_DATE      build date (default: today, UTC)
#   LICENSE_PUBKEY  base64 Ed25519 public key from `go run ./cmd/licensegen keygen`
#                   (empty = developer build without licence enforcement)
#   SKIP_WEB=1      reuse the web UI already built in internal/webui/dist
#
# Output:
#   dist/windows-amd64/einvoice.exe   (input for packaging/windows/einvoice-suite.iss)
#   dist/linux-amd64/einvoice         (+ packaging/linux files)
#   dist/vendor-tools/<os>-amd64/licensegen  (vendor only — never give to customers)
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

for os in windows linux; do
	ext=""
	[ "$os" = "windows" ] && ext=".exe"
	out="dist/$os-amd64"
	mkdir -p "$out" "dist/vendor-tools/$os-amd64"
	echo "Building $out/einvoice$ext (version $VERSION)..."
	GOOS=$os GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$out/einvoice$ext" ./cmd/einvoice
	GOOS=$os GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "dist/vendor-tools/$os-amd64/licensegen$ext" ./cmd/licensegen
	cp README.md LICENSE THIRD-PARTY-NOTICES.txt "$out/"
	mkdir -p "$out/docs"
	cp docs/*.md "$out/docs/" 2>/dev/null || true
done
cp packaging/linux/einvoice.service packaging/linux/install.sh dist/linux-amd64/

# Archives (optional tools).
if command -v tar >/dev/null 2>&1; then
	tar -C dist -czf "dist/einvoice-$VERSION-linux-amd64.tar.gz" linux-amd64
fi
if command -v zip >/dev/null 2>&1; then
	(cd dist && zip -qr "einvoice-$VERSION-windows-amd64.zip" windows-amd64)
fi

echo
echo "Done. Next steps:"
printf '%s\n' "  Windows installer: ISCC.exe /DAppVersion=$VERSION packaging\\windows\\einvoice-suite.iss"
echo "  Linux:             copy dist/linux-amd64 to the server and run: sudo sh install.sh ./einvoice"
