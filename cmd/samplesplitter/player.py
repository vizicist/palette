"""
player.py — Polyphonic sample player with MIDI input and pitch bend.

Usage:
    python3 player.py <file.mp3> [options]

Options:
    --cues FILE          Cue file (default: <file>.cues.json)
    --midi-port NAME     MIDI input port name (or partial match)
    --list-ports         List available MIDI input ports and exit
    --base-note INT      MIDI note number for split 0 (default: 48 = C3)
    --base-channel INT   MIDI channel to listen on, 1-16 or 0=all (default: 0)
"""

import argparse
import json
import sys
import time
from pathlib import Path

import mido
import pyo


# Pitch bend range in semitones (standard: ±2)
PITCH_BEND_RANGE = 2.0
PITCH_BEND_MAX = 8192


def list_midi_ports():
    ports = mido.get_input_names()
    if not ports:
        print("No MIDI input ports found.")
    else:
        print("Available MIDI input ports:")
        for i, name in enumerate(ports):
            print(f"  [{i}] {name}")
    return ports


def find_midi_port(name_fragment):
    ports = mido.get_input_names()
    if not ports:
        print("Error: no MIDI input ports available.", file=sys.stderr)
        sys.exit(1)
    if name_fragment is None:
        return ports[0]
    matches = [p for p in ports if name_fragment.lower() in p.lower()]
    if not matches:
        print(f"Error: no MIDI port matching '{name_fragment}'.", file=sys.stderr)
        print("Available ports:", file=sys.stderr)
        for p in ports:
            print(f"  {p}", file=sys.stderr)
        sys.exit(1)
    return matches[0]


class SamplePlayer:
    def __init__(self, mp3_path, cue_data, base_note=48, midi_channel=0):
        self.mp3_path = str(mp3_path)
        self.splits = cue_data["splits"]
        self.duration = cue_data["duration"]
        self.base_note = base_note
        self.midi_channel = midi_channel  # 0 = all channels
        self.pitch_bend_semitones = 0.0   # current pitch bend in semitones
        self.active_voices = {}           # note -> (SfPlayer, TrigFunc)

        # Boot pyo server
        self.server = pyo.Server().boot()
        self.server.start()

        # Load the MP3 as a SfPlayer sound table
        print(f"Loading audio: {self.mp3_path}")
        self.table = pyo.SndTable(self.mp3_path)
        print(f"Loaded: {self.duration:.2f}s, {len(self.splits)} splits")

    def _semitones_to_ratio(self, semitones):
        return 2.0 ** (semitones / 12.0)

    def note_on(self, note, velocity):
        split_index = note - self.base_note
        if split_index < 0 or split_index >= len(self.splits):
            return  # out of range

        # Stop existing voice on same note if any
        self.note_off(note)

        start_sec = self.splits[split_index]
        # End is next split point or end of file
        if split_index + 1 < len(self.splits):
            end_sec = self.splits[split_index + 1]
        else:
            end_sec = self.duration

        volume = velocity / 127.0
        pitch_ratio = self._semitones_to_ratio(self.pitch_bend_semitones)

        # pyo SfPlayer: speed=1.0 is normal pitch, loop=False, offset in seconds
        player = pyo.SfPlayer(
            self.mp3_path,
            speed=pitch_ratio,
            loop=False,
            offset=start_sec,
            interp=2,
            mul=volume,
        ).out()

        # Schedule note-off after segment duration (safety fallback)
        seg_duration = (end_sec - start_sec) / pitch_ratio
        stop_trig = pyo.CallAfter(player.stop, seg_duration)

        self.active_voices[note] = (player, stop_trig)

    def note_off(self, note):
        if note in self.active_voices:
            player, stop_trig = self.active_voices.pop(note)
            player.stop()

    def set_pitch_bend(self, bend_value):
        """bend_value: -8192 to +8191 (MIDI pitch bend)"""
        self.pitch_bend_semitones = (bend_value / PITCH_BEND_MAX) * PITCH_BEND_RANGE
        # Update pitch on all active voices
        ratio = self._semitones_to_ratio(self.pitch_bend_semitones)
        for note, (player, _) in self.active_voices.items():
            player.setSpeed(ratio)

    def handle_message(self, msg):
        # Filter by channel if specified
        if self.midi_channel != 0 and hasattr(msg, "channel"):
            if msg.channel != (self.midi_channel - 1):
                return

        if msg.type == "note_on" and msg.velocity > 0:
            self.note_on(msg.note, msg.velocity)
        elif msg.type == "note_off" or (msg.type == "note_on" and msg.velocity == 0):
            self.note_off(msg.note)
        elif msg.type == "pitchwheel":
            self.set_pitch_bend(msg.pitch)

    def run(self, midi_port_name):
        print(f"Opening MIDI port: {midi_port_name}")
        print(f"Base note: {self.base_note} (MIDI note {self.base_note} = split 0)")
        print(f"Pitch bend range: ±{PITCH_BEND_RANGE} semitones")
        print("Ready. Press Ctrl+C to quit.\n")

        with mido.open_input(midi_port_name) as port:
            try:
                for msg in port:
                    self.handle_message(msg)
            except KeyboardInterrupt:
                print("\nStopping...")

        self.server.stop()

    def stop(self):
        for note in list(self.active_voices.keys()):
            self.note_off(note)
        self.server.stop()


def main():
    parser = argparse.ArgumentParser(description="Polyphonic sample player with MIDI and pitch bend.")
    parser.add_argument("file", nargs="?", help="Input MP3 file")
    parser.add_argument("--cues", help="Cue file (default: <file>.cues.json)")
    parser.add_argument("--midi-port", help="MIDI input port name (partial match ok)")
    parser.add_argument("--list-ports", action="store_true", help="List MIDI ports and exit")
    parser.add_argument("--base-note", type=int, default=48, help="MIDI note for split 0 (default: 48 = C3)")
    parser.add_argument("--base-channel", type=int, default=0,
                        help="MIDI channel 1-16, or 0 for all (default: 0)")
    args = parser.parse_args()

    if args.list_ports:
        list_midi_ports()
        sys.exit(0)

    if not args.file:
        parser.print_help()
        sys.exit(1)

    mp3_path = Path(args.file)
    if not mp3_path.exists():
        print(f"Error: file not found: {mp3_path}", file=sys.stderr)
        sys.exit(1)

    cue_path = Path(args.cues) if args.cues else mp3_path.with_suffix(".cues.json")
    if not cue_path.exists():
        print(f"Error: cue file not found: {cue_path}", file=sys.stderr)
        print("Run splitter.py first to generate cue points.", file=sys.stderr)
        sys.exit(1)

    with open(cue_path) as f:
        cue_data = json.load(f)

    midi_port = find_midi_port(args.midi_port)

    player = SamplePlayer(mp3_path, cue_data, base_note=args.base_note, midi_channel=args.base_channel)
    player.run(midi_port)


if __name__ == "__main__":
    main()
