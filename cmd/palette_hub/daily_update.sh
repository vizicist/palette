#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Request daily logs from all palettes
"$SCRIPT_DIR/palettes_requestdays.sh" > daily_update.out

# Analyze and generate HTML report
"$SCRIPT_DIR/update_usage.sh" >> daily_update.out
