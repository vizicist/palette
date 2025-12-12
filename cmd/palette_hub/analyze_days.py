#!/usr/bin/env python3
"""
Analyze daily palette dump files and generate an interactive web page.
Shows load counts per palette per day.
"""

import json
import os
import re
from collections import defaultdict
from datetime import datetime
import sys

# Hardcoded mapping of palette hostnames to readable names
PALETTE_NAME_MAP = {
    'palette12': 'Carleton University',
    'spacepalette34': 'Idea Fab Labs',
    'spacepalette35': 'MADE',
}

def extract_palette_name(subject):
    """
    Extract palette name from subject like 'from_palette.palette7.load'
    Returns mapped readable name or original hostname if not in map.
    Returns None if not a load message or no palette found.
    """
    if not subject.endswith('.load'):
        return None

    # Pattern to match palette names like 'palette7', 'palette.7', etc.
    # Subject format: from_palette.{hostname}.load
    # We want to extract the hostname part
    match = re.match(r'from_palette\.([^.]+)\.load', subject)
    if match:
        hostname = match.group(1)
        # Map to readable name if available, otherwise use hostname
        return PALETTE_NAME_MAP.get(hostname, hostname)

    return None

def analyze_day_file(filepath):
    """
    Analyze a single day file and return palette load counts and time-of-day data.
    Tracks attractmode state changes and ignores loads when attractmode is active.
    Returns tuple: (palette_loads dict, time_of_day_loads dict)
        - palette_loads: {palette_name: count}
        - time_of_day_loads: {palette_name: {hour: count}}
    """
    palette_loads = defaultdict(int)
    time_of_day_loads = defaultdict(lambda: defaultdict(int))
    # Track attract mode state per palette (default: False)
    attract_mode_state = defaultdict(bool)

    if not os.path.exists(filepath):
        return palette_loads, time_of_day_loads

    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue

                try:
                    entry = json.loads(line)
                    subject = entry.get('subject', '')
                    time_str = entry.get('time', '')

                    # Extract palette/host name from subject
                    # Subject format: from_palette.{hostname}.{event}
                    parts = subject.split('.')
                    if len(parts) >= 3 and parts[0] == 'from_palette':
                        palette_name = parts[1]
                        event_type = parts[2]

                        # Parse the data field
                        data_str = entry.get('data', '')
                        if data_str:
                            try:
                                data_obj = json.loads(data_str)
                            except (json.JSONDecodeError, AttributeError):
                                data_obj = {}
                        else:
                            data_obj = {}

                        # Check for attractmode state change events
                        if 'attractmode' in data_obj:
                            attract_mode_state[palette_name] = data_obj['attractmode']
                            # Don't count attractmode events themselves
                            continue

                        # Count .load events only if not in attract mode
                        if event_type == 'load':
                            if not attract_mode_state[palette_name]:
                                palette_loads[palette_name] += 1

                                # Extract hour from timestamp
                                if time_str:
                                    try:
                                        # Parse ISO format timestamp
                                        dt = datetime.fromisoformat(time_str.replace('Z', '+00:00'))
                                        hour = dt.hour
                                        time_of_day_loads[palette_name][hour] += 1
                                    except (ValueError, AttributeError):
                                        pass

                except json.JSONDecodeError as e:
                    print(f"Warning: Failed to parse JSON in {filepath}: {e}", file=sys.stderr)
                    continue

    except Exception as e:
        print(f"Error reading {filepath}: {e}", file=sys.stderr)

    return palette_loads, time_of_day_loads

def analyze_all_days(days_dir='days'):
    """
    Analyze all day files in the days directory.
    Returns tuple: (daily_data, per_day_time_of_day_data)
        - daily_data: {date_str: {palette_name: count}}
        - per_day_time_of_day_data: {date_str: {palette_name: {hour: count}}}
    """
    if not os.path.exists(days_dir):
        print(f"Error: Directory '{days_dir}' not found", file=sys.stderr)
        return {}, {}

    results = {}
    # Store per-day time-of-day data
    per_day_time_of_day = {}

    files = sorted([f for f in os.listdir(days_dir) if f.endswith('.json')])

    for filename in files:
        date_str = filename.replace('.json', '')
        filepath = os.path.join(days_dir, filename)

        print(f"Analyzing {filename}...", file=sys.stderr)
        palette_loads, time_of_day_loads = analyze_day_file(filepath)

        if palette_loads:
            results[date_str] = dict(palette_loads)

        # Store time-of-day data for this day
        if time_of_day_loads:
            per_day_time_of_day[date_str] = {}
            for palette, hours in time_of_day_loads.items():
                per_day_time_of_day[date_str][palette] = dict(hours)

    return results, per_day_time_of_day

