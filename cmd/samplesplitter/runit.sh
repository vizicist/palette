#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

PORT="${PORT:-9876}"
MP3_DIR="${MP3_DIR:-mp3s}"
URL="http://localhost:${PORT}/"
STATE_URL="http://localhost:${PORT}/api/state"
LOG_FILE="${TMPDIR:-/tmp}/samplesplitter-${PORT}.log"

if [[ -x ".venv/bin/python" ]]; then
    PYTHON=".venv/bin/python"
elif command -v python3.11 >/dev/null 2>&1; then
    PYTHON="$(command -v python3.11)"
elif command -v python3 >/dev/null 2>&1; then
    PYTHON="$(command -v python3)"
else
    echo "Python 3 was not found. Install Python 3.11, then try again." >&2
    exit 1
fi

if [[ ! -d "$MP3_DIR" ]]; then
    echo "MP3 directory not found: $MP3_DIR" >&2
    exit 1
fi

echo "Stopping any existing Sample Splitter servers..."
if command -v pgrep >/dev/null 2>&1; then
    while IFS= read -r PID; do
        [[ -z "$PID" || "$PID" == "$$" ]] && continue
        kill "$PID" >/dev/null 2>&1 || true
    done < <(pgrep -f "[s]amplesplitter.py" || true)
    sleep 1
    while IFS= read -r PID; do
        [[ -z "$PID" || "$PID" == "$$" ]] && continue
        kill -9 "$PID" >/dev/null 2>&1 || true
    done < <(pgrep -f "[s]amplesplitter.py" || true)
fi

if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Port $PORT is already in use, but it does not look like Sample Splitter." >&2
    lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >&2 || true
    echo "Set another port with: PORT=9877 ./runit.sh" >&2
    exit 1
fi

if ! "$PYTHON" -c "import pyo, mido, rtmidi" >/dev/null 2>&1; then
    echo "Python dependencies are missing for $PYTHON." >&2
    echo "Run:" >&2
    echo "  $PYTHON -m pip install pyo mido python-rtmidi" >&2
    exit 1
fi

echo "Starting Sample Splitter on $URL"
PYTHONUNBUFFERED=1 "$PYTHON" samplesplitter.py --dir "$MP3_DIR" --port "$PORT" --no-open >"$LOG_FILE" 2>&1 &
SERVER_PID=$!

cleanup() {
    if kill -0 "$SERVER_PID" >/dev/null 2>&1; then
        kill "$SERVER_PID" >/dev/null 2>&1 || true
        wait "$SERVER_PID" >/dev/null 2>&1 || true
    fi
}
trap cleanup INT TERM EXIT

echo "Waiting for the web interface..."
for _ in $(seq 1 30); do
    if ! kill -0 "$SERVER_PID" >/dev/null 2>&1; then
        echo "The server stopped before the web interface became available." >&2
        echo "Server log:" >&2
        sed -n '1,120p' "$LOG_FILE" >&2 || true
        wait "$SERVER_PID"
        exit 1
    fi

    STATE="$(curl -fsS "$STATE_URL" 2>/dev/null || true)"
    if [[ -n "$STATE" ]] && "$PYTHON" -c 'import json, sys; state = json.load(sys.stdin); sys.exit(0 if state.get("pyo_ready") or state.get("audio_error") else 1)' <<<"$STATE"; then
        open "$URL"
        echo "Sample Splitter is running. Press Ctrl+C to quit."
        echo "Server log: $LOG_FILE"
        wait "$SERVER_PID"
        exit $?
    fi

    sleep 1
done

echo "The server did not respond on $URL within 30 seconds." >&2
echo "Server log:" >&2
sed -n '1,120p' "$LOG_FILE" >&2 || true
exit 1
