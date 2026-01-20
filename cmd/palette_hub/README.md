# Palette Hub Analysis Tools

## palette_hub - NATS Stream Dumper

Command-line tool to extract data from NATS JetStream.

### Commands

```bash
# List available streams
palette_hub streams

# Dump raw stream data
palette_hub dumpraw [streamname]

# Dump load events only
palette_hub dumpload [streamname]

# Dump data for a specific day
palette_hub dumpday {date} [streamname]
  # Date formats: 2025-12-11, 12-11, 12/11, today, yesterday

# Generate daily dump files (2025-01-01 to yesterday)
palette_hub dumpdays [streamname]
  # Creates days/*.json files, skips existing files

# Import events from a local engine.log file
palette_hub import_log {hostname}
  # Reads engine.log from stdin and merges into days/*.json files
  # Deduplicates against existing events
  # Filters out attract-mode loads (matching NATS behavior)
```

### Examples

```bash
# Dump all days to separate files
palette_hub dumpdays

# Dump just today's data to stdout
palette_hub dumpday today

# Dump a specific date
palette_hub dumpday 2025-12-11

# Import engine.log from a remote palette via SSH
# (run this command ON the palette machine)
cat engine.log | ssh tjt@timthompson.com "cd /home/tjt/github/palette/cmd/palette_hub && ./palette_hub import_log spacepalette34"
```

### Offline Palette Recovery

When a palette is disconnected from the network, it can't send NATS messages to log usage. However, the local `engine.log` file continues to record all events. The `import_log` command allows you to recover this data:

1. **Connect to the palette** (e.g., via Chrome Remote Desktop)
2. **Run the import command** from the palette, piping the engine.log to the hub machine:
   ```bash
   cat /path/to/engine.log | ssh user@hub-machine "cd /path/to/palette_hub && ./palette_hub import_log {hostname}"
   ```
3. The command will:
   - Parse the engine.log to extract `setAttractMode` and `Quad.Load` events
   - Reconstruct absolute timestamps from the log's relative uptime values
   - Skip loads that occurred during attract mode (matching NATS behavior)
   - Deduplicate against existing events in the days files
   - Merge new events chronologically into the appropriate day files

## analyze_days.py - Web-based Analysis

Python script that analyzes the daily JSON files and generates an interactive web page.

### Usage

```bash
# Analyze all files in days/ directory
python analyze_days.py

# Open the generated HTML file
# (opens palette_analysis.html in your browser)
```

### Requirements

- Python 3.6+
- No additional packages required (uses only standard library)
- Generated HTML uses Plotly.js from CDN (requires internet for charts)

### What It Shows

The web page displays:

1. **Summary Statistics** (updates based on date range)
   - Total loads across all palettes
   - Number of days analyzed
   - Number of unique palettes

2. **Date Range Filter**
   - Start date and end date pickers
   - Filter the chart and statistics to a specific date range
   - Defaults to showing the last month of data

3. **Interactive Stacked Bar Chart**
   - Displays daily loads as stacked bars
   - Each palette shown as a different color
   - Responds to date range selection
   - Shows day of week on x-axis labels

4. **Summary Table** (updates based on date range)
   - Total loads per palette
   - Days active
   - Average loads per day
   - Peak day and peak load count

### How It Works

The script:
1. Reads all `*.json` files from the `days/` directory (in chronological order)
2. Parses each line as JSON
3. Extracts palette names from subjects like `from_palette.palette7.load`
4. Tracks attract mode state changes:
   - When it sees an event with `attractmode: true` in the data, it marks that palette as in attract mode
   - When it sees an event with `attractmode: false` in the data, it marks that palette as not in attract mode
5. Counts `.load` events per palette per day, **excluding** loads that occur while attract mode is active
6. Generates a self-contained HTML file with embedded JavaScript

The palette name is extracted from the hostname portion of the subject:
- `from_palette.palette7.load` → palette: `palette7`
- `from_palette.photonsalon.load` → palette: `photonsalon`

**Note**: The script maintains per-palette attract mode state as it processes events sequentially. Loads that occur during attract mode are automatically excluded from the analysis, so the counts represent only real user activity.

## Environment Setup

Make sure your `.env` file contains:

```bash
NATS_HUB_CLIENT_URL=nats://username:password@hostname:4222
```

Use `palette env set NATS_HUB_CLIENT_URL "nats://..."` to configure.
