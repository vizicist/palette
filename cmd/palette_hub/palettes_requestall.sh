#!/bin/bash

# Request full engine logs from all palettes listed in palettes.json
# Output goes to palettes/{location}/engine.log

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTES_JSON="$SCRIPT_DIR/palettes.json"
PALETTES_DIR="$SCRIPT_DIR/palettes"

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
    palette_hub request_log "$hostname" > "$outdir/engine.log" 2>&1

    if [ $? -eq 0 ]; then
        lines=$(wc -l < "$outdir/engine.log")
        echo "  -> Saved $lines lines to $outdir/engine.log"
    else
        echo "  -> Error requesting logs from $hostname"
    fi
done < "$PALETTES_JSON"

echo "Done."
