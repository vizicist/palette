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

## Web UI Architecture

### Categories
The GUI organizes content into categories shown as tabs:
- **Global** - Global settings that affect the entire system
- **Quad** - Quad presets (combinations of all 4 patches)
- **Patch** - Individual patch presets
- **Misc/Sound/Visual/Effect** - Per-patch parameter categories

### Presets vs Parameters
Each category has two views, toggled by clicking the category tab again:
1. **Presets view** (default) - Grid of saved preset names that can be loaded
2. **Parameters view** - Vertical list of all parameters with current values and controls

### Patch Selector
- **A, B, C, D** buttons - Select a single patch for loading presets or editing params
- **\*** button - Select all patches (loads/edits apply to all 4 patches)
- Clicking **\*** toggles between all-selected and single-patch mode

### API Structure
APIs use `{type}.{action}` format:
- `global.status`, `global.get`, `global.set`, `global.load`
- `quad.load` - Load a quad preset
- `patch.load`, `patch.get`, `patch.set`, `patch.getparams` - Patch operations (require `patch` param: A/B/C/D)
- `saved.list`, `saved.paramdefs` - List presets or parameter definitions

## Logging

### Log Types
All logging is disabled by default. Available types:
- `api` - API calls
- `info`, `config`, `params`, `patch`, `load`
- `cursor`, `gesture`, `loop`, `midi`, `note`
- `process`, `attract`, `bidule`, `ffgl`, etc.
- `*` - Enable all log types

### Enable Logging at Runtime
Via API:
```
palette set global.log api
palette set global.log api,info,params
```

Or from browser console:
```javascript
API.call('global.set', { name: 'global.log', value: 'api' })
```

### Check Current Log Setting
```
palette get global.log
```

### Log File Location
Logs are written to `engine.log` in the logs directory.
