# SampleSplitter Compatibility Contract

The Go implementation is the reference implementation. The legacy Python
implementation lives in the standalone `vizicist/samplesplitter` repository.

This contract describes behavior that should stay stable across the Go
standalone server, the in-engine Go service, and the external legacy Python
server where practical.

## CLI

The standalone command accepts these flags:

- `--dir`: accepted for compatibility but ignored. MP3s are always loaded from
  `%USERPROFILE%\mp3s`.
- `--port`: HTTP port, default `9876`.
- `--base-note`: MIDI note for split 0, default `48`.
- `--midi-port`: preferred MIDI input port.
- `--no-open`: accepted for compatibility. Go currently accepts it without
  opening a browser.

## HTTP API

The browser UI and Palette code may rely on these endpoints:

- `GET /`
- `GET /api/state`
- `GET /api/files`
- `GET /api/media`
- `POST /api/analyze`
- `POST /api/set_base_note`
- `POST /api/set_peak_start`
- `POST /api/set_pitch_bend`
- `POST /api/set_audio_output`
- `POST /api/set_compressed`
- `POST /api/reload_sigils`
- `POST /api/stop_all`
- `POST /api/preview_on`
- `POST /api/preview_off`
- `GET /api/midi_ports`
- `GET /api/audio_outputs`

Endpoint responses should include an `ok` boolean for mutation endpoints when
possible, and `/api/state` should expose enough state for the UI to redraw
without hidden local assumptions.

## Splitting

- Default split mode is `words`.
- Default words per split is `2`.
- Words-per-split accepts values `1` through `5`.
- When a split groups multiple words, the peak-start point is the peak of the
  first word in that group.
- Peak-start playback defaults to enabled.

## Sigil Sample Sets

- The four sigils are `chaos`, `oracle`, `sacred`, and `directive`.
- Sigil rows/pads map to channels:
  - channel 0: chaos
  - channel 1: oracle
  - channel 2: sacred
  - channel 3: directive
- Startup/reload should select one MP3 per sigil from filenames beginning with
  that sigil name.
- Selection may be randomized, but every sigil should have an independent
  current sample.

## Playback

- MIDI note `base-note + N` plays split `N`.
- MIDI notes below `base-note` proportionally select a split across the loaded
  cue data.
- Velocity controls playback volume.
- Pitch bend controls pitch over a 24-semitone range, centered at MIDI pitch
  bend value 8192.
- Note-off stops the corresponding held note/sample.
- Held notes/fingers loop until stopped.
- Multiple held notes/fingers on one sigil channel are retained and cycled as
  each loop reaches its end, rather than replacing everything with the last
  note/finger.
- A stop-all action stops all active voices.

## Audio

- Audio output devices should be enumerable and selectable.
- MP3 decoding uses FFmpeg. On Windows, Go should prefer the bundled
  `ffmpeg/bin/ffmpeg.exe`.
- Sample playback should apply a short edge fade to reduce clicks.
- Optional compression/normalization may be enabled by config.

## Palette Integration

- Palette's in-engine SamplePlayback uses the Go service directly.
- Palette should not depend on MIDI routing into the in-engine service.
- Standalone Go still supports MIDI input for external use.
