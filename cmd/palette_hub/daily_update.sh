#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOCKDIR="$SCRIPT_DIR/daily_update.lock"

if ! mkdir "$LOCKDIR" 2>/dev/null; then
    echo "daily_update already running; skipping this run" > "$SCRIPT_DIR/daily_update.out"
    exit 0
fi
trap 'rmdir "$LOCKDIR"' EXIT

# Request daily logs from all palettes.  Do a full missing-day backfill so if a
# palette is down for a week, its recovered logs still make it into the report.
# Capture stderr too, otherwise cron/launchd failures can hide in local mail.
"$SCRIPT_DIR/palettes_requestdays.sh" > "$SCRIPT_DIR/daily_update.out" 2>&1

# Analyze and generate HTML report
"$SCRIPT_DIR/update_usage.sh" >> "$SCRIPT_DIR/daily_update.out" 2>&1
