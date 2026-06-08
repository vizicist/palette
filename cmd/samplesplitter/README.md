# SampleSplitter

Polyphonic MIDI-driven MP3 sample splitter and player with a browser UI.

## Implementation Status

The Go implementation in this directory is the reference implementation for
Palette and for new SampleSplitter work.

The Python implementation (`samplesplitter.py` plus helper scripts) is retained
as a legacy standalone tool. It is useful for comparison, manual testing, and
running outside Palette, but new behavior should be implemented in Go first.
Palette builds and runtime should use the Go `samplesplitter.exe` or the
in-engine Go service, not `samplesplitter.py`.

See `CONTRACT.md` for the compatibility contract that both implementations
should preserve where practical.

## Features

- Browser UI at `http://localhost:9876` — select MP3s, adjust split settings, view waveform
- Split by silence detection or fixed intervals — re-splits live as you change settings
- Waveform visualisation with labelled split markers (MIDI note names + timestamps)
- Each split mapped to a MIDI note (starting at C3 by default)
- Polyphonic playback — multiple splits simultaneously
- Pitch bend for pitch shifting
- Velocity controls volume
- Note-off stops playback
- MIDI port selectable in the UI

## Go Usage

From this directory:

```bash
go run .
```

MP3s are always loaded from `%USERPROFILE%\mp3s`. The `--dir` flag is retained
for compatibility but is ignored.

Build:

```bash
go build .
```

The Go version uses the bundled `ffmpeg/bin/ffmpeg.exe` on Windows when
available, then falls back to `ffmpeg` on PATH.

## Legacy Python Usage

```bash
py -3.11 samplesplitter.py
```

The Python version requires Python 3.11, `pyo`, `mido`, `python-rtmidi`, and
`ffmpeg`. It opens a browser at `http://localhost:9876` automatically unless
`--no-open` is supplied.

MP3s are always loaded from `%USERPROFILE%\mp3s`.

```
Options:
  --dir DIR          Ignored; MP3 directory is always %USERPROFILE%\mp3s
  --port PORT        HTTP port (default: 9876)
  --base-note NOTE   MIDI note number for split 0 (default: 48 = C3)
```

## MIDI Mapping

| Control | Action |
|---|---|
| Note on (base + N) | Play split N |
| Note off | Stop that split |
| Velocity | Volume (0–127 → 0–1) |
| Pitch bend | Pitch shift |
