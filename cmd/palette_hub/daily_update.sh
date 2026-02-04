#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
palettesdir="$SCRIPT_DIR/palettes"

# Request daily logs from all palettes
"$SCRIPT_DIR/palettes_requestdays.sh" > daily_update.out

htmlout=/var/www/timthompson.com/html/spacepalette/usage/index.html
# Analyze and generate HTML report
python3 "$SCRIPT_DIR/palettes_analyze.py" "$palettesdir" "$htmlout" >> daily_update.out
