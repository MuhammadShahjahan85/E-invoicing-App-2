#!/bin/sh
# Regenerates THIRD-PARTY-NOTICES.txt from the Go modules linked into the
# server binary and the npm packages bundled into the web UI.
# Run after changing dependencies:  sh scripts/third-party-notices.sh
set -eu
cd "$(dirname "$0")/.."
OUT=THIRD-PARTY-NOTICES.txt
export GOTOOLCHAIN=local
MODCACHE=$(go env GOMODCACHE)

{
	echo "Veridian E-invoicing PK — Third-party notices"
	echo
	echo "Veridian E-invoicing PK is proprietary software of Veridian Partners Consultancy"
	echo "Private Limited (see LICENSE). It includes the open-source components below, each"
	echo "used under its own licence, reproduced in full."
	echo
} >"$OUT"

license_file() {
	for f in LICENSE LICENSE.txt LICENSE.md LICENCE COPYING LICENSE-MIT; do
		if [ -f "$1/$f" ]; then
			echo "$1/$f"
			return
		fi
	done
}

# The Go standard library and runtime are compiled into the binary.
{
	echo "================================================================================"
	echo "Go standard library and runtime ($(go env GOVERSION))"
	echo "================================================================================"
	cat "$(go env GOROOT)/LICENSE"
	echo
} >>"$OUT"

# Go modules compiled into the server (Windows build includes golang.org/x/sys/windows).
for os in linux windows; do
	GOOS=$os go list -deps -f '{{if .Module}}{{if not .Module.Main}}{{.Module.Path}} {{.Module.Version}}{{end}}{{end}}' ./cmd/einvoice
done | sort -u | while read -r path version; do
	dir="$MODCACHE/$(echo "$path" | sed 's/[A-Z]/!\L&/g')@$version"
	lf=$(license_file "$dir")
	{
		echo "================================================================================"
		echo "Go module: $path $version"
		echo "================================================================================"
		if [ -n "$lf" ]; then cat "$lf"; else echo "(licence file not found in module cache)"; fi
		echo
	} >>"$OUT"
done

# npm packages bundled into the web UI (production dependencies only).
if [ -d web/node_modules ]; then
	(cd web && npm ls --omit=dev --all --parseable 2>/dev/null | tail -n +2) | sort -u | while read -r dir; do
		name=$(node -p "require('$dir/package.json').name" 2>/dev/null || basename "$dir")
		version=$(node -p "require('$dir/package.json').version" 2>/dev/null || echo "")
		lf=$(license_file "$dir")
		{
			echo "================================================================================"
			echo "npm package: $name $version"
			echo "================================================================================"
			if [ -n "$lf" ]; then cat "$lf"; else node -p "'Licence: ' + require('$dir/package.json').license" 2>/dev/null || true; fi
			echo
		} >>"$OUT"
	done
fi

echo "Wrote $OUT"
