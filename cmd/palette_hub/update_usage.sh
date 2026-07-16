#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
palettesdir="$SCRIPT_DIR/palettes"

# Generate locally first.  Writing/opening the SMB-mounted /nas/t path can hang
# in headless launchd jobs on macOS (TCC/Network Volumes), which blocks future
# hourly runs behind daily_update.lock.  Deploy to the NAS over SSH instead.
localout="$SCRIPT_DIR/spacepalette_usage_index.html.tmp"
webcacheout="/Users/tjt/webcache/timthompson.com/html/spacepalette/usage/index.html"
naspath="/volume1/nosuch/www/timthompson.com/html/spacepalette/usage/index.html"

python3 "$SCRIPT_DIR/palettes_analyze.py" "$palettesdir" "$localout"
chmod 644 "$localout"

mkdir -p "$(dirname "$webcacheout")"
cp "$localout" "$webcacheout"
chmod 644 "$webcacheout"

# Synology's scp/sftp path handling can fail here even when the file exists, so
# stream via ssh and atomically move the temp file into place on the NAS.
ssh -q nosuchNAS2 "mkdir -p \"$(dirname "$naspath")\" && cat > \"$naspath.tmp\" && chmod 644 \"$naspath.tmp\" && mv \"$naspath.tmp\" \"$naspath\"" < "$localout"

rm -f "$localout"
