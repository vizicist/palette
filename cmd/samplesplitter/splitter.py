"""
splitter.py — Analyze an MP3 and generate split cue points.

Usage:
    python3 splitter.py <file.mp3> [options]

Options:
    --mode silence|fixed     Split mode (default: silence)
    --interval SECONDS       Interval for fixed mode (default: 10.0)
    --silence-thresh FLOAT   Silence threshold, 0.0-1.0 RMS (default: 0.01)
    --silence-min SECONDS    Min silence duration in seconds (default: 0.5)
    --output FILE            Output JSON cue file (default: <file>.cues.json)
"""

import argparse
import json
import sys
import subprocess
import struct
import wave
import tempfile
import os
import math
from pathlib import Path


def mp3_to_wav(mp3_path, wav_path):
    """Convert MP3 to WAV using ffmpeg."""
    result = subprocess.run(
        ["ffmpeg", "-y", "-i", str(mp3_path), "-ar", "44100", "-ac", "1", str(wav_path)],
        capture_output=True
    )
    if result.returncode != 0:
        print(f"ffmpeg error: {result.stderr.decode()}", file=sys.stderr)
        sys.exit(1)


def read_wav_rms_blocks(wav_path, block_size=0.05):
    """Read WAV file and return list of (time_sec, rms) for each block."""
    with wave.open(str(wav_path), "rb") as wf:
        frame_rate = wf.getframerate()
        n_frames = wf.getnframes()
        duration = n_frames / frame_rate
        block_frames = int(frame_rate * block_size)

        blocks = []
        t = 0.0
        while True:
            data = wf.readframes(block_frames)
            if not data:
                break
            # 16-bit mono samples
            samples = struct.unpack(f"{len(data)//2}h", data)
            rms = math.sqrt(sum(s * s for s in samples) / len(samples)) / 32768.0
            blocks.append((round(t, 4), rms))
            t += block_size

    return blocks, duration


def split_by_silence(blocks, duration, silence_thresh=0.01, min_silence_sec=0.5):
    """Return split points detected by silence regions."""
    block_size = blocks[1][0] - blocks[0][0] if len(blocks) > 1 else 0.05
    min_blocks = max(1, int(min_silence_sec / block_size))

    # Find runs of silent blocks
    silent = [rms < silence_thresh for _, rms in blocks]

    split_points = [0.0]
    i = 0
    while i < len(blocks):
        if silent[i]:
            run_start = i
            while i < len(blocks) and silent[i]:
                i += 1
            run_end = i
            if (run_end - run_start) >= min_blocks:
                # Use midpoint of silence region as split point
                mid_t = blocks[(run_start + run_end) // 2][0]
                if mid_t > 0.0:
                    split_points.append(round(mid_t, 4))
        else:
            i += 1

    return split_points


def split_by_interval(duration, interval_sec=10.0):
    """Return split points at fixed intervals."""
    splits = []
    t = 0.0
    while t < duration:
        splits.append(round(t, 4))
        t += interval_sec
    return splits


def main():
    parser = argparse.ArgumentParser(description="Generate split cue points for an MP3 file.")
    parser.add_argument("file", help="Input MP3 file")
    parser.add_argument("--mode", choices=["silence", "fixed"], default="silence",
                        help="Split mode (default: silence)")
    parser.add_argument("--interval", type=float, default=10.0,
                        help="Interval in seconds for fixed mode (default: 10.0)")
    parser.add_argument("--silence-thresh", type=float, default=0.01,
                        help="Silence threshold 0.0-1.0 RMS (default: 0.01)")
    parser.add_argument("--silence-min", type=float, default=0.5,
                        help="Minimum silence duration in seconds (default: 0.5)")
    parser.add_argument("--output", help="Output JSON cue file (default: <file>.cues.json)")
    args = parser.parse_args()

    mp3_path = Path(args.file)
    if not mp3_path.exists():
        print(f"Error: file not found: {mp3_path}", file=sys.stderr)
        sys.exit(1)

    output_path = Path(args.output) if args.output else mp3_path.with_suffix(".cues.json")

    print(f"Loading {mp3_path}...")

    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        wav_path = tmp.name

    try:
        mp3_to_wav(mp3_path, wav_path)
        blocks, duration = read_wav_rms_blocks(wav_path)
    finally:
        os.unlink(wav_path)

    print(f"Duration: {duration:.2f}s")

    if args.mode == "silence":
        print(f"Detecting splits by silence (thresh={args.silence_thresh} RMS, min={args.silence_min}s)...")
        splits = split_by_silence(blocks, duration,
                                  silence_thresh=args.silence_thresh,
                                  min_silence_sec=args.silence_min)
    else:
        print(f"Splitting at fixed intervals of {args.interval}s...")
        splits = split_by_interval(duration, interval_sec=args.interval)

    cue_data = {
        "file": str(mp3_path.resolve()),
        "duration": round(duration, 4),
        "mode": args.mode,
        "splits": splits,
        "num_splits": len(splits),
    }

    with open(output_path, "w") as f:
        json.dump(cue_data, f, indent=2)

    print(f"Found {len(splits)} split points:")
    for i, t in enumerate(splits):
        print(f"  [{i:3d}] {t:.3f}s  (MIDI note {48 + i})")
    print(f"Cue file saved to: {output_path}")


if __name__ == "__main__":
    main()
