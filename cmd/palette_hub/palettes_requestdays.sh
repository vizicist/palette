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
START_DATE="2026-01-01"
END_DATE=$(date +%Y-%m-%d)

echo "Requesting logs from $START_DATE to $END_DATE"

# Read each line from palettes.json and request logs
while IFS= read -r line; do
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

    # Loop through each day
    current_date="$START_DATE"
    while [[ "$current_date" < "$END_DATE" ]] || [[ "$current_date" == "$END_DATE" ]]; do
        outfile="$outdir/${current_date}.json"

        # Skip if file already exists
        if [ -f "$outfile" ]; then
            current_date=$(date -d "$current_date + 1 day" +%Y-%m-%d)
            continue
        fi

        # Calculate start and end times for this day (UTC)
        day_start="${current_date}T00:00:00Z"
        day_end="${current_date}T23:59:59Z"

        echo "  Requesting $current_date...  $day_start to $day_end"
        "$SCRIPT_DIR/palette_hub" request_log "$hostname" "start=$day_start" "end=$day_end" > "$outfile" 2>/dev/null

        # Check if we got any data
        if [ -s "$outfile" ]; then
            lines=$(wc -l < "$outfile")
            echo "    -> $lines entries"
        else
            # Remove empty file
            rm -f "$outfile"
            echo "    -> no entries"
        fi

        # Next day
        current_date=$(date -d "$current_date + 1 day" +%Y-%m-%d)
    done

done < "$PALETTES_JSON"

echo "Done."
