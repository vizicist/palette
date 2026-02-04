#!/bin/bash

# Request logs from palettes
# Usage: palettes_requestlogs.sh [location]
#   If location is given, request logs only from that palette
#   If no location, request logs from all palettes in palettes.json
# Output goes to palettes/{location}/{logfile} for each log type

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTES_JSON="$SCRIPT_DIR/palettes.json"
PALETTES_DIR="$SCRIPT_DIR/palettes"

# Log files to request from each palette
LOGFILES="engine.log monitor.log"

if [ ! -f "$PALETTES_JSON" ]; then
    echo "Error: $PALETTES_JSON not found"
    exit 1
fi

# Function to request logs from a single palette
request_logs() {
    local hostname="$1"
    local location="$2"

    # Create output directory using location
    local outdir="$PALETTES_DIR/$location"
    mkdir -p "$outdir"

    echo "Requesting logs from $hostname ($location)..."

    # Request each log file
    for logfile in $LOGFILES; do
        local destfile="$outdir/$logfile"
        local tmpfile="$outdir/$logfile.tmp"

        palette_hub request_log "$hostname" "logfile=$logfile" > "$tmpfile" 2>&1

        if [ $? -eq 0 ] && [ -s "$tmpfile" ]; then
            local newsize=$(stat -c%s "$tmpfile" 2>/dev/null || echo 0)
            local oldsize=$(stat -c%s "$destfile" 2>/dev/null || echo 0)

            if [ "$newsize" -gt "$oldsize" ]; then
                mv "$tmpfile" "$destfile"
                local lines=$(wc -l < "$destfile")
                echo "  -> $logfile: $lines lines (updated)"
            else
                rm -f "$tmpfile"
                echo "  -> $logfile: no new data (keeping existing)"
            fi
        else
            rm -f "$tmpfile"
            echo "  -> $logfile: no response"
        fi
    done
}

# Check if a specific location was provided
if [ -n "$1" ]; then
    target_location="$1"
    found=false

    # Find the palette in palettes.json
    while IFS= read -r line || [ -n "$line" ]; do
        [ -z "$line" ] && continue

        hostname=$(echo "$line" | sed -n 's/.*"hostname":"\([^"]*\)".*/\1/p')
        location=$(echo "$line" | sed -n 's/.*"location":"\([^"]*\)".*/\1/p')

        if [ "$location" = "$target_location" ]; then
            request_logs "$hostname" "$location"
            found=true
            break
        fi
    done < "$PALETTES_JSON"

    if [ "$found" = false ]; then
        echo "Error: location '$target_location' not found in palettes.json"
        exit 1
    fi
else
    # Process all palettes from palettes.json
    while IFS= read -r line || [ -n "$line" ]; do
        [ -z "$line" ] && continue

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

        request_logs "$hostname" "$location"
    done < "$PALETTES_JSON"
fi

echo "Done."
