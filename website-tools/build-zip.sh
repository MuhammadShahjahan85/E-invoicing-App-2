#!/bin/sh
# Builds the upload package for Hostinger: website-dist/veridian-partners-website.zip
# Its contents go straight into public_html (no extra folder level).
set -e
ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/website-dist"
mkdir -p "$OUT"
rm -f "$OUT/veridian-partners-website.zip"
cd "$ROOT/website"
# Never ship runtime data: only the protective files of /storage are included.
zip -r -q -X "$OUT/veridian-partners-website.zip" . \
  -x 'storage/*' -x '*.DS_Store' -x '*/Thumbs.db'
zip -q -X "$OUT/veridian-partners-website.zip" storage/.htaccess storage/index.html
echo "Built $OUT/veridian-partners-website.zip"
unzip -l "$OUT/veridian-partners-website.zip" | tail -1
