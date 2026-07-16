#!/bin/bash

# Request daily engine logs from all palettes listed in palettes.json
# Output goes to palettes/{location}/{YYYY-MM-DD}.json for each day

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTES_JSON="$SCRIPT_DIR/palettes.json"
PALETTES_DIR="$SCRIPT_DIR/palettes"

if [ ! -f "$PALETTES_JSON" ]; then
    echo "Error: $PALETTES_JSON not found"
    exit 1
fi

# Get date range
START_DATE="2025-01-01"
END_DATE=$(date +%Y-%m-%d)
# Always refresh the last 3 days (current day + 2 past days)
if command -v gdate >/dev/null 2>&1; then
    DATE_CMD=gdate
else
    DATE_CMD=date
fi

next_day() {
    if [ "$DATE_CMD" = "gdate" ]; then
        gdate -d "$1 + 1 day" +%Y-%m-%d
    else
        # BSD/macOS date requires -v adjustments before the input parse (-f).
        date -j -v+1d -f %Y-%m-%d "$1" +%Y-%m-%d
    fi
}

rfc3339_tz_offset() {
    if [ "$DATE_CMD" = "gdate" ]; then
        gdate +%:z
    else
        # BSD/macOS date has %z (-0700), not GNU %:z (-07:00).
        date +%z | sed 's/\(..\)$/\:\1/'
    fi
}

if [ "$DATE_CMD" = "gdate" ]; then
    REFRESH_CUTOFF=$(gdate -d "$END_DATE - 2 days" +%Y-%m-%d)
else
    REFRESH_CUTOFF=$(date -j -v-2d +%Y-%m-%d)
fi

# Optional quick mode for manual troubleshooting.  Normal cron runs should not
# set RECENT_ONLY, so missing days are backfilled when a palette comes back.
if [ "${RECENT_ONLY:-}" = "1" ]; then
    START_DATE="$REFRESH_CUTOFF"
fi

echo "Requesting logs from $START_DATE to $END_DATE (refreshing $REFRESH_CUTOFF and later)"

# Read each line from palettes.json and request logs
while IFS= read -r line || [ -n "$line" ]; do
    # Skip empty lines
    [ -z "$line" ] && continue

    # Extract the hostname and location fields
    hostname=$(echo "$line" | sed -n 's/.*"hostname":"\([^"]*\)".*/\1/p')
    location=$(echo "$line" | sed -n 's/.*"location":"\([^"]*\)".*/\1/p')

    if [ -z "$hostname" ]; then
        echo "Warning: Could not parse hostname from: $line"
        continue
    fi
    if [ -z "$location" ]; then
        echo "Warning: Could not parse location from: $line"
        continue
    fi

    # Create output directory using location
    outdir="$PALETTES_DIR/$location"
    mkdir -p "$outdir"

    echo "Processing $hostname ($location)..."

    # Loop through each day.  If a palette is offline and has a long historical
    # gap, do not spend an entire hourly launchd run requesting every old missing
    # day; after a few old "no responders" misses, skip forward to the normal
    # recent refresh window.  Recent days are still always requested.
    consecutive_old_no_responders=0
    current_date="$START_DATE"
    while [[ "$current_date" < "$END_DATE" ]] || [[ "$current_date" == "$END_DATE" ]]; do
        outfile="$outdir/${current_date}.json"

        # For recent days (last 3), always re-request to capture new events.
        # For older days, skip if file already exists.
        if [ -f "$outfile" ] && [[ "$current_date" < "$REFRESH_CUTOFF" ]]; then
            current_date=$(next_day "$current_date")
            continue
        fi

        # Calculate start and end times for this day (local timezone)
        tz_offset=$(rfc3339_tz_offset)
        day_start="${current_date}T00:00:00${tz_offset}"
        day_end="${current_date}T23:59:59${tz_offset}"

        echo "  Requesting $current_date...  $day_start to $day_end"
        tmpfile="$outfile.tmp.$$"
        if ! palette_hub request_log "$hostname" "start=$day_start" "end=$day_end" > "$tmpfile" 2>&1; then
            echo "    -> ERROR: palette_hub command failed"
            sed 's/^/       /' "$tmpfile" | head -5
            if [[ "$current_date" < "$REFRESH_CUTOFF" ]] && grep -q 'no responders available' "$tmpfile"; then
                consecutive_old_no_responders=$((consecutive_old_no_responders + 1))
            else
                consecutive_old_no_responders=0
            fi
            rm -f "$tmpfile"
            if [ -f "$outfile" ] && head -1 "$outfile" | grep -q '^Error:'; then
                rm -f "$outfile"
            fi
        elif grep -q '^Error:' "$tmpfile"; then
            echo "    -> ERROR: palette_hub returned an error"
            sed 's/^/       /' "$tmpfile" | head -5
            if [[ "$current_date" < "$REFRESH_CUTOFF" ]] && grep -q 'no responders available' "$tmpfile"; then
                consecutive_old_no_responders=$((consecutive_old_no_responders + 1))
            else
                consecutive_old_no_responders=0
            fi
            rm -f "$tmpfile"
            if [ -f "$outfile" ] && head -1 "$outfile" | grep -q '^Error:'; then
                rm -f "$outfile"
            fi
        else
            mv "$tmpfile" "$outfile"
            consecutive_old_no_responders=0
            # Check if we got any data
            if [ -s "$outfile" ]; then
                lines=$(wc -l < "$outfile")
                echo "    -> $lines entries"
            else
                echo "    -> no entries"
            fi
        fi

        if [[ "$current_date" < "$REFRESH_CUTOFF" ]] && [ "$consecutive_old_no_responders" -ge 3 ]; then
            echo "    -> skipping old missing days for $hostname until $REFRESH_CUTOFF after $consecutive_old_no_responders no-responder replies"
            current_date="$REFRESH_CUTOFF"
            consecutive_old_no_responders=0
            continue
        fi

        # Next day
        current_date=$(next_day "$current_date")
    done

done < "$PALETTES_JSON"

echo "Done."