def generate_html(data, time_of_day_data, output_file='palette_analysis.html'):
    """
    Generate an interactive HTML page with the analysis data.
    """
    # Get all unique palettes (using original hostnames from data)
    all_palettes_in_data = set()
    for day_data in data.values():
        all_palettes_in_data.update(day_data.keys())

    # Filter to only include palettes that are in the mapping
    all_palettes = sorted([p for p in all_palettes_in_data if p in PALETTE_NAME_MAP])

    # Sort dates
    dates = sorted(data.keys())

    # Map palette hostnames to readable names
    mapped_palettes = [PALETTE_NAME_MAP[p] for p in all_palettes]

    # Prepare data for JavaScript
    js_data = {
        'dates': dates,
        'palettes': mapped_palettes,
        'loads': {},
        'timeOfDayByDate': {}
    }

    # Build loads data: loads[mapped_name][date] = count
    for palette, mapped_name in zip(all_palettes, mapped_palettes):
        js_data['loads'][mapped_name] = {}
        for date in dates:
            count = data.get(date, {}).get(palette, 0)
            js_data['loads'][mapped_name][date] = count

    # Build per-day time-of-day data: timeOfDayByDate[date][mapped_name][hour] = count
    for date in dates:
        js_data['timeOfDayByDate'][date] = {}
        for palette, mapped_name in zip(all_palettes, mapped_palettes):
            js_data['timeOfDayByDate'][date][mapped_name] = {}
            if date in time_of_day_data and palette in time_of_day_data[date]:
                for hour in range(24):
                    js_data['timeOfDayByDate'][date][mapped_name][hour] = time_of_day_data[date][palette].get(hour, 0)
            else:
                for hour in range(24):
                    js_data['timeOfDayByDate'][date][mapped_name][hour] = 0

    html = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Palette Load Analysis</title>
    <script src="https://cdn.plot.ly/plotly-2.27.0.min.js"></script>
    <style>
        body {{
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .container {{
            max-width: 1400px;
            margin: 0 auto;
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        h1 {{
            color: #333;
            margin-bottom: 10px;
        }}
        .subtitle {{
            color: #666;
            margin-bottom: 30px;
        }}
        #chart {{
            width: 100%;
            height: 600px;
            margin-bottom: 30px;
        }}
        #summary {{
            margin-top: 30px;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }}
        th, td {{
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }}
        th {{
            background-color: #4CAF50;
            color: white;
            position: sticky;
            top: 0;
        }}
        tr:hover {{
            background-color: #f5f5f5;
        }}
        .stats {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }}
        .stat-card {{
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }}
        .stat-value {{
            font-size: 32px;
            font-weight: bold;
            margin-bottom: 5px;
        }}
        .stat-label {{
            font-size: 14px;
            opacity: 0.9;
        }}
        .controls {{
            margin-bottom: 20px;
        }}
        select {{
            padding: 8px 12px;
            font-size: 14px;
            border: 1px solid #ddd;
            border-radius: 4px;
            background-color: white;
            cursor: pointer;
        }}
        input[type="date"] {{
            padding: 8px 12px;
            font-size: 14px;
            border: 1px solid #ddd;
            border-radius: 4px;
            background-color: white;
            cursor: pointer;
        }}
        label {{
            margin-right: 10px;
            font-weight: 500;
        }}
        .date-range {{
            display: inline-block;
        }}
        .date-range label {{
            margin-left: 15px;
        }}
        .date-range label:first-child {{
            margin-left: 0;
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>Palette Load Analysis</h1>
        <p class="subtitle">Daily .load event counts per palette</p>

        <div class="stats">
            <div class="stat-card">
                <div class="stat-value" id="total-loads">-</div>
                <div class="stat-label">Total Loads</div>
            </div>
            <div class="stat-card" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);">
                <div class="stat-value" id="total-days">-</div>
                <div class="stat-label">Days Analyzed</div>
            </div>
            <div class="stat-card" style="background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);">
                <div class="stat-value" id="total-palettes">-</div>
                <div class="stat-label">Palettes</div>
            </div>
        </div>

        <div class="controls">
            <label for="view-type">View:</label>
            <select id="view-type" onchange="updateChart()">
                <option value="daily">Daily Loads</option>
                <option value="time-of-day">Time of Day</option>
            </select>

            <div class="date-range">
                <label for="start-date">From:</label>
                <input type="date" id="start-date" onchange="updateChart()">

                <label for="end-date">To:</label>
                <input type="date" id="end-date" onchange="updateChart()">
            </div>
        </div>

        <div id="chart"></div>

        <div id="summary">
            <h2>Summary Table</h2>
            <table id="summary-table">
                <thead>
                    <tr>
                        <th>Palette</th>
                        <th>Total Loads</th>
                        <th>Days Active</th>
                        <th>Avg Loads/Day</th>
                        <th>Peak Day</th>
                        <th>Peak Loads</th>
                    </tr>
                </thead>
                <tbody id="summary-tbody">
                </tbody>
            </table>
        </div>
    </div>

    <script>
        // Data from Python
        const data = {json.dumps(js_data, indent=8)};

        // Calculate statistics for filtered date range
        function calculateStats(filteredDates) {{
            const stats = {{}};
            let totalLoads = 0;

            for (const palette of data.palettes) {{
                const loads = data.loads[palette];

                // Only consider dates in the filtered range
                const filteredLoadValues = filteredDates.map(date => loads[date] || 0);
                const total = filteredLoadValues.reduce((a, b) => a + b, 0);
                const daysActive = filteredLoadValues.filter(v => v > 0).length;
                const avg = daysActive > 0 ? (total / daysActive).toFixed(1) : 0;

                let peakDay = '';
                let peakLoads = 0;
                for (const date of filteredDates) {{
                    const count = loads[date] || 0;
                    if (count > peakLoads) {{
                        peakLoads = count;
                        peakDay = date;
                    }}
                }}

                stats[palette] = {{
                    total,
                    daysActive,
                    avg,
                    peakDay,
                    peakLoads
                }};

                totalLoads += total;
            }}

            return {{ stats, totalLoads }};
        }}

        // Update summary statistics
        function updateSummary() {{
            const filteredDates = getFilteredDates();
            const {{ stats, totalLoads }} = calculateStats(filteredDates);

            document.getElementById('total-loads').textContent = totalLoads.toLocaleString();
            document.getElementById('total-days').textContent = filteredDates.length;
            document.getElementById('total-palettes').textContent = data.palettes.length;

            const tbody = document.getElementById('summary-tbody');
            tbody.innerHTML = '';

            // Sort palettes by total loads (descending)
            const sortedPalettes = data.palettes.slice().sort((a, b) =>
                stats[b].total - stats[a].total
            );

            for (const palette of sortedPalettes) {{
                const s = stats[palette];
                const row = tbody.insertRow();
                row.innerHTML = `
                    <td>${{palette}}</td>
                    <td>${{s.total.toLocaleString()}}</td>
                    <td>${{s.daysActive}}</td>
                    <td>${{s.avg}}</td>
                    <td>${{s.peakDay}}</td>
                    <td>${{s.peakLoads}}</td>
                `;
            }}
        }}

        // Get filtered date range
        function getFilteredDates() {{
            const startDate = document.getElementById('start-date').value;
            const endDate = document.getElementById('end-date').value;

            return data.dates.filter(date => {{
                if (startDate && date < startDate) return false;
                if (endDate && date > endDate) return false;
                return true;
            }});
        }}

        // Format date with day of week
        function formatDateWithDay(dateStr) {{
            const date = new Date(dateStr + 'T00:00:00');
            const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
            const dayName = days[date.getDay()];
            return `${{dayName}}<br>${{dateStr}}`;
        }}

        // Update chart based on selected view type
        function updateChart() {{
            const viewType = document.getElementById('view-type').value;

            if (viewType === 'daily') {{
                updateDailyChart();
            }} else if (viewType === 'time-of-day') {{
                updateTimeOfDayChart();
            }}

            updateSummary();
        }}

        // Update daily loads chart with stacked bars
        function updateDailyChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];

            // Format x-axis labels with day of week
            const xLabels = filteredDates.map(date => formatDateWithDay(date));

            for (const palette of data.palettes) {{
                const counts = filteredDates.map(date => data.loads[palette][date] || 0);

                const trace = {{
                    x: xLabels,
                    y: counts,
                    name: palette,
                    type: 'bar'
                }};

                traces.push(trace);
            }}

            const layout = {{
                title: 'Palette Loads Over Time',
                xaxis: {{
                    title: 'Date',
                    tickangle: -45
                }},
                yaxis: {{
                    title: 'Number of Loads'
                }},
                barmode: 'stack',
                hovermode: 'x unified',
                showlegend: true,
                legend: {{
                    orientation: 'v',
                    x: 1.02,
                    y: 1
                }}
            }};

            Plotly.newPlot('chart', traces, layout, {{responsive: true}});
        }}

        // Update time-of-day heatmap
        function updateTimeOfDayChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];

            // Aggregate time-of-day data across filtered dates
            const aggregatedTimeOfDay = {{}};
            for (const palette of data.palettes) {{
                aggregatedTimeOfDay[palette] = Array(24).fill(0);

                for (const date of filteredDates) {{
                    if (data.timeOfDayByDate[date] && data.timeOfDayByDate[date][palette]) {{
                        for (let hour = 0; hour < 24; hour++) {{
                            aggregatedTimeOfDay[palette][hour] += data.timeOfDayByDate[date][palette][hour] || 0;
                        }}
                    }}
                }}
            }}

            // Create a bar trace for each palette
            for (const palette of data.palettes) {{
                const trace = {{
                    x: Array.from({{length: 24}}, (_, i) => i),
                    y: aggregatedTimeOfDay[palette],
                    name: palette,
                    type: 'bar'
                }};

                traces.push(trace);
            }}

            const hourLabels = Array.from({{length: 24}}, (_, i) => {{
                const hour12 = i === 0 ? 12 : (i > 12 ? i - 12 : i);
                const ampm = i < 12 ? 'AM' : 'PM';
                return `${{hour12}}${{ampm}}`;
            }});

            const dateRangeText = filteredDates.length === data.dates.length
                ? 'All Days Combined'
                : `${{filteredDates[0]}} to ${{filteredDates[filteredDates.length - 1]}}`;

            const layout = {{
                title: `Loads by Time of Day (${{dateRangeText}})`,
                xaxis: {{
                    title: 'Hour of Day',
                    tickmode: 'array',
                    tickvals: Array.from({{length: 24}}, (_, i) => i),
                    ticktext: hourLabels,
                    tickangle: -45
                }},
                yaxis: {{
                    title: 'Number of Loads'
                }},
                barmode: 'stack',
                hovermode: 'x unified',
                showlegend: true,
                legend: {{
                    orientation: 'v',
                    x: 1.02,
                    y: 1
                }}
            }};

            Plotly.newPlot('chart', traces, layout, {{responsive: true}});
        }}

        // Initialize date pickers with data range
        function initializeDatePickers() {{
            const startInput = document.getElementById('start-date');
            const endInput = document.getElementById('end-date');

            if (data.dates.length > 0) {{
                const minDate = data.dates[0];
                const maxDate = data.dates[data.dates.length - 1];

                startInput.min = minDate;
                startInput.max = maxDate;
                endInput.min = minDate;
                endInput.max = maxDate;

                // Default to showing last month of data
                const endDateObj = new Date(maxDate + 'T00:00:00');
                const startDateObj = new Date(endDateObj);
                startDateObj.setMonth(startDateObj.getMonth() - 1);

                // Format as YYYY-MM-DD
                const defaultStart = startDateObj.toISOString().split('T')[0];

                // Make sure we don't go before the minimum available date
                startInput.value = defaultStart < minDate ? minDate : defaultStart;
                endInput.value = maxDate;
            }}
        }}

        // Initialize
        initializeDatePickers();
        updateChart();
    </script>
</body>
</html>
"""

    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(html)

    print(f"Generated {output_file}", file=sys.stderr)

def main():
    """Main function"""
    print("Palette Load Analysis", file=sys.stderr)
    print("=" * 50, file=sys.stderr)

    # Analyze all day files
    data, time_of_day_data = analyze_all_days('days')

    if not data:
        print("No data found to analyze.", file=sys.stderr)
        return 1

    print(f"\nAnalyzed {len(data)} days", file=sys.stderr)

    # Generate HTML report
    generate_html(data, time_of_day_data)

    print("\nDone! Open palette_analysis.html in your browser.", file=sys.stderr)
    return 0

if __name__ == '__main__':
    sys.exit(main())
