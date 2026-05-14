# SampleSplitter Go Port

This directory now contains the beginning of a Go port alongside the existing
Python implementation. The Python script remains the behavioral reference until
the Go port reaches feature parity.

## First Slice

- `cmd/samplesplitter` starts an HTTP server with a Python-compatible CLI shape:
  `--dir`, `--port`, `--base-note`, `--midi-port`, and `--no-open`.
- `internal/samplesplitter` contains config/state structs, MP3 discovery,
  filename-safe media resolution, WAV-backed analysis through `ffmpeg`, cue data,
  waveform generation, and a small HTTP API skeleton.
- Implemented API endpoints include `/api/files`, `/api/media`, `/api/state`,
  `/api/analyze`, `/api/set_base_note`, `/api/set_peak_start`, `/api/stop_all`,
  `/api/midi_ports`, and `/api/audio_outputs`.
- Audio playback is implemented for preview and MIDI-triggered sample segments
  using `ffmpeg` decode-to-PCM plus a `malgo`/miniaudio callback output backend.
  Browser auto-open is intentionally not implemented yet.

## Compatibility Notes From `samplesplitter.py`

- Default split mode is `words`, matching the recent Python behavior.
- Words mode now supports `words_per_split`, defaulting to 2 words per split.
  Grouped splits keep their peak start at the peak of the first word in each
  group, matching the Python reference.
- Peak-start playback is represented in state and defaults to enabled. The Go
  port computes `peak_starts`, but does not play audio yet.
- Startup sigil loading matches the Python prefix convention:
  `chaos`, `oracle`, `sacred`, and `directive`.
- MIDI input, pitch bend state, note-off planning, and MIDI channel-to-sigil
  routing are implemented at the control/state layer:
  channel 0 -> chaos, 1 -> oracle, 2 -> sacred, 3 -> directive.
- Base note defaults to 48. The Python behavior where low MIDI notes below
  `base_note` proportionally select splits is represented in playback requests.
- The default MIDI port preference is `16. Internal MIDI`.
- Audio playback caches decoded MP3 PCM in memory, renders requested split
  segments with pitch ratio and an 8 ms edge fade, loops held MIDI notes, and
  stops active voices on note-off.
- One-shot preview voices now remove themselves from active state after their
  PCM buffer should have finished, while looped MIDI voices remain active until
  note-off or explicit stop.
- The Go audio backend enumerates miniaudio playback devices through
  `/api/audio_outputs` and opens a selected device through
  `/api/set_audio_output?id=N`.
- Active audio voice names are now reflected in `/api/state`.

## Suggested Next Slices

1. Add focused parity tests comparing Go split output against Python for a small
   fixture WAV/MP3.
2. Add API/test coverage for MIDI-triggered playback-request planning.
3. Run hands-on parity testing against the Python version with real MP3/MIDI
   input and tune latency/buffering if needed.
4. Replace the simple render-time pitch shift with a higher-quality resampler if
   the nearest-neighbor stepping is audible on larger pitch bends.
