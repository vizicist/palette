#!/usr/bin/env python3
"""
samplesplitter.py — MIDI-driven MP3 sample splitter and polyphonic player.

Legacy standalone implementation. The Go implementation in cmd/samplesplitter
and pkg/samplesplitter is the reference implementation for Palette behavior.

Usage:
    python3 samplesplitter.py --dir /path/to/mp3s [--port 9876] [--base-note 48]

Opens a browser UI at http://localhost:9876 for file selection, splitting,
and MIDI port configuration. Plays back splits polyphonically via pyo.
"""

import argparse
import json
import math
import os
import random
import struct
import subprocess
import sys
import tempfile
import threading
import time
import webbrowser
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse

try:
    import mido
except ImportError:
    mido = None

try:
    import pyo
except Exception as e:
    PYO_IMPORT_ERROR = e
    pyo = None
else:
    PYO_IMPORT_ERROR = None

# ---------------------------------------------------------------------------
# ffmpeg path resolution
# ---------------------------------------------------------------------------

def find_ffmpeg():
    """Find ffmpeg binary: check script-relative ffmpeg/bin/ first, then PATH."""
    script_dir = Path(__file__).parent.resolve()
    bin_dir = script_dir / "ffmpeg" / "bin"
    candidates = [bin_dir / ("ffmpeg.exe" if os.name == "nt" else "ffmpeg")]
    for c in candidates:
        if c.exists() and os.access(c, os.X_OK):
            return str(c)
    # Fall back to PATH
    import shutil
    found = shutil.which("ffmpeg")
    if found:
        return found
    print("Error: ffmpeg not found. Put ffmpeg/bin/ffmpeg next to this script, or add ffmpeg to PATH.",
          file=sys.stderr)
    sys.exit(1)

FFMPEG = find_ffmpeg()

# ---------------------------------------------------------------------------
# Global state (shared between HTTP handler and player thread)
# ---------------------------------------------------------------------------

state = {
    "mp3_dir": None,
    "current_file": None,
    "cue_data": None,
    "waveform": None,          # list of floats 0.0-1.0 (downsampled RMS)
    "sigil_samples": {},
    "midi_port_name": None,
    "midi_error": None,
    "midi_activity_count": 0,
    "midi_activity_time": None,
    "base_note": 48,
    "peak_start_enabled": True,
    "pitch_bend_semitones_preview": 0.0,
    "pitch_bend_semitones": {},
    "active_voices": {},       # voice key -> {"player", "stop_cb", "channel", "fade"}
    "voice_order": [],         # oldest -> newest active voice keys
    "midi_note_voices": {},    # MIDI note -> queued voice keys
    "midi_voice_counter": 0,
    "pyo_server": None,
    "pyo_ready": False,
    "audio_error": None,
    "audio_output_id": None,
    "audio_output_name": None,
    "midi_thread": None,
    "midi_stop": threading.Event(),
}

state_lock = threading.Lock()

PITCH_BEND_RANGE = 12.0
PITCH_BEND_MAX = 8192
WAVEFORM_POINTS = 1200        # number of amplitude points sent to browser
MAX_ACTIVE_VOICES = 48
MAX_MIDI_VOICES_PER_NOTE = 8
SAMPLE_EDGE_FADE_SECONDS = 0.008
DEFAULT_WORDS_PER_SPLIT = 2
SIGIL_BY_MIDI_CHANNEL = {
    0: "chaos",
    1: "oracle",
    2: "sacred",
    3: "directive",
}


def get_midi_input_names():
    if mido is None:
        return [], "mido is not installed"
    try:
        return mido.get_input_names(), None
    except Exception as e:
        return [], f"MIDI backend unavailable: {e}"


def resolve_midi_input_name(port_name):
    ports, error = get_midi_input_names()
    if error:
        raise RuntimeError(error)
    if port_name in ports:
        return port_name
    matches = [name for name in ports if name.startswith(port_name)]
    if len(matches) == 1:
        return matches[0]
    if len(matches) > 1:
        raise RuntimeError(f"ambiguous port '{port_name}': {', '.join(matches)}")
    raise RuntimeError(f"unknown port '{port_name}'")


def get_audio_output_devices():
    if pyo is None:
        return [], None, "pyo is not installed"
    try:
        _, output_infos = pyo.pa_get_devices_infos()
        default_id = pyo.pa_get_default_output()
        preferred_host_order = {0: 0, 1: 1, 3: 2, 4: 3}
        by_name = {}
        for device_id, info in output_infos.items():
            name = info["name"]
            host_api = info["host api index"]
            normalized = name.lower()
            if (
                host_api == 4 and (
                    normalized.startswith("speakers 1 ") or
                    normalized.startswith("speakers 2 ") or
                    normalized.startswith("headphones 1 ") or
                    normalized.startswith("headphones 2 ")
                )
            ):
                continue
            rank = preferred_host_order.get(host_api, 99)
            if int(device_id) == int(default_id):
                rank = -1
            current = by_name.get(normalized)
            if current is None or rank < current["rank"]:
                by_name[normalized] = {
                    "id": int(device_id),
                    "name": name,
                    "rank": rank,
                }

        devices = sorted(
            (
                {"id": d["id"], "name": d["name"]}
                for d in by_name.values()
            ),
            key=lambda d: (0 if d["id"] == int(default_id) else 1, d["name"].lower())
        )
        return devices, int(default_id), None
    except Exception as e:
        return [], None, f"Audio outputs unavailable: {e}"


