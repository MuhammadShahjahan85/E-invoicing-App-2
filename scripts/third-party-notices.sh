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
	echo "Veridian E-invoicing Pakistan — Third-party notices"
	echo
	echo "Veridian E-invoicing Pakistan is proprietary software of Veridian Partners Consultancy"
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

# Fonts embedded in the server for PDF documents.
for pair in "Inter:Inter" "PlusJakartaSans:Plus Jakarta Sans"; do
	file=${pair%%:*}
	name=${pair#*:}
	{
		echo "================================================================================"
		echo "Font embedded in PDF documents: $name (SIL Open Font License 1.1)"
		echo "================================================================================"
		cat "internal/pdf/fonts/$file-OFL.txt"
		echo
	} >>"$OUT"
done

# The Windows installer is built with NSIS.
cat >>"$OUT" <<'TXT'
================================================================================
Windows installer: NSIS (Nullsoft Scriptable Install System)
================================================================================
The setup program (VeridianEInvoicingPakistan-Setup-*.exe) and its uninstaller are
built with NSIS, https://nsis.sourceforge.io/
Copyright (C) 1999-2021 Nullsoft and Contributors. Licensed under the zlib/libpng
licence:

This software is provided 'as-is', without any express or implied warranty. In no
event will the authors be held liable for any damages arising from the use of this
software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to the
following restrictions:

1. The origin of this software must not be misrepresented; you must not claim that
   you wrote the original software. If you use this software in a product, an
   acknowledgment in the product documentation would be appreciated but is not
   required.
2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.
3. This notice may not be removed or altered from any source distribution.

The installer's LZMA decompression code is by Igor Pavlov (7-Zip), used under the
Common Public License with the special exception for NSIS installers.

TXT

echo "Wrote $OUT"
