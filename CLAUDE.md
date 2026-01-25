# Palette Project Notes

## Build Instructions

### palette_engine
Build from the cmd/palette_engine directory:
```
cd cmd/palette_engine && go build .
```
The exe stays in cmd/palette_engine/. Don't try to execute it from Claude - the user will run it manually.

### Other commands
- `palette` CLI: `cd cmd/palette && go build .`
- `palette_hub`: `cd cmd/palette_hub && go build .`

## Web UI
The web UI files are in `kit/webui/` and are embedded into the engine binary using Go's embed package. After modifying any files in `kit/webui/`, you must rebuild palette_engine for changes to take effect.

## Running
- Start engine only (no auto-start processes): `palette start engineonly`
- Web UI available at: http://127.0.0.1:3330/
- API endpoint: http://127.0.0.1:3330/api

## Project Structure
- `kit/` - Core Go library
- `cmd/palette/` - CLI tool
- `cmd/palette_engine/` - Engine binary
- `cmd/palette_hub/` - Hub service
- `python/` - Python GUI and utilities (being replaced by web UI)