# ---------------------------------------------------------------------------
# Audio analysis
# ---------------------------------------------------------------------------

def mp3_to_wav(mp3_path, wav_path):
    result = subprocess.run(
        [FFMPEG, "-y", "-i", str(mp3_path), "-ar", "44100", "-ac", "1", str(wav_path)],
        capture_output=True
    )
    if result.returncode != 0:
        raise RuntimeError(f"ffmpeg failed: {result.stderr.decode()}")


def read_wav(wav_path):
    """Return (samples_float[], frame_rate, duration_sec)."""
    import wave
    with wave.open(str(wav_path), "rb") as wf:
        frame_rate = wf.getframerate()
        n_frames = wf.getnframes()
        raw = wf.readframes(n_frames)
    samples = struct.unpack(f"{len(raw)//2}h", raw)
    floats = [s / 32768.0 for s in samples]
    duration = n_frames / frame_rate
    return floats, frame_rate, duration


def compute_waveform(samples, num_points=WAVEFORM_POINTS):
    """Downsample to num_points RMS values, normalised 0-1."""
    block = max(1, len(samples) // num_points)
    out = []
    for i in range(num_points):
        chunk = samples[i * block: (i + 1) * block]
        if not chunk:
            out.append(0.0)
        else:
            rms = math.sqrt(sum(s * s for s in chunk) / len(chunk))
            out.append(rms)
    peak = max(out) or 1.0
    return [v / peak for v in out]


def compute_peak_starts(samples, frame_rate, splits, duration):
    peak_starts = []
    total_samples = len(samples)
    for i, start in enumerate(splits):
        end = splits[i + 1] if i + 1 < len(splits) else duration
        start_idx = max(0, min(total_samples, int(start * frame_rate)))
        end_idx = max(start_idx + 1, min(total_samples, int(end * frame_rate)))
        chunk = samples[start_idx:end_idx]
        if not chunk:
            peak_starts.append(round(start, 4))
            continue
        peak_offset = max(range(len(chunk)), key=lambda idx: abs(chunk[idx]))
        peak_starts.append(round((start_idx + peak_offset) / frame_rate, 4))
    return peak_starts


def compute_first_word_peak_starts(samples, frame_rate, grouped_splits, word_splits, duration):
    peak_starts = []
    if not word_splits:
        return compute_peak_starts(samples, frame_rate, grouped_splits, duration)

    total_samples = len(samples)
    for grouped_start in grouped_splits:
        word_index = min(
            range(len(word_splits)),
            key=lambda idx: abs(word_splits[idx] - grouped_start)
        )
        start = word_splits[word_index]
        end = word_splits[word_index + 1] if word_index + 1 < len(word_splits) else duration
        start_idx = max(0, min(total_samples, int(start * frame_rate)))
        end_idx = max(start_idx + 1, min(total_samples, int(end * frame_rate)))
        chunk = samples[start_idx:end_idx]
        if not chunk:
            peak_starts.append(round(start, 4))
            continue
        peak_offset = max(range(len(chunk)), key=lambda idx: abs(chunk[idx]))
        peak_starts.append(round((start_idx + peak_offset) / frame_rate, 4))
    return peak_starts


def detect_splits_silence(samples, frame_rate, duration,
                           silence_thresh=0.01, min_silence_sec=0.5):
    block_sec = 0.05
    block_size = int(frame_rate * block_sec)
    min_blocks = max(1, int(min_silence_sec / block_sec))
    num_blocks = len(samples) // block_size

    silent = []
    for i in range(num_blocks):
        chunk = samples[i * block_size: (i + 1) * block_size]
        rms = math.sqrt(sum(s * s for s in chunk) / len(chunk))
        silent.append(rms < silence_thresh)

    splits = [0.0]
    i = 0
    while i < len(silent):
        if silent[i]:
            run_start = i
            while i < len(silent) and silent[i]:
                i += 1
            run_end = i
            if (run_end - run_start) >= min_blocks:
                mid_t = ((run_start + run_end) // 2) * block_sec
                if mid_t > 0.0:
                    splits.append(round(mid_t, 4))
        else:
            i += 1
    return splits


def detect_splits_fixed(duration, interval_sec=10.0):
    splits = []
    t = 0.0
    while t < duration:
        splits.append(round(t, 4))
        t += interval_sec
    return splits


def detect_splits_words(samples, frame_rate, duration,
                        silence_thresh=0.01, min_silence_sec=0.12,
                        min_word_sec=0.16, max_word_sec=0.65):
    block_sec = 0.01
    block_size = max(1, int(frame_rate * block_sec))
    min_gap_blocks = max(1, int(min_silence_sec / block_sec))
    min_word_blocks = max(1, int(min_word_sec / block_sec))
    max_word_blocks = max(min_word_blocks + 1, int(max_word_sec / block_sec))
    num_blocks = len(samples) // block_size

    if num_blocks == 0:
        return [0.0]

    rms_values = []
    for i in range(num_blocks):
        chunk = samples[i * block_size: (i + 1) * block_size]
        rms = math.sqrt(sum(s * s for s in chunk) / len(chunk))
        rms_values.append(rms)

    smooth_radius = 3
    envelope = []
    for i in range(len(rms_values)):
        start = max(0, i - smooth_radius)
        end = min(len(rms_values), i + smooth_radius + 1)
        envelope.append(sum(rms_values[start:end]) / (end - start))

    sorted_rms = sorted(rms_values)
    noise_floor = sorted_rms[max(0, int(len(sorted_rms) * 0.2) - 1)]
    peak = max(envelope) or 1.0
    threshold = max(silence_thresh, noise_floor * 3.0, peak * 0.04)
    voiced = [rms >= threshold for rms in envelope]

    runs = []
    i = 0
    while i < len(voiced):
        if voiced[i]:
            start = i
            while i < len(voiced) and voiced[i]:
                i += 1
            runs.append([start, i])
        else:
            i += 1

    if not runs:
        return [0.0]

    merged = [runs[0]]
    for start, end in runs[1:]:
        prev = merged[-1]
        if start - prev[1] < min_gap_blocks:
            prev[1] = end
        else:
            merged.append([start, end])

    split_blocks = []
    for start, end in merged:
        if end - start >= min_word_blocks:
            split_blocks.append(start)

            segment_start = start
            while segment_start + max_word_blocks < end:
                search_start = segment_start + min_word_blocks
                search_end = min(segment_start + max_word_blocks, end - min_word_blocks)
                if search_end <= search_start:
                    break

                valley = min(
                    range(search_start, search_end),
                    key=lambda idx: envelope[idx]
                )
                local_peak = max(envelope[segment_start:search_end]) or peak
                if envelope[valley] < local_peak * 0.75:
                    split_blocks.append(valley)
                    segment_start = valley
                else:
                    segment_start += max_word_blocks

    if not split_blocks:
        return [0.0]

    split_blocks = sorted(set(split_blocks))
    splits = []
    for block in split_blocks:
        t = round(block * block_sec, 4)
        if not splits or t - splits[-1] >= min_word_sec:
            splits.append(t)

    if splits[0] > 0.05:
        splits.insert(0, 0.0)
    else:
        splits[0] = 0.0
    return splits


def group_word_splits(splits, words_per_split=DEFAULT_WORDS_PER_SPLIT):
    words_per_split = max(1, int(words_per_split or DEFAULT_WORDS_PER_SPLIT))
    if words_per_split == 1 or len(splits) <= 1:
        return splits
    grouped = splits[::words_per_split]
    if grouped[0] != 0.0:
        grouped.insert(0, 0.0)
    return grouped


def analyze_file(mp3_path, mode="silence", interval=10.0,
                 silence_thresh=0.01, silence_min=0.5,
                 words_per_split=DEFAULT_WORDS_PER_SPLIT):
    """Return (cue_data dict, waveform list)."""
    with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp:
        wav_path = tmp.name
    try:
        mp3_to_wav(mp3_path, wav_path)
        samples, frame_rate, duration = read_wav(wav_path)
    finally:
        os.unlink(wav_path)

    waveform = compute_waveform(samples)

    if mode == "silence":
        splits = detect_splits_silence(samples, frame_rate, duration,
                                       silence_thresh=silence_thresh,
                                       min_silence_sec=silence_min)
        peak_starts = compute_peak_starts(samples, frame_rate, splits, duration)
    elif mode == "words":
        word_splits = detect_splits_words(samples, frame_rate, duration,
                                          silence_thresh=silence_thresh,
                                          min_silence_sec=silence_min)
        splits = group_word_splits(word_splits, words_per_split)
        peak_starts = compute_first_word_peak_starts(samples, frame_rate, splits, word_splits, duration)
    else:
        splits = detect_splits_fixed(duration, interval_sec=interval)
        peak_starts = compute_peak_starts(samples, frame_rate, splits, duration)

    cue_data = {
        "file": str(mp3_path),
        "duration": round(duration, 4),
        "mode": mode,
        "words_per_split": words_per_split if mode == "words" else None,
        "splits": splits,
        "peak_starts": peak_starts,
        "num_splits": len(splits),
    }
    return cue_data, waveform


# ---------------------------------------------------------------------------
# Pyo player
# ---------------------------------------------------------------------------

def boot_pyo_server(output_id=None):
    server = pyo.Server(duplex=0)
    if output_id is not None:
        server.setOutputDevice(int(output_id))
    server.boot()
    server.start()
    return server


def init_pyo():
    if pyo is None:
        msg = "pyo is not installed"
        if PYO_IMPORT_ERROR is not None:
            msg = f"pyo is unavailable: {PYO_IMPORT_ERROR}"
        with state_lock:
            state["audio_error"] = msg
            state["pyo_ready"] = False
        print(f"Audio disabled: {msg}.", file=sys.stderr)
        return

    with state_lock:
        output_id = state["audio_output_id"]

    try:
        server = boot_pyo_server(output_id)
        devices, default_id, _ = get_audio_output_devices()
    except Exception as e:
        with state_lock:
            state["audio_error"] = f"pyo failed to start: {e}"
            state["pyo_ready"] = False
            state["pyo_server"] = None
        print(f"Audio disabled: pyo failed to start: {e}", file=sys.stderr)
        return
    active_id = output_id if output_id is not None else default_id
    active_name = next((d["name"] for d in devices if d["id"] == active_id), None)
    with state_lock:
        state["pyo_server"] = server
        state["pyo_ready"] = True
        state["audio_error"] = None
        state["audio_output_id"] = active_id
        state["audio_output_name"] = active_name
    print("pyo audio server ready.")


def set_audio_output(output_id):
    if pyo is None:
        raise RuntimeError("pyo is not installed")

    devices, _, error = get_audio_output_devices()
    if error:
        raise RuntimeError(error)
    output_id = int(output_id)
    if output_id not in {d["id"] for d in devices}:
        raise RuntimeError("audio output device not found")

    player_stop_all()
    with state_lock:
        old_server = state["pyo_server"]
        state["pyo_ready"] = False
        state["pyo_server"] = None

    if old_server:
        old_server.stop()
        old_server.shutdown()

    server = boot_pyo_server(output_id)
    active_name = next((d["name"] for d in devices if d["id"] == output_id), None)
    with state_lock:
        state["pyo_server"] = server
        state["pyo_ready"] = True
        state["audio_error"] = None
        state["audio_output_id"] = output_id
        state["audio_output_name"] = active_name
    return active_name


def semitones_to_ratio(semitones):
    return 2.0 ** (semitones / 12.0)


def sample_for_channel(channel):
    sigil = SIGIL_BY_MIDI_CHANNEL.get(channel)
    if sigil:
        sample = state["sigil_samples"].get(sigil)
        if sample and sample.get("current_file") and sample.get("cue_data"):
            return sample
    return {
        "sigil": None,
        "current_file": state["current_file"],
        "cue_data": state["cue_data"],
        "waveform": state["waveform"],
    }


def player_note_on(note, velocity, channel=None):
    with state_lock:
        if not state["pyo_ready"]:
            return
        sample = sample_for_channel(channel)
        cue_data = sample.get("cue_data")
        mp3_path = sample.get("current_file")
        if cue_data is None or mp3_path is None:
            return
        base_note = state["base_note"]
        splits = cue_data["splits"]
        if note < base_note:
            split_index = min(len(splits) - 1, max(0, int((note / max(1, base_note)) * len(splits))))
        else:
            split_index = note - base_note
        if split_index < 0 or split_index >= len(splits):
            return

        _stop_channel_voices(channel)

        midi_key = (channel, note)
        note_voice_keys = state["midi_note_voices"].setdefault(midi_key, [])
        while len(note_voice_keys) >= MAX_MIDI_VOICES_PER_NOTE:
            _stop_voice(note_voice_keys[0])
            note_voice_keys = state["midi_note_voices"].setdefault(midi_key, [])

        voice_key = f"midi-{channel if channel is not None else 'all'}-{note}-{state['midi_voice_counter']}"
        state["midi_voice_counter"] += 1
        _play_split_locked(voice_key, split_index, velocity, cue_data=cue_data, mp3_path=mp3_path, channel=channel, loop=True)
        note_voice_keys.append(voice_key)


def player_preview_on(split_index, velocity=110, voice_key="preview"):
    with state_lock:
        if not state["pyo_ready"]:
            raise RuntimeError("audio server is not ready")
        if state["current_file"] is None or state["cue_data"] is None:
            raise RuntimeError("no file has been analyzed")
        splits = state["cue_data"]["splits"]
        if split_index < 0 or split_index >= len(splits):
            raise RuntimeError(f"split index {split_index} out of range for {len(splits)} splits")

        _play_split_locked(voice_key, split_index, velocity)


def _play_split_locked(voice_key, split_index, velocity, cue_data=None, mp3_path=None, channel=None, loop=False):
    """Must be called with state_lock held."""
    if cue_data is None:
        cue_data = state["cue_data"]
    if mp3_path is None:
        mp3_path = state["current_file"]

    _stop_voice(voice_key)
    while len(state["active_voices"]) >= MAX_ACTIVE_VOICES and state["voice_order"]:
        _stop_voice(state["voice_order"][0])

    voice = {
        "cue_data": cue_data,
        "mp3_path": mp3_path,
        "split_index": split_index,
        "velocity": velocity,
        "channel": channel,
        "loop": loop,
    }
    state["active_voices"][voice_key] = voice
    state["voice_order"].append(voice_key)
    _start_voice_segment_locked(voice_key, voice)


def _start_voice_segment_locked(voice_key, voice):
    """Must be called with state_lock held."""
    cue_data = voice["cue_data"]
    mp3_path = voice["mp3_path"]
    split_index = voice["split_index"]
    velocity = voice["velocity"]
    channel = voice["channel"]
    splits = cue_data["splits"]
    duration = cue_data["duration"]

    start_sec = splits[split_index]
    end_sec = splits[split_index + 1] if split_index + 1 < len(splits) else duration
    if state["peak_start_enabled"]:
        peak_starts = cue_data.get("peak_starts") or []
        if split_index < len(peak_starts):
            start_sec = min(max(start_sec, peak_starts[split_index]), end_sec)
    volume = velocity / 127.0
    pitch_ratio = semitones_to_ratio(state["pitch_bend_semitones"].get(channel, 0.0))
    seg_duration = (end_sec - start_sec) / pitch_ratio
    fade_time = min(SAMPLE_EDGE_FADE_SECONDS, max(0.001, seg_duration * 0.45))
    fade = pyo.Fader(fadein=fade_time, fadeout=fade_time, dur=seg_duration, mul=volume).play()

    player = pyo.SfPlayer(
        str(mp3_path),
        speed=pitch_ratio,
        loop=False,
        offset=start_sec,
        interp=2,
        mul=fade,
    ).out()

    stop_cb = pyo.CallAfter(lambda: _voice_segment_done_cb(voice_key), seg_duration)
    voice["player"] = player
    voice["stop_cb"] = stop_cb
    voice["fade"] = fade


def _stop_voice(note):
    """Must be called with state_lock held."""
    voice = state["active_voices"].pop(note, None)
    if voice:
        fade = voice.get("fade")
        player = voice.get("player")
        stop_cb = voice.get("stop_cb")
        if stop_cb is not None and hasattr(stop_cb, "stop"):
            stop_cb.stop()
        if fade is not None and hasattr(fade, "stop"):
            fade.stop()
        pyo.CallAfter(lambda: _stop_player(player), SAMPLE_EDGE_FADE_SECONDS * 1.5)
    if note in state["voice_order"]:
        state["voice_order"].remove(note)
    _remove_midi_voice_key_locked(note)


def _stop_channel_voices(channel):
    """Must be called with state_lock held."""
    for voice_key, voice in list(state["active_voices"].items()):
        if voice.get("channel") == channel:
            _stop_voice(voice_key)


def _remove_midi_voice_key_locked(voice_key):
    """Must be called with state_lock held."""
    for note, voice_keys in list(state["midi_note_voices"].items()):
        if voice_key in voice_keys:
            voice_keys.remove(voice_key)
        if not voice_keys:
            state["midi_note_voices"].pop(note, None)


def _voice_segment_done_cb(note):
    with state_lock:
        voice = state["active_voices"].get(note)
        if not voice:
            return
        _stop_player(voice.get("player"))
        voice["player"] = None
        voice["stop_cb"] = None
        voice["fade"] = None
        if voice.get("loop"):
            _start_voice_segment_locked(note, voice)
            return
        state["active_voices"].pop(note, None)
        if note in state["voice_order"]:
            state["voice_order"].remove(note)
        _remove_midi_voice_key_locked(note)


def _stop_player(player):
    if player is not None and hasattr(player, "stop"):
        player.stop()


def player_note_off(note, channel=None):
    with state_lock:
        voice_keys = state["midi_note_voices"].get((channel, note))
        if not voice_keys and channel is not None:
            voice_keys = state["midi_note_voices"].get((None, note))
        if not voice_keys:
            return
        _stop_voice(voice_keys[0])


def player_preview_off(voice_key="preview"):
    with state_lock:
        _stop_voice(voice_key)


def player_pitch_bend(bend_value, channel=None):
    with state_lock:
        semitones = (bend_value / PITCH_BEND_MAX) * PITCH_BEND_RANGE
        state["pitch_bend_semitones"][channel] = semitones
        if channel is None:
            state["pitch_bend_semitones_preview"] = semitones
        ratio = semitones_to_ratio(semitones)
        for voice in state["active_voices"].values():
            if voice.get("channel") == channel:
                player = voice.get("player")
                if player is not None:
                    player.setSpeed(ratio)


def player_stop_all():
    with state_lock:
        for note in list(state["active_voices"].keys()):
            _stop_voice(note)
        state["voice_order"].clear()
        state["midi_note_voices"].clear()


# ---------------------------------------------------------------------------
# MIDI listener
# ---------------------------------------------------------------------------

def midi_listener(port_name, stop_event, port):
    if mido is None:
        print("MIDI disabled: mido is not installed.", file=sys.stderr)
        return

    print(f"MIDI: listening on port '{port_name}'")
    try:
        while not stop_event.is_set():
            for msg in port.iter_pending():
                handle_midi_message(msg)
            time.sleep(0.001)
    except Exception as e:
        print(f"MIDI error: {e}", file=sys.stderr)
    finally:
        port.close()


def handle_midi_message(msg):
    with state_lock:
        state["midi_activity_count"] += 1
        state["midi_activity_time"] = time.time()

    if msg.type == "note_on" and msg.velocity > 0:
        player_note_on(msg.note, msg.velocity, getattr(msg, "channel", None))
    elif msg.type == "note_off" or (msg.type == "note_on" and msg.velocity == 0):
        player_note_off(msg.note, getattr(msg, "channel", None))
    elif msg.type == "pitchwheel":
        player_pitch_bend(msg.pitch, getattr(msg, "channel", None))


def start_midi(port_name):
    if mido is None:
        raise RuntimeError("mido is not installed")

    with state_lock:
        old_thread = state["midi_thread"] if state["midi_thread"] and state["midi_thread"].is_alive() else None
        old_stop = state["midi_stop"]
    if old_thread:
        old_stop.set()
        old_thread.join(timeout=2)

    player_stop_all()
    with state_lock:
        old_server = state["pyo_server"]
        old_pyo_ready = state["pyo_ready"]
        state["pyo_server"] = None
        state["pyo_ready"] = False

    if old_server:
        old_server.stop()
        old_server.shutdown()

    try:
        resolved_port_name = resolve_midi_input_name(port_name)
        midi_port = mido.open_input(resolved_port_name)
    except Exception as e:
        with state_lock:
            state["midi_error"] = str(e)
        if old_pyo_ready:
            init_pyo()
        raise

    with state_lock:
        state["midi_stop"] = threading.Event()
        state["midi_port_name"] = resolved_port_name
        state["midi_error"] = None
        t = threading.Thread(target=midi_listener,
                             args=(resolved_port_name, state["midi_stop"], midi_port),
                             daemon=True)
        state["midi_thread"] = t
    t.start()
    if old_pyo_ready:
        init_pyo()


# ---------------------------------------------------------------------------
# HTTP API
# ---------------------------------------------------------------------------

def json_response(handler, data, status=200):
    body = json.dumps(data).encode()
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", len(body))
    handler.send_header("Access-Control-Allow-Origin", "*")
    handler.end_headers()
    handler.wfile.write(body)


def serve_file(handler, path, content_type):
    try:
        with open(path, "rb") as f:
            data = f.read()
        handler.send_response(200)
        handler.send_header("Content-Type", content_type)
        handler.send_header("Content-Length", len(data))
        handler.end_headers()
        handler.wfile.write(data)
    except FileNotFoundError:
        handler.send_response(404)
        handler.end_headers()


def resolve_mp3_file(filename):
    if not filename:
        return None
    mp3_dir = Path(state["mp3_dir"]).resolve()
    mp3_path = (mp3_dir / filename).resolve()
    if mp3_path.parent != mp3_dir or mp3_path.suffix.lower() != ".mp3":
        return None
    return mp3_path if mp3_path.exists() else None


def load_and_analyze_file(mp3_path, mode="words", interval=1.0,
                          silence_thresh=0.01, silence_min=0.5,
                          words_per_split=DEFAULT_WORDS_PER_SPLIT):
    player_stop_all()
    cue_data, waveform = analyze_file(
        mp3_path, mode=mode, interval=interval,
        silence_thresh=silence_thresh, silence_min=silence_min,
        words_per_split=words_per_split
    )
    with state_lock:
        state["current_file"] = mp3_path
        state["cue_data"] = cue_data
        state["waveform"] = waveform
    return cue_data, waveform


def choose_random_prefixed_mp3(mp3_dir, prefix):
    matches = [
        p for p in mp3_dir.iterdir()
        if p.suffix.lower() == ".mp3" and p.name.lower().startswith(prefix.lower())
    ]
    if not matches:
        return None
    return random.choice(sorted(matches))


def load_sigil_mp3s_with_defaults():
    mp3_dir = Path(state["mp3_dir"])
    loaded = {}
    for sigil in ["chaos", "oracle", "sacred", "directive"]:
        mp3_path = choose_random_prefixed_mp3(mp3_dir, sigil)
        if mp3_path is None:
            loaded[sigil] = {"sigil": sigil, "error": f"No MP3 files start with '{sigil}'"}
            print(f"No MP3 files start with '{sigil}' in {mp3_dir}", file=sys.stderr)
            continue
        try:
            cue_data, waveform = analyze_file(
                mp3_path.resolve(),
                mode="words",
                interval=1.0,
                words_per_split=DEFAULT_WORDS_PER_SPLIT,
            )
            loaded[sigil] = {
                "sigil": sigil,
                "current_file": mp3_path.resolve(),
                "cue_data": cue_data,
                "waveform": waveform,
                "error": None,
            }
            print(f"Loaded {sigil} MP3: {mp3_path.name}")
        except Exception as e:
            loaded[sigil] = {"sigil": sigil, "current_file": mp3_path.resolve(), "error": str(e)}
            print(f"Failed to load {sigil} MP3 '{mp3_path.name}': {e}", file=sys.stderr)

    with state_lock:
        state["sigil_samples"] = loaded
        first = next((sample for sample in loaded.values() if sample.get("cue_data")), None)
        if first:
            state["current_file"] = first["current_file"]
            state["cue_data"] = first["cue_data"]
            state["waveform"] = first["waveform"]


def load_first_mp3_with_defaults():
    mp3_dir = Path(state["mp3_dir"])
    files = sorted(p for p in mp3_dir.iterdir() if p.suffix.lower() == ".mp3")
    if not files:
        print(f"No MP3 files found in {mp3_dir}", file=sys.stderr)
        return
    mp3_path = files[0].resolve()
    try:
        load_and_analyze_file(mp3_path)
        print(f"Loaded MP3: {mp3_path.name}")
    except Exception as e:
        print(f"Failed to load MP3 '{mp3_path.name}': {e}", file=sys.stderr)


class Handler(BaseHTTPRequestHandler):

    def log_message(self, fmt, *args):
        pass  # suppress request logging

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path
        params = parse_qs(parsed.query)

        if path == "/" or path == "/index.html":
            serve_file(self, Path(__file__).parent / "static" / "index.html", "text/html")

        elif path == "/api/files":
            mp3_dir = state["mp3_dir"]
            files = sorted(
                p.name for p in Path(mp3_dir).iterdir()
                if p.suffix.lower() == ".mp3"
            )
            json_response(self, {"files": files, "dir": str(mp3_dir)})

        elif path == "/api/media":
            filename = params.get("file", [None])[0]
            mp3_path = resolve_mp3_file(filename)
            if mp3_path is None:
                self.send_response(404)
                self.end_headers()
                return
            serve_file(self, mp3_path, "audio/mpeg")

        elif path == "/api/midi_ports":
            ports, error = get_midi_input_names()
            with state_lock:
                state["midi_error"] = error
                current = state["midi_port_name"]
            json_response(self, {
                "ports": ports,
                "current": current,
                "error": error,
            })

        elif path == "/api/audio_outputs":
            devices, default_id, error = get_audio_output_devices()
            with state_lock:
                current_id = state["audio_output_id"]
                current_name = state["audio_output_name"]
                if current_id is None:
                    current_id = default_id
                    current_name = next(
                        (d["name"] for d in devices if d["id"] == current_id),
                        None
                    )
            json_response(self, {
                "devices": devices,
                "default": default_id,
                "current": current_id,
                "current_name": current_name,
                "error": error,
            })

        elif path == "/api/state":
            with state_lock:
                resp = {
                    "current_file": Path(state["current_file"]).name if state["current_file"] else None,
                    "cue_data": state["cue_data"],
                    "waveform": state["waveform"],
                    "sigil_samples": {
                        sigil: {
                            "current_file": Path(sample["current_file"]).name if sample.get("current_file") else None,
                            "cue_data": sample.get("cue_data"),
                            "error": sample.get("error"),
                        }
                        for sigil, sample in state["sigil_samples"].items()
                    },
                    "midi_port": state["midi_port_name"],
                    "midi_error": state["midi_error"],
                    "midi_activity_count": state["midi_activity_count"],
                    "midi_activity_time": state["midi_activity_time"],
                    "base_note": state["base_note"],
                    "peak_start_enabled": state["peak_start_enabled"],
                    "pitch_bend_semitones": state["pitch_bend_semitones_preview"],
                    "active_voices": list(state["active_voices"].keys()),
                    "pyo_ready": state["pyo_ready"],
                    "audio_error": state["audio_error"],
                    "audio_output_id": state["audio_output_id"],
                    "audio_output_name": state["audio_output_name"],
                }
            json_response(self, resp)

        elif path == "/api/analyze":
            filename = params.get("file", [None])[0]
            mode = params.get("mode", ["words"])[0]
            interval = float(params.get("interval", [1.0])[0])
            silence_thresh = float(params.get("silence_thresh", [0.01])[0])
            silence_min = float(params.get("silence_min", [0.5])[0])
            words_per_split = max(1, int(params.get("words_per_split", [DEFAULT_WORDS_PER_SPLIT])[0]))

            if not filename:
                json_response(self, {"error": "missing file"}, 400)
                return

            mp3_path = resolve_mp3_file(filename)
            if mp3_path is None:
                json_response(self, {"error": "file not found"}, 404)
                return

            try:
                cue_data, waveform = load_and_analyze_file(
                    mp3_path, mode=mode, interval=interval,
                    silence_thresh=silence_thresh, silence_min=silence_min,
                    words_per_split=words_per_split
                )
                json_response(self, {"cue_data": cue_data, "waveform": waveform})
            except Exception as e:
                json_response(self, {"error": str(e)}, 500)

        elif path == "/api/set_midi":
            port = params.get("port", [None])[0]
            if not port:
                json_response(self, {"error": "missing port"}, 400)
                return
            if mido is None:
                json_response(self, {"error": "mido is not installed"}, 503)
                return
            try:
                start_midi(port)
                with state_lock:
                    current = state["midi_port_name"]
                json_response(self, {"ok": True, "port": current})
            except Exception as e:
                json_response(self, {"error": str(e)}, 500)

        elif path == "/api/set_audio_output":
            output_id = params.get("id", [None])[0]
            if output_id is None:
                json_response(self, {"error": "missing id"}, 400)
                return
            try:
                name = set_audio_output(int(output_id))
                json_response(self, {"ok": True, "id": int(output_id), "name": name})
            except Exception as e:
                with state_lock:
                    state["audio_error"] = str(e)
                    state["pyo_ready"] = False
                json_response(self, {"error": str(e)}, 500)

        elif path == "/api/set_base_note":
            note = params.get("note", [None])[0]
            if note is None:
                json_response(self, {"error": "missing note"}, 400)
                return
            with state_lock:
                state["base_note"] = int(note)
            json_response(self, {"ok": True, "base_note": int(note)})

        elif path == "/api/set_peak_start":
            enabled = params.get("enabled", ["0"])[0] in ("1", "true", "yes", "on")
            with state_lock:
                state["peak_start_enabled"] = enabled
            json_response(self, {"ok": True, "peak_start_enabled": enabled})

        elif path == "/api/set_pitch_bend":
            try:
                semitones = float(params.get("semitones", [0.0])[0])
            except ValueError:
                json_response(self, {"error": "bad semitones"}, 400)
                return
            semitones = max(-PITCH_BEND_RANGE, min(PITCH_BEND_RANGE, semitones))
            with state_lock:
                state["pitch_bend_semitones"][None] = semitones
                state["pitch_bend_semitones_preview"] = semitones
            json_response(self, {"ok": True, "pitch_bend_semitones": semitones})

        elif path == "/api/stop_all":
            player_stop_all()
            json_response(self, {"ok": True})

        elif path == "/api/preview_on":
            filename = params.get("file", [None])[0]
            index = params.get("index", [None])[0]
            voice = params.get("voice", ["preview"])[0]
            if index is None:
                json_response(self, {"error": "missing index"}, 400)
                return
            try:
                split_index = int(index)
            except ValueError:
                json_response(self, {"error": f"bad index: {index}"}, 400)
                return

            with state_lock:
                current_file = Path(state["current_file"]).name if state["current_file"] else None
                split_count = len(state["cue_data"]["splits"]) if state["cue_data"] else 0
            if filename and current_file and filename != current_file:
                json_response(self, {
                    "error": f"selected file '{filename}' is not loaded on the server; loaded file is '{current_file}'",
                    "current_file": current_file,
                    "split_count": split_count,
                }, 409)
                return
            if split_index < 0 or split_index >= split_count:
                json_response(self, {
                    "error": f"split index {split_index} out of range for {split_count} splits",
                    "current_file": current_file,
                    "split_count": split_count,
                }, 400)
                return

            try:
                player_preview_on(split_index, voice_key=voice)
                json_response(self, {
                    "ok": True,
                    "index": split_index,
                    "voice": voice,
                    "current_file": current_file,
                    "split_count": split_count,
                })
            except Exception as e:
                json_response(self, {"error": str(e)}, 500)

        elif path == "/api/preview_off":
            voice = params.get("voice", ["preview"])[0]
            player_preview_off(voice)
            json_response(self, {"ok": True, "voice": voice})

        else:
            self.send_response(404)
            self.end_headers()


# ---------------------------------------------------------------------------
# Browser open (detect if already open via a flag file)
# ---------------------------------------------------------------------------

def open_browser(port):
    url = f"http://localhost:{port}"
    flag = Path(tempfile.gettempdir()) / f"samplesplitter_{port}.open"
    if not flag.exists():
        flag.touch()
        time.sleep(0.8)
        webbrowser.open(url)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def resolve_default_mp3_dir(raw_dir):
    raw_path = Path(raw_dir).expanduser()
    if raw_dir != "mp3s" or raw_path.is_dir():
        return raw_path.resolve()
    script_dir = Path(__file__).parent.resolve()
    candidates = [
        script_dir / "mp3s",
        script_dir.parent.parent / "data_default" / "samplesplitter" / "mp3s",
        Path.cwd() / "data_default" / "samplesplitter" / "mp3s",
        Path.cwd().parent / "data_default" / "samplesplitter" / "mp3s",
    ]
    for candidate in candidates:
        if candidate.is_dir():
            return candidate.resolve()
    return raw_path.resolve()


def main():
    parser = argparse.ArgumentParser(description="Sample splitter and MIDI player.")
    parser.add_argument("--dir", default="mp3s", help="Directory containing MP3 files (default: mp3s)")
    parser.add_argument("--port", type=int, default=9876, help="HTTP port (default: 9876)")
    parser.add_argument("--base-note", type=int, default=48, help="MIDI base note (default: 48 = C3)")
    parser.add_argument("--midi-port", nargs="+", default=None, help="MIDI input port to listen to on startup")
    parser.add_argument("--no-open", action="store_true", help="Do not open the browser automatically")
    args = parser.parse_args()

    mp3_dir = resolve_default_mp3_dir(args.dir)
    if not mp3_dir.is_dir():
        print(f"Error: directory not found: {mp3_dir}", file=sys.stderr)
        sys.exit(1)

    state["mp3_dir"] = mp3_dir
    state["base_note"] = args.base_note

    load_sigil_mp3s_with_defaults()
    if state["current_file"] is None:
        load_first_mp3_with_defaults()

    # Start pyo in background thread
    pyo_thread = threading.Thread(target=init_pyo, daemon=True)
    pyo_thread.start()

    if not args.no_open:
        # Open browser (non-blocking, detects if already open)
        browser_thread = threading.Thread(target=open_browser, args=(args.port,), daemon=True)
        browser_thread.start()

    print(f"Sample Splitter running at http://localhost:{args.port}")
    print(f"MP3 directory: {mp3_dir}")
    midi_port = " ".join(args.midi_port) if args.midi_port else None
    if midi_port:
        try:
            start_midi(midi_port)
            print(f"MIDI input: {midi_port}")
        except Exception as e:
            print(f"MIDI input failed: {e}", file=sys.stderr)
    print("Press Ctrl+C to quit.\n")

    server = HTTPServer(("localhost", args.port), Handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down...")
        player_stop_all()
        with state_lock:
            if state["midi_stop"]:
                state["midi_stop"].set()
            if state["pyo_server"]:
                state["pyo_server"].stop()


if __name__ == "__main__":
    main()
