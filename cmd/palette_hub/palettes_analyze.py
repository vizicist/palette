#!/usr/bin/env python3
"""
Analyze palette log files from palettes/{location}/ directories and generate an interactive web page.
Shows load counts per palette per day.
"""

import json
import os
from collections import defaultdict
from datetime import datetime, timezone
import sys

# Default color palette for auto-assignment
DEFAULT_COLORS = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
]

# Read location list and colors from palettes.json
def load_palette_config(script_dir):
    """Load palette locations and colors from palettes.json"""
    config_path = os.path.join(script_dir, 'palettes.json')
    locations = []
    colors = {}
    used_colors = set()

    if os.path.exists(config_path):
        with open(config_path, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    entry = json.loads(line)
                    location = entry.get('location', '')
                    color = entry.get('color', '')
                    if location:
                        locations.append(location)
                        if color:
                            colors[location] = color
                            used_colors.add(color.lower())
                except json.JSONDecodeError:
                    continue

    # Assign colors to palettes that don't have one
    for location in locations:
        if location not in colors:
            # Find an unused color from the default palette
            assigned_color = None
            for default_color in DEFAULT_COLORS:
                if default_color.lower() not in used_colors:
                    assigned_color = default_color
                    break

            # If all default colors are used, generate a new one
            if assigned_color is None:
                # Generate a color based on hash of location name
                hash_val = hash(location) & 0xFFFFFF
                assigned_color = f'#{hash_val:06x}'

            colors[location] = assigned_color
            used_colors.add(assigned_color.lower())
            print(f"Warning: No color assigned for '{location}' in palettes.json. Using: {assigned_color}", file=sys.stderr)

    return locations, colors

def analyze_day_file(filepath, location):
    """
    Analyze a single day file from a palette location directory.
    The file contains engine log entries with "msg" field indicating event type.
    Returns tuple: (palette_loads dict, time_of_day_loads dict, session_durations dict, sessions list, restart_count int)
    """
    palette_loads = defaultdict(int)
    time_of_day_loads = defaultdict(lambda: defaultdict(int))
    session_durations = defaultdict(float)
    sessions = []
    restart_count = 0

    # Track attract mode state (default: True, palette starts in attract mode)
    attract_mode_on = True
    # Track when attract mode last turned off (session started)
    session_start_time = None
    # Track load count per current session
    session_load_count = 0
    # Track which palettes have seen any attract mode events
    has_attract_events = False
    # Track load times for session estimation
    first_load_time = None
    last_load_time = None
    # Track uptime to detect reboots
    prev_uptime = None
    last_event_time = None

    if not os.path.exists(filepath):
        return palette_loads, time_of_day_loads, session_durations, sessions, restart_count

    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue

                try:
                    entry = json.loads(line)
                    msg = entry.get('msg', '')
                    time_str = entry.get('time', '')
                    uptime_str = entry.get('uptime', '')

                    # Detect reboot via uptime decrease (uptime resets on reboot)
                    if uptime_str:
                        try:
                            current_uptime = float(uptime_str)
                            if prev_uptime is not None and current_uptime < prev_uptime - 60:
                                # Uptime decreased significantly - this is a reboot
                                restart_count += 1
                                # Close any open session at the last known time
                                if session_start_time is not None and session_load_count > 0 and last_event_time is not None:
                                    duration = (last_event_time - session_start_time).total_seconds()
                                    if duration > 0:
                                        session_durations[location] += duration
                                        sessions.append({
                                            'palette': location,
                                            'start_time': session_start_time.isoformat(),
                                            'duration_seconds': duration
                                        })
                                # Reset session state after reboot
                                session_start_time = None
                                session_load_count = 0
                                attract_mode_on = True  # Assume attract mode on after reboot
                            prev_uptime = current_uptime
                        except (ValueError, TypeError):
                            pass

                    # Update last event time for reboot detection
                    if time_str:
                        try:
                            last_event_time = datetime.fromisoformat(time_str.replace('Z', '+00:00')).astimezone(timezone.utc)
                        except (ValueError, AttributeError):
                            pass

                    # Check for attract mode state change
                    if msg == 'setAttractMode':
                        onoff = entry.get('onoff', False)
                        has_attract_events = True

                        if time_str:
                            try:
                                event_time = datetime.fromisoformat(time_str.replace('Z', '+00:00')).astimezone(timezone.utc)

                                # If attract mode is turning OFF (session starting)
                                if not onoff and attract_mode_on:
                                    session_start_time = event_time
                                    session_load_count = 0

                                # If attract mode is turning ON (session ending)
                                elif onoff and not attract_mode_on:
                                    if session_start_time is not None and session_load_count > 0:
                                        duration = (event_time - session_start_time).total_seconds()
                                        session_durations[location] += duration
                                        sessions.append({
                                            'palette': location,
                                            'start_time': session_start_time.isoformat(),
                                            'duration_seconds': duration
                                        })
                                    session_start_time = None
                                    session_load_count = 0
                            except (ValueError, AttributeError):
                                pass

                        attract_mode_on = onoff
                        continue

                    # Count Quad.Load events only if not in attract mode
                    if msg == 'Quad.Load':
                        filename = entry.get('filename', '')
                        if filename == '_Current':
                            continue

                        if not attract_mode_on:
                            palette_loads[location] += 1
                            session_load_count += 1

                            # Extract hour from timestamp (in local time for display)
                            if time_str:
                                try:
                                    dt = datetime.fromisoformat(time_str.replace('Z', '+00:00'))
                                    dt_local = dt.astimezone()  # Convert to local timezone
                                    hour = dt_local.hour
                                    time_of_day_loads[location][hour] += 1

                                    if first_load_time is None:
                                        first_load_time = dt.astimezone(timezone.utc)
                                    last_load_time = dt.astimezone(timezone.utc)
                                except (ValueError, AttributeError):
                                    pass

                except json.JSONDecodeError as e:
                    continue

    except Exception as e:
        print(f"Error reading {filepath}: {e}")

    # Estimate session duration if session wasn't closed
    if has_attract_events and palette_loads[location] > 0 and first_load_time is not None:
        if session_start_time is not None and last_load_time is not None:
            # Only estimate if last_load_time is after session_start_time
            # (otherwise last_load_time is from a previous session)
            if last_load_time > session_start_time:
                estimated_end = last_load_time
                duration = (estimated_end - session_start_time).total_seconds() + 300
                session_durations[location] += duration
                sessions.append({
                    'palette': location,
                    'start_time': session_start_time.isoformat(),
                    'duration_seconds': duration
                })

    return palette_loads, time_of_day_loads, session_durations, sessions, restart_count

def analyze_all_palettes(palettes_dir='palettes', script_dir=None):
    """
    Analyze all palette directories under palettes_dir.
    Each subdirectory is a location with date-named JSON files.
    Returns tuple: (daily_data, per_day_time_of_day_data, per_day_session_durations, all_sessions, palette_colors, per_day_restarts)
    """
    if not os.path.exists(palettes_dir):
        print(f"Error: Directory '{palettes_dir}' not found")
        return {}, {}, {}, [], {}, {}

    results = {}
    per_day_time_of_day = {}
    per_day_session_durations = {}
    per_day_restarts = {}
    all_sessions = []

    # Get locations and colors from palettes.json
    if script_dir is None:
        script_dir = os.path.dirname(os.path.abspath(__file__))
    locations, palette_colors = load_palette_config(script_dir)
    if not locations:
        print("Error: No locations found in palettes.json")
        return {}, {}, {}, [], {}, {}

    for location in locations:
        location_dir = os.path.join(palettes_dir, location)
        files = sorted([f for f in os.listdir(location_dir) if f.endswith('.json')])

        print(f"Analyzing {location} ({len(files)} files)...")

        for filename in files:
            filepath = os.path.join(location_dir, filename)
            date_str = filename.replace('.json', '')

            palette_loads, time_of_day_loads, _session_durations, sessions, restart_count = analyze_day_file(filepath, location)

            # Aggregate restart counts by date
            if restart_count > 0:
                if date_str not in per_day_restarts:
                    per_day_restarts[date_str] = {}
                per_day_restarts[date_str][location] = per_day_restarts[date_str].get(location, 0) + restart_count

            # Aggregate results by date
            if palette_loads:
                if date_str not in results:
                    results[date_str] = {}
                for palette, count in palette_loads.items():
                    results[date_str][palette] = results[date_str].get(palette, 0) + count

            # Store time-of-day data
            if time_of_day_loads:
                if date_str not in per_day_time_of_day:
                    per_day_time_of_day[date_str] = {}
                for palette, hours in time_of_day_loads.items():
                    if palette not in per_day_time_of_day[date_str]:
                        per_day_time_of_day[date_str][palette] = {}
                    for hour, count in hours.items():
                        per_day_time_of_day[date_str][palette][hour] = \
                            per_day_time_of_day[date_str][palette].get(hour, 0) + count

            # Process sessions
            for session in sessions:
                session_date = session['start_time'].split('T')[0]
                if session_date not in per_day_session_durations:
                    per_day_session_durations[session_date] = defaultdict(float)
                per_day_session_durations[session_date][session['palette']] += session['duration_seconds']

            all_sessions.extend(sessions)

    # Convert defaultdicts to regular dicts
    for date in per_day_session_durations:
        per_day_session_durations[date] = dict(per_day_session_durations[date])

    return results, per_day_time_of_day, per_day_session_durations, all_sessions, palette_colors, per_day_restarts

def generate_html(data, time_of_day_data, session_duration_data, all_sessions, palette_colors, restart_data, output_file='palette_analysis.html'):
    """
    Generate an interactive HTML page with the analysis data.
    """
    # Get all unique palettes from data
    all_palettes = set()
    for day_data in data.values():
        all_palettes.update(day_data.keys())
    all_palettes = sorted(all_palettes)

    # Sort dates
    dates = sorted(data.keys())

    # Prepare data for JavaScript
    js_data = {
        'dates': dates,
        'palettes': all_palettes,
        'colors': palette_colors,
        'loads': {},
        'timeOfDayByDate': {},
        'sessionDurationByDate': {},
        'restartsByDate': {},
        'sessions': all_sessions
    }

    # Build loads data
    for palette in all_palettes:
        js_data['loads'][palette] = {}
        for date in dates:
            count = data.get(date, {}).get(palette, 0)
            js_data['loads'][palette][date] = count

    # Build per-day time-of-day data
    for date in dates:
        js_data['timeOfDayByDate'][date] = {}
        for palette in all_palettes:
            js_data['timeOfDayByDate'][date][palette] = {}
            if date in time_of_day_data and palette in time_of_day_data[date]:
                for hour in range(24):
                    js_data['timeOfDayByDate'][date][palette][hour] = time_of_day_data[date][palette].get(hour, 0)
            else:
                for hour in range(24):
                    js_data['timeOfDayByDate'][date][palette][hour] = 0

    # Build per-day session duration data
    for date in dates:
        js_data['sessionDurationByDate'][date] = {}
        for palette in all_palettes:
            if date in session_duration_data and palette in session_duration_data[date]:
                js_data['sessionDurationByDate'][date][palette] = session_duration_data[date][palette]
            else:
                js_data['sessionDurationByDate'][date][palette] = 0

    # Build per-day restart data
    for date in dates:
        js_data['restartsByDate'][date] = {}
        for palette in all_palettes:
            if date in restart_data and palette in restart_data[date]:
                js_data['restartsByDate'][date][palette] = restart_data[date][palette]
            else:
                js_data['restartsByDate'][date][palette] = 0

    html = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Space Palette Usage</title>
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
        .controls {{
            margin-bottom: 20px;
        }}
        select, input[type="date"] {{
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
        .chart-container {{
            display: flex;
            gap: 20px;
            align-items: flex-start;
        }}
        .chart-container #chart {{
            flex: 1;
            min-width: 0;
        }}
        .palette-selector {{
            padding: 15px;
            background-color: #f9f9f9;
            border-radius: 8px;
            min-width: 180px;
        }}
        /* Responsive: stack chart and legend vertically on narrow screens */
        @media (max-width: 768px) {{
            .chart-container {{
                flex-direction: column;
            }}
            .palette-selector {{
                width: 100%;
                min-width: unset;
                order: 1;
            }}
            .chart-container #chart {{
                order: 0;
            }}
            #chart {{
                height: 400px;
            }}
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>Space Palette Usage</h1>

        <div class="chart-container">
            <div id="chart"></div>
            <div class="palette-selector">
                <div style="margin-bottom: 10px; font-weight: 500;">Palettes:</div>
                <div id="palette-checkboxes" style="display: flex; flex-direction: column; gap: 8px;">
                </div>

                <div class="controls" style="margin-top: 20px; padding-top: 15px; border-top: 1px solid #ddd; display: flex; flex-direction: column; gap: 10px;">
                    <div>
                        <label for="view-type" style="display: block; margin-bottom: 4px;">View:</label>
                        <select id="view-type" onchange="onSettingsChange()" style="width: 100%;">
                            <option value="daily">Daily Loads</option>
                            <option value="time-of-day">Time of Day</option>
                            <option value="session-duration" selected>Session Duration</option>
                            <option value="restarts">Restarts</option>
                        </select>
                    </div>

                    <div>
                        <label for="start-date" style="display: block; margin-bottom: 4px;">From:</label>
                        <input type="date" id="start-date" onchange="onSettingsChange()" style="width: 100%;">
                    </div>

                    <div>
                        <label for="end-date" style="display: block; margin-bottom: 4px;">To:</label>
                        <input type="date" id="end-date" onchange="onSettingsChange()" style="width: 100%;">
                    </div>

                    <button id="reset-defaults" onclick="resetToDefaults()" style="padding: 8px 12px; font-size: 14px; border: 1px solid #ddd; border-radius: 4px; background-color: #f0f0f0; cursor: pointer;">Reset to Defaults</button>
                </div>
            </div>
        </div>

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
        const data = {json.dumps(js_data, indent=8)};

        // Fallback colors for palettes without assigned colors
        const fallbackColors = [
            '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
            '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
        ];

        // Get color for a palette (from config or fallback)
        function getPaletteColor(palette, index) {{
            return data.colors[palette] || fallbackColors[index % fallbackColors.length];
        }}

        const paletteVisibility = {{}};
        data.palettes.forEach(palette => {{
            paletteVisibility[palette] = true;
        }});

        function calculateStats(filteredDates) {{
            const stats = {{}};
            for (const palette of data.palettes) {{
                const loads = data.loads[palette];
                const filteredLoadValues = filteredDates.map(date => loads[date] || 0);
                const total = filteredLoadValues.reduce((a, b) => a + b, 0);
                const daysActive = filteredLoadValues.filter(v => v > 0).length;
                const avg = daysActive > 0 ? (total / daysActive).toFixed(1) : 0;

                let totalSessionSeconds = 0;
                for (const date of filteredDates) {{
                    totalSessionSeconds += data.sessionDurationByDate[date][palette] || 0;
                }}
                const totalSessionHours = (totalSessionSeconds / 3600).toFixed(1);
                const avgHoursPerDay = filteredDates.length > 0 ? (totalSessionSeconds / 3600 / filteredDates.length).toFixed(1) : 0;

                stats[palette] = {{ total, daysActive, avg, totalSessionHours, avgHoursPerDay }};
            }}
            return stats;
        }}

        function updateSummary() {{
            const filteredDates = getFilteredDates();
            const stats = calculateStats(filteredDates);
            const tbody = document.getElementById('summary-tbody');
            tbody.innerHTML = '';

            const sortedPalettes = data.palettes.slice().sort((a, b) => stats[b].total - stats[a].total);

            for (const palette of sortedPalettes) {{
                const s = stats[palette];
                const row = tbody.insertRow();
                row.innerHTML = `<td>${{palette}}</td><td>${{s.total.toLocaleString()}}</td><td>${{s.daysActive}}</td><td>${{s.avg}}</td><td>${{s.totalSessionHours}}</td><td>${{s.avgHoursPerDay}}</td>`;
            }}
        }}

        function getFilteredDates() {{
            const startDate = document.getElementById('start-date').value;
            const endDate = document.getElementById('end-date').value;
            return data.dates.filter(date => {{
                if (startDate && date < startDate) return false;
                if (endDate && date > endDate) return false;
                return true;
            }});
        }}

        function formatDateWithDay(dateStr) {{
            const date = new Date(dateStr + 'T00:00:00');
            const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
            return `${{days[date.getDay()]}} ${{date.getMonth() + 1}}/${{date.getDate()}}`;
        }}

        function updateChart() {{
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
            }} else if (viewType === 'restarts') {{
                updateRestartsChart();
                sessionList.style.display = 'none';
            }}
            updateSummary();
        }}

        function updateDailyChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];
            const xLabels = filteredDates.map(date => formatDateWithDay(date));

            data.palettes.forEach((palette, index) => {{
                if (!paletteVisibility[palette]) return;
                const counts = filteredDates.map(date => data.loads[palette][date] || 0);
                traces.push({{
                    x: xLabels, y: counts, name: palette, type: 'bar',
                    marker: {{ color: getPaletteColor(palette, index) }}
                }});
            }});

            const layout = {{
                title: 'Palette Loads Over Time',
                xaxis: {{ tickangle: -45, fixedrange: true }},
                yaxis: {{ title: 'Number of Loads', fixedrange: true }},
                barmode: 'stack', hovermode: 'x unified', showlegend: false, dragmode: false
            }};

            Plotly.react('chart', traces, layout, {{responsive: true, displayModeBar: false}});
        }}

        function updateTimeOfDayChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];
            const aggregated = {{}};

            for (const palette of data.palettes) {{
                aggregated[palette] = Array(24).fill(0);
                for (const date of filteredDates) {{
                    if (data.timeOfDayByDate[date] && data.timeOfDayByDate[date][palette]) {{
                        for (let hour = 0; hour < 24; hour++) {{
                            aggregated[palette][hour] += data.timeOfDayByDate[date][palette][hour] || 0;
                        }}
                    }}
                }}
            }}

            data.palettes.forEach((palette, index) => {{
                if (!paletteVisibility[palette]) return;
                traces.push({{
                    x: Array.from({{length: 24}}, (_, i) => i),
                    y: aggregated[palette], name: palette, type: 'bar',
                    marker: {{ color: getPaletteColor(palette, index) }}
                }});
            }});

            const hourLabels = Array.from({{length: 24}}, (_, i) => {{
                const hour12 = i === 0 ? 12 : (i > 12 ? i - 12 : i);
                return `${{hour12}}${{i < 12 ? 'AM' : 'PM'}}`;
            }});

            const layout = {{
                title: 'Loads by Time of Day (Local Time)',
                xaxis: {{ title: 'Hour of Day', tickmode: 'array', tickvals: Array.from({{length: 24}}, (_, i) => i), ticktext: hourLabels, tickangle: -45, fixedrange: true }},
                yaxis: {{ title: 'Number of Loads', fixedrange: true }},
                barmode: 'stack', hovermode: 'x unified', showlegend: false, dragmode: false
            }};

            Plotly.react('chart', traces, layout, {{responsive: true, displayModeBar: false}});
        }}

        function updateSessionDurationChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];
            const xLabels = filteredDates.map(date => formatDateWithDay(date));

            data.palettes.forEach((palette, index) => {{
                if (!paletteVisibility[palette]) return;
                const durations = filteredDates.map(date => (data.sessionDurationByDate[date][palette] || 0) / 3600);
                traces.push({{
                    x: xLabels, y: durations, name: palette, type: 'bar',
                    marker: {{ color: getPaletteColor(palette, index) }}
                }});
            }});

            const layout = {{
                title: 'Session Duration (Non-Attract Mode Time)',
                xaxis: {{ tickangle: -45, fixedrange: true }},
                yaxis: {{ title: 'Hours', fixedrange: true }},
                barmode: 'stack', hovermode: 'x unified', showlegend: false, dragmode: false
            }};

            Plotly.react('chart', traces, layout, {{responsive: true, displayModeBar: false}});
        }}

        function updateRestartsChart() {{
            const filteredDates = getFilteredDates();
            const traces = [];
            const xLabels = filteredDates.map(date => formatDateWithDay(date));

            data.palettes.forEach((palette, index) => {{
                if (!paletteVisibility[palette]) return;
                const restarts = filteredDates.map(date => data.restartsByDate[date][palette] || 0);
                traces.push({{
                    x: xLabels, y: restarts, name: palette, type: 'bar',
                    marker: {{ color: getPaletteColor(palette, index) }}
                }});
            }});

            const layout = {{
                title: 'Restarts per Day',
                xaxis: {{ tickangle: -45, fixedrange: true }},
                yaxis: {{ title: 'Number of Restarts', fixedrange: true }},
                barmode: 'stack', hovermode: 'x unified', showlegend: false, dragmode: false
            }};

            Plotly.react('chart', traces, layout, {{responsive: true, displayModeBar: false}});
        }}

        function updateSessionList() {{
            const filteredDates = getFilteredDates();
            const startDate = filteredDates[0];
            const endDate = filteredDates[filteredDates.length - 1];

            const filteredSessions = data.sessions.filter(session => {{
                const sessionDate = session.start_time.split('T')[0];
                return sessionDate >= startDate && sessionDate <= endDate && paletteVisibility[session.palette];
            }});

            filteredSessions.sort((a, b) => new Date(a.start_time) - new Date(b.start_time));

            const content = document.getElementById('session-list-content');
            if (filteredSessions.length === 0) {{
                content.innerHTML = 'No sessions found in the selected date range.';
                return;
            }}

            const lines = filteredSessions.map(session => {{
                const startTime = new Date(session.start_time).toLocaleString();
                const durationMinutes = (session.duration_seconds / 60).toFixed(1);
                return `${{session.palette.padEnd(20)}} - ${{startTime}} - ${{durationMinutes}} minutes`;
            }});

            content.innerHTML = lines.join('<br>');
        }}

        function getDefaultDateRange() {{
            if (data.dates.length === 0) return {{ start: '', end: '' }};
            const maxDate = data.dates[data.dates.length - 1];
            const endDateObj = new Date(maxDate + 'T00:00:00');
            const startDateObj = new Date(endDateObj);
            startDateObj.setMonth(startDateObj.getMonth() - 1);
            const defaultStart = startDateObj.toISOString().split('T')[0];
            return {{ start: defaultStart < data.dates[0] ? data.dates[0] : defaultStart, end: maxDate }};
        }}

        function initializeDatePickers() {{
            const startInput = document.getElementById('start-date');
            const endInput = document.getElementById('end-date');
            if (data.dates.length > 0) {{
                startInput.min = data.dates[0];
                startInput.max = data.dates[data.dates.length - 1];
                endInput.min = data.dates[0];
                endInput.max = data.dates[data.dates.length - 1];
                const defaults = getDefaultDateRange();
                startInput.value = defaults.start;
                endInput.value = defaults.end;
            }}
        }}

        function onSettingsChange() {{
            updateChart();
        }}

        function resetToDefaults() {{
            window.location.href = window.location.pathname;
        }}

        function generatePaletteCheckboxes() {{
            const container = document.getElementById('palette-checkboxes');
            container.innerHTML = '';

            data.palettes.forEach((palette, index) => {{
                const color = getPaletteColor(palette, index);
                const label = document.createElement('label');
                label.style.cssText = 'display: flex; align-items: center; cursor: pointer; padding: 4px 8px; border-radius: 4px; background-color: #fff; border: 1px solid #ddd;';

                const checkbox = document.createElement('input');
                checkbox.type = 'checkbox';
                checkbox.checked = paletteVisibility[palette];
                checkbox.style.cssText = 'margin-right: 6px; cursor: pointer;';
                checkbox.addEventListener('change', function() {{
                    paletteVisibility[palette] = this.checked;
                    updateChart();
                }});

                const colorSwatch = document.createElement('span');
                colorSwatch.style.cssText = `display: inline-block; width: 12px; height: 12px; background-color: ${{color}}; margin-right: 6px; border-radius: 2px;`;

                const text = document.createElement('span');
                text.textContent = palette;

                label.appendChild(checkbox);
                label.appendChild(colorSwatch);
                label.appendChild(text);
                container.appendChild(label);
            }});
        }}

        initializeDatePickers();
        generatePaletteCheckboxes();
        updateChart();
    </script>
</body>
</html>
"""

    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(html)

    print(f"Generated {output_file}")

def main():
    """Main function"""
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <palettes_directory> <output_html_file>")
        print(f"  palettes_directory - directory containing location subdirectories with JSON files")
        print(f"  output_html_file   - path for the generated HTML report")
        return 1

    palettes_dir = sys.argv[1]
    output_file = sys.argv[2]
    script_dir = os.path.dirname(os.path.abspath(__file__))

    print("Space Palette Usage Analysis")
    print("=" * 50)

    data, time_of_day_data, session_duration_data, all_sessions, palette_colors, restart_data = analyze_all_palettes(palettes_dir, script_dir)

    if not data:
        print("No data found to analyze.")
        return 1

    print(f"\nAnalyzed {len(data)} days")

    generate_html(data, time_of_day_data, session_duration_data, all_sessions, palette_colors, restart_data, output_file)

    print(f"\nDone! Open {output_file} in your browser.")
    return 0

if __name__ == '__main__':
    sys.exit(main())
