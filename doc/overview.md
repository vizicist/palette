# Palette

Palette is the software used in the
<a href=https://youtu.be/HDtxEyCI_zc>Space Palette Pro</a>, an instrument that
lets you fingerpaint sound and visuals using your fingers as three-dimensional
cursors on Sensel Morph pads.

The system is centered around a Go runtime engine. That engine receives pad,
MIDI, OSC, and API input; schedules musical events on a shared clock; controls
sound and visuals; serves the browser UI; manages presets and parameters; and
starts or monitors the external programs used by the installation.

For a deeper module-by-module map, see
<a href=architecture.md>Current architecture summary and diagram</a>.

# Main Open Source Parts

## palette_engine

`palette_engine` is the central runtime process. It is responsible for:

- Reading Sensel Morph pad input and other cursor/MIDI/OSC inputs.
- Running the click clock, scheduler, looping system, and Stepper sequencer.
- Converting cursor gestures into MIDI notes, visual OSC messages, and direct
  SamplePlayback.
- Serving the browser-based UI at `http://127.0.0.1:3330/`.
- Exposing the local JSON API at `http://127.0.0.1:3330/api`.
- Loading and saving global, quad, and patch presets.
- Starting and stopping managed processes such as Bidule, Resolume, OBS, and
  the in-engine SampleSplitter service.

The normal Oscillation/Synth path sends MIDI to Bidule/Omnisphere and sends OSC
to Resolume and the Palette FFGL plugin. The newer SamplePlayback path uses the
Go SampleSplitter service inside the engine for direct sample playback from cursor
gestures.

## Browser UI

The current UI is browser-based and embedded directly into `palette_engine` from
`kit/webui`. It replaces the older Python GUI as the main interactive control
surface.

The UI supports preset selection, parameter editing, advanced mode, virtual pad
mode selection, SamplePlayback controls, Stepper/Sequencer display, and startup
page selection through `global.mode`.

Because the UI files are embedded, changes under `kit/webui` require rebuilding
`palette_engine`.

## palette CLI

`palette`, built from `cmd/palette`, is the command-line control tool. It calls
the engine API to start and stop the system, query status, inspect and set
global parameters, inspect and set patch parameters, load presets, run tests,
and manage boot values.

Common examples include:

```bat
palette status
palette get global.log
palette set global.process.resolume true
palette patchget A visual
palette patchset * stepper.route samplesplitter
```

## palette_monitor

`palette_monitor` is a small watchdog process that can restart
`palette_engine` if it exits unexpectedly. It is part of the reliability layer
for installations that should keep running unattended.

## SampleSplitter

SampleSplitter is an in-engine Go service from `pkg/samplesplitter`, used by
Palette SamplePlayback. Its browser UI and ffmpeg runtime assets live under
`pkg/samplesplitter/assets`.

SampleSplitter loads MP3 files, splits them into playable chunks, optionally
compresses/normalizes them, supports peak-position playback, exposes a browser
UI on port 9876, and plays audio through a selected output device. The bundled
`ffmpeg.exe` is used for MP3 decoding on Windows.

## Palette.dll

`Palette.dll` is the FreeFrame/FFGL visual plugin used by Resolume. The engine
sends cursor and parameter data to Resolume/FFGL over OSC. The FFGL plugin draws
and animates the visual shapes, while Resolume hosts the layer and effect chain.

The plugin source lives under `ffgl`.

# External Software Parts

## Resolume

Resolume hosts the visual performance pipeline. The Palette FFGL plugin draws
the base sprites/shapes, and Resolume effects process the result. Palette
controls the four main visual layers, corresponding to pads A, B, C, and D.

## Bidule and Omnisphere

Bidule is the VST host used for the main synth sound path. Palette sends MIDI to
Bidule, and Bidule routes that MIDI to Omnisphere instances and their configured
multis.

## LoopBe30

LoopBe30 provides virtual MIDI ports on Windows. Palette and Bidule use these
ports to communicate locally by MIDI.

## OBS and Other Optional Processes

Palette can also manage optional supporting processes such as OBS, chat tools,
and other installation-specific utilities through the same process manager and
`global.process.*` parameter model.

# Runtime Data

Palette uses factory defaults from `data_default` plus a per-user runtime data
area for mutable state. Runtime data includes saved parameters, presets, logs,
SampleSplitter MP3s, current state, and boot configuration.

Important concepts:

- Global parameters affect the whole system.
- Quad presets describe all four pads together.
- Patch presets describe one pad at a time.
- Boot values define what should be active after startup.
- Logs are controlled by `global.log` and written to the runtime logs
  directory.

# Build and Install

Windows build and install scripts live under `build/windows`. The installed
system is designed around a per-user install location. The build includes the Go
executables, embedded web UI, default data, FFGL plugin artifacts, SampleSplitter
assets, and the bundled `ffmpeg.exe`.

`doit.bat` is the local build/install/run convenience script and should be run
from `build/windows`. `testit.bat` runs the current test suite, including the
minimal browser UI smoke test.
Manual MIDI diagnostics such as `cmd/miditest` are kept in the tree but are not
part of regression testing.
