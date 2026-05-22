#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
palettesdir="$SCRIPT_DIR/palettes"

htmlout=/nas/t/www/timthompson.com/html/spacepalette/usage/index.html
# Analyze and generate HTML report
python3 "$SCRIPT_DIR/palettes_analyze.py" "$palettesdir" "$htmlout"
chmod 644 "$htmlout"
