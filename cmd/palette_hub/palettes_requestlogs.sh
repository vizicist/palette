#!/bin/bash

# Request logs from all palettes listed in palettes.json
# Output goes to palettes/{location}/{logfile} for each log type

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTES_JSON="$SCRIPT_DIR/palettes.json"
PALETTES_DIR="$SCRIPT_DIR/palettes"

# Log files to request from each palette
LOGFILES="engine.log monitor.log ffgl.log"

if [ ! -f "$PALETTES_JSON" ]; then
    echo "Error: $PALETTES_JSON not found"
    exit 1
fi

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

    echo "Requesting logs from $hostname ($location)..."

    # Request each log file
    for logfile in $LOGFILES; do
        palette_hub request_log "$hostname" "logfile=$logfile" > "$outdir/$logfile" 2>&1

        if [ $? -eq 0 ] && [ -s "$outdir/$logfile" ]; then
            lines=$(wc -l < "$outdir/$logfile")
            echo "  -> $logfile: $lines lines"
        else
            # Remove empty file
            rm -f "$outdir/$logfile"
            echo "  -> $logfile: no data"
        fi
    done
done < "$PALETTES_JSON"

echo "Done."
