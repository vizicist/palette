#!/usr/bin/sh

daysdir=/home/tjt/github/palette/cmd/palette_hub/days
htmlout=/var/www/timthompson.com/html/spacepalette/usage/index.html

./palette_hub dumpdays > daily_update.out 2>&1
python3 ./analyze_days.py $daysdir $htmlout >> daily_update.out 2>&1
