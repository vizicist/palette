# SampleSplitter Implementation Notes

The Go implementation is now the reference implementation.

The Python implementation remains in this directory as a legacy standalone
tool. It should not be treated as the behavior source for new work. When the
two implementations differ, prefer changing Python to match Go, unless there is
a clear bug in the Go implementation.

## Current Go Responsibilities

- Run the standalone HTTP server from `cmd/samplesplitter`.
- Provide the reusable `pkg/samplesplitter` service used by Palette's in-engine
  Transmission playback.
- Own MP3 discovery, cue analysis, waveform data, audio-device selection, MIDI
  input, sample playback, pitch bend, compression, words-per-chunk splitting,
  sigil sample selection, and held-note voice cycling.
- Use bundled `ffmpeg/bin/ffmpeg.exe` on Windows when available, falling back to
  `ffmpeg` on PATH.

## Legacy Python Responsibilities

- Keep `samplesplitter.py` runnable as a standalone tool for manual testing and
  historical comparison.
- Preserve familiar CLI flags where practical.
- Avoid adding new Palette-dependent behavior to Python unless it is needed for
  standalone parity.

## Migration Rule

New features should be implemented in Go first. If the Python standalone tool
still needs that behavior, add it afterward as a compatibility update.

See `CONTRACT.md` for the runtime/API contract that both implementations should
preserve where practical.
