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
    Analyze a single day file and return palette load counts, time-of-day data, session durations, and session list.
    Tracks attractmode state changes and ignores loads when attractmode is active.
    Returns tuple: (palette_loads dict, time_of_day_loads dict, session_durations dict, sessions list)
        - palette_loads: {palette_name: count}
        - time_of_day_loads: {palette_name: {hour: count}}
        - session_durations: {palette_name: total_seconds}
        - sessions: list of {palette, start_time, duration_seconds}
    """
    palette_loads = defaultdict(int)
    time_of_day_loads = defaultdict(lambda: defaultdict(int))
    session_durations = defaultdict(float)
    sessions = []
    # Track attract mode state per palette (default: False)
    attract_mode_state = defaultdict(bool)
    # Track when attract mode last turned off (session started)
    session_start_time = defaultdict(lambda: None)
    # Track load times to estimate session duration when attract events are missing
    first_load_time = defaultdict(lambda: None)
    last_load_time = defaultdict(lambda: None)
    # Track load count per current session
    session_load_count = defaultdict(int)
    # Track which palettes have seen any attract mode events (required for session tracking)
    has_attract_events = set()

    if not os.path.exists(filepath):
        return palette_loads, time_of_day_loads, session_durations, sessions

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
                        # The field is 'onoff' for .attract events, 'attractmode' for other events
                        attract_value = None
                        if 'onoff' in data_obj and event_type == 'attract':
                            attract_value = data_obj['onoff']
                        elif 'attractmode' in data_obj:
                            attract_value = data_obj['attractmode']

                        if attract_value is not None:
                            new_attract_state = attract_value
                            attract_mode_state[palette_name] = new_attract_state
                            has_attract_events.add(palette_name)

                            # Track session duration
                            if time_str:
                                try:
                                    event_time = datetime.fromisoformat(time_str.replace('Z', '+00:00'))

                                    # If attract mode is turning OFF (onoff: false, session starting)
                                    if not new_attract_state:
                                        session_start_time[palette_name] = event_time
                                        session_load_count[palette_name] = 0  # Reset load count for new session

                                    # If attract mode is turning ON (onoff: true, session ending)
                                    elif new_attract_state:
                                        if session_start_time[palette_name] is not None:
                                            # Only record session if it has loads
                                            if session_load_count[palette_name] > 0:
                                                duration = (event_time - session_start_time[palette_name]).total_seconds()
                                                session_durations[palette_name] += duration
                                                # Record individual session
                                                sessions.append({
                                                    'palette': palette_name,
                                                    'start_time': session_start_time[palette_name].isoformat(),
                                                    'duration_seconds': duration
                                                })
                                            session_start_time[palette_name] = None
                                            session_load_count[palette_name] = 0
                                except (ValueError, AttributeError):
                                    pass

                            # Don't count attractmode events themselves
                            continue

                        # Count .load events only if not in attract mode
                        if event_type == 'load':
                            # Skip if filename is "_Current"
                            filename = data_obj.get('filename', '')
                            if filename == '_Current':
                                continue

                            # Some palettes include attractmode state in load events - use it if present
                            if 'attractmode' in data_obj:
                                attract_mode_state[palette_name] = data_obj['attractmode']

                            if not attract_mode_state[palette_name]:
                                palette_loads[palette_name] += 1
                                session_load_count[palette_name] += 1  # Count load for current session

                                # Extract hour from timestamp
                                if time_str:
                                    try:
                                        # Parse ISO format timestamp
                                        dt = datetime.fromisoformat(time_str.replace('Z', '+00:00'))
                                        hour = dt.hour
                                        time_of_day_loads[palette_name][hour] += 1

                                        # Track first and last load times for session estimation
                                        if first_load_time[palette_name] is None:
                                            first_load_time[palette_name] = dt
                                        last_load_time[palette_name] = dt
                                    except (ValueError, AttributeError):
                                        pass

                except json.JSONDecodeError as e:
                    print(f"Warning: Failed to parse JSON in {filepath}: {e}", file=sys.stderr)
                    continue

    except Exception as e:
        print(f"Error reading {filepath}: {e}", file=sys.stderr)

    # Estimate session duration for palettes that have attract events but session wasn't closed
    # Only do this for palettes that have attract mode events - otherwise we can't track sessions
    for palette_name in palette_loads.keys():
        # Skip palettes without attract events - we can't track sessions for them
        if palette_name not in has_attract_events:
            continue

        if palette_loads[palette_name] > 0 and first_load_time[palette_name] is not None:
            # If there's an active session (started but not ended), estimate the duration
            if session_start_time[palette_name] is not None and last_load_time[palette_name] is not None:
                # Session started with attract mode off, but never ended
                # Estimate it lasted until last load time + buffer (5 minutes)
                estimated_end = last_load_time[palette_name]
                duration = (estimated_end - session_start_time[palette_name]).total_seconds() + 300
                session_durations[palette_name] += duration
                sessions.append({
                    'palette': palette_name,
                    'start_time': session_start_time[palette_name].isoformat(),
                    'duration_seconds': duration
                })

    return palette_loads, time_of_day_loads, session_durations, sessions

def analyze_all_days(days_dir='days'):
    """
    Analyze all day files in the days directory.
    Returns tuple: (daily_data, per_day_time_of_day_data, per_day_session_durations, all_sessions)
        - daily_data: {date_str: {palette_name: count}}
        - per_day_time_of_day_data: {date_str: {palette_name: {hour: count}}}
        - per_day_session_durations: {date_str: {palette_name: total_seconds}}
        - all_sessions: list of all sessions with {palette, start_time, duration_seconds}
    """
    if not os.path.exists(days_dir):
        print(f"Error: Directory '{days_dir}' not found", file=sys.stderr)
        return {}, {}, {}, []

    results = {}
    # Store per-day time-of-day data
    per_day_time_of_day = {}
    # Store per-day session durations
    per_day_session_durations = {}
    # Store all individual sessions
    all_sessions = []

    files = sorted([f for f in os.listdir(days_dir) if f.endswith('.json')])

    for filename in files:
        filepath = os.path.join(days_dir, filename)

        print(f"Analyzing {filename}...", file=sys.stderr)
        palette_loads, time_of_day_loads, _session_durations, sessions = analyze_day_file(filepath)

        # Group data by actual event date (from timestamps) instead of file date
        # For loads and time-of-day, we already track by actual event date in analyze_day_file
        if palette_loads:
            # Note: palette_loads are already from the file, but we keep using filename date for backward compat
            # This is actually correct since loads are counted per file processing
            date_str = filename.replace('.json', '')
            results[date_str] = dict(palette_loads)

        # Store time-of-day data
        if time_of_day_loads:
            date_str = filename.replace('.json', '')
            per_day_time_of_day[date_str] = {}
            for palette, hours in time_of_day_loads.items():
                per_day_time_of_day[date_str][palette] = dict(hours)

        # For sessions, group by their actual start date (not file date)
        for session in sessions:
            # Extract date from session start_time
            session_date = session['start_time'].split('T')[0]

            if session_date not in per_day_session_durations:
                per_day_session_durations[session_date] = defaultdict(float)

            per_day_session_durations[session_date][session['palette']] += session['duration_seconds']

        # Collect all sessions
        all_sessions.extend(sessions)

    # Convert defaultdicts to regular dicts
    for date in per_day_session_durations:
        per_day_session_durations[date] = dict(per_day_session_durations[date])

    return results, per_day_time_of_day, per_day_session_durations, all_sessions

def generate_html(data, time_of_day_data, session_duration_data, all_sessions, output_file='palette_analysis.html'):
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

    # Map sessions to use readable palette names
    mapped_sessions = []
    for session in all_sessions:
        mapped_sessions.append({
            'palette': PALETTE_NAME_MAP.get(session['palette'], session['palette']),
            'start_time': session['start_time'],
            'duration_seconds': session['duration_seconds']
        })

    # Prepare data for JavaScript
    js_data = {
        'dates': dates,
        'palettes': mapped_palettes,
        'loads': {},
        'timeOfDayByDate': {},
        'sessionDurationByDate': {},
        'sessions': mapped_sessions
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

    # Build per-day session duration data: sessionDurationByDate[date][mapped_name] = seconds
    for date in dates:
        js_data['sessionDurationByDate'][date] = {}
        for palette, mapped_name in zip(all_palettes, mapped_palettes):
            if date in session_duration_data and palette in session_duration_data[date]:
                js_data['sessionDurationByDate'][date][mapped_name] = session_duration_data[date][palette]
            else:
                js_data['sessionDurationByDate'][date][mapped_name] = 0

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
                <option value="session-duration" selected>Session Duration</option>
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
                        <th>Total Session Hours</th>
                        <th>Avg Hours/Day</th>
                        <th>Peak Day</th>
                        <th>Peak Loads</th>
                    </tr>
                </thead>
                <tbody id="summary-tbody">
                </tbody>
            </table>
        </div>

        <div id="session-list" style="display: block; margin-top: 30px;">
            <h2>Session List</h2>
            <div id="session-list-content" style="font-family: monospace; font-size: 14px; max-height: 500px; overflow-y: auto; background-color: #f9f9f9; padding: 15px; border-radius: 4px;">
            </div>
        </div>
    </div>

    <script>
        // Data from Python
        const data = {json.dumps(js_data, indent=8)};

        // Track palette visibility state
        const paletteVisibility = {{}};
        data.palettes.forEach(palette => {{
            paletteVisibility[palette] = true; // All visible by default
        }});

        // Capture visibility state from current chart
        function captureVisibilityState() {{
            const chartDiv = document.getElementById('chart');
            if (chartDiv && chartDiv.data) {{
                chartDiv.data.forEach(trace => {{
                    if (trace.name && paletteVisibility.hasOwnProperty(trace.name)) {{
                        paletteVisibility[trace.name] = trace.visible !== 'legendonly';
                    }}
                }});
            }}
        }}

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

                // Calculate session duration stats
                let totalSessionSeconds = 0;
                for (const date of filteredDates) {{
                    totalSessionSeconds += data.sessionDurationByDate[date][palette] || 0;
                }}
                const totalSessionHours = (totalSessionSeconds / 3600).toFixed(1);
                const avgHoursPerDay = filteredDates.length > 0 ? (totalSessionSeconds / 3600 / filteredDates.length).toFixed(1) : 0;

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
                    totalSessionHours,
                    avgHoursPerDay,
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
                    <td>${{s.totalSessionHours}}</td>
                    <td>${{s.avgHoursPerDay}}</td>
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
            // Capture current visibility state before updating
            captureVisibilityState();

            const viewType = document.getElementById('view-type').value;
            const sessionList = document.getElementById('session-list');

            if (viewType === 'daily') {{
                updateDailyChart();
                sessionList.style.display = 'none';
            }} else if (viewType === 'time-of-day') {{
                updateTimeOfDayChart();
                sessionList.style.display = 'none';
            }} else if (viewType === 'session-duration') {{
                updateSessionDurationChart();
                updateSessionList();
                sessionList.style.display = 'block';
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
                    type: 'bar',
                    visible: paletteVisibility[palette] ? true : 'legendonly'
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

            Plotly.react('chart', traces, layout, {{responsive: true}});
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
                    type: 'bar',
                    visible: paletteVisibility[palette] ? true : 'legendonly'
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

            Plotly.react('chart', traces, layout, {{responsive: true}});
        }}

        // Update session duration chart
        function updateSessionDurationChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];

            // Format x-axis labels with day of week
            const xLabels = filteredDates.map(date => formatDateWithDay(date));

            // Create a bar trace for each palette showing duration in hours
            for (const palette of data.palettes) {{
                const durations = filteredDates.map(date => {{
                    const seconds = data.sessionDurationByDate[date][palette] || 0;
                    return seconds / 3600; // Convert seconds to hours
                }});

                const trace = {{
                    x: xLabels,
                    y: durations,
                    name: palette,
                    type: 'bar',
                    visible: paletteVisibility[palette] ? true : 'legendonly'
                }};

                traces.push(trace);
            }}

            const layout = {{
                title: 'Session Duration (Non-Attract Mode Time)',
                xaxis: {{
                    title: 'Date',
                    tickangle: -45
                }},
                yaxis: {{
                    title: 'Hours'
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

            Plotly.react('chart', traces, layout, {{responsive: true}});
        }}

        // Update session list based on filtered date range
        function updateSessionList() {{
            const filteredDates = getFilteredDates();
            const startDate = filteredDates[0];
            const endDate = filteredDates[filteredDates.length - 1];

            // Filter sessions within date range and for visible palettes only
            const filteredSessions = data.sessions.filter(session => {{
                const sessionDate = session.start_time.split('T')[0];
                const dateInRange = sessionDate >= startDate && sessionDate <= endDate;
                const paletteVisible = paletteVisibility[session.palette];
                return dateInRange && paletteVisible;
            }});

            // Sort sessions by start time
            filteredSessions.sort((a, b) => {{
                return new Date(a.start_time) - new Date(b.start_time);
            }});

            // Format and display sessions
            const content = document.getElementById('session-list-content');
            if (filteredSessions.length === 0) {{
                content.innerHTML = 'No sessions found in the selected date range for the selected palettes.';
                return;
            }}

            const lines = filteredSessions.map(session => {{
                const startTime = new Date(session.start_time).toLocaleString();
                const durationMinutes = (session.duration_seconds / 60).toFixed(1);
                return `${{session.palette.padEnd(20)}} - ${{startTime}} - ${{durationMinutes}} minutes`;
            }});

            content.innerHTML = lines.join('<br>');
        }}

        // Handle zoom/pan events on the chart to update date range selectors
        function setupChartEventHandlers() {{
            const chartDiv = document.getElementById('chart');

            chartDiv.on('plotly_relayout', function(eventData) {{
                // Check if this is a zoom event with x-axis range change
                if (eventData['xaxis.range[0]'] && eventData['xaxis.range[1]']) {{
                    // Get the filtered dates to map indices back to dates
                    const filteredDates = getFilteredDates();

                    // The range values are indices into the x-axis array
                    const startIdx = Math.max(0, Math.floor(eventData['xaxis.range[0]']));
                    const endIdx = Math.min(filteredDates.length - 1, Math.ceil(eventData['xaxis.range[1]']));

                    // Map indices to dates
                    const newStartDate = filteredDates[startIdx];
                    const newEndDate = filteredDates[endIdx];

                    // Update the date inputs
                    document.getElementById('start-date').value = newStartDate;
                    document.getElementById('end-date').value = newEndDate;

                    // Trigger chart update to reflect the new date range
                    updateChart();
                }}
            }});

            // Handle legend clicks (palette visibility changes)
            chartDiv.on('plotly_restyle', function(eventData) {{
                // Capture the new visibility state
                captureVisibilityState();

                // Update session list if in session duration view
                const viewType = document.getElementById('view-type').value;
                if (viewType === 'session-duration') {{
                    updateSessionList();
                }}
            }});
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
        setupChartEventHandlers();
    </script>
</body>
</html>
"""

    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(html)

    print(f"Generated {output_file}", file=sys.stderr)

def main():
    """Main function"""
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <days_directory> <output_html_file>", file=sys.stderr)
        print(f"  days_directory   - directory containing daily JSON dump files", file=sys.stderr)
        print(f"  output_html_file - path for the generated HTML report", file=sys.stderr)
        return 1

    days_dir = sys.argv[1]
    output_file = sys.argv[2]

    print("Palette Load Analysis", file=sys.stderr)
    print("=" * 50, file=sys.stderr)

    # Analyze all day files
    data, time_of_day_data, session_duration_data, all_sessions = analyze_all_days(days_dir)

    if not data:
        print("No data found to analyze.", file=sys.stderr)
        return 1

    print(f"\nAnalyzed {len(data)} days", file=sys.stderr)

    # Generate HTML report
    generate_html(data, time_of_day_data, session_duration_data, all_sessions, output_file)

    print(f"\nDone! Open {output_file} in your browser.", file=sys.stderr)
    return 0

if __name__ == '__main__':
    sys.exit(main())
