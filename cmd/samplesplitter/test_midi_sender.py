import argparse
import re
import sys
import time

import mido


def paired_internal_output_name(input_name, output_names):
    match = re.search(r"Internal MIDI (\d+)$", input_name)
    if not match:
        return None

    paired_number = int(match.group(1)) + 1
    suffix = f"Internal MIDI {paired_number}"
    for name in output_names:
        if name.endswith(suffix):
            return name
    return None


def resolve_output_port(requested, output_names):
    if requested in output_names:
        return requested

    paired = paired_internal_output_name(requested, output_names)
    if paired:
        return paired

    requested_lower = requested.lower()
    matches = [name for name in output_names if requested_lower in name.lower()]
    if len(matches) == 1:
        return matches[0]

    return None


def print_ports():
    print("MIDI inputs:")
    for name in mido.get_input_names():
        print(f"  {name}")

    print("\nMIDI outputs:")
    for name in mido.get_output_names():
        print(f"  {name}")


def send_test_notes(port_name, base_note, count, velocity, hold_seconds, gap_seconds):
    output_names = mido.get_output_names()
    resolved = resolve_output_port(port_name, output_names)
    if not resolved:
        print(f"Could not find a MIDI output matching: {port_name!r}", file=sys.stderr)
        print_ports()
        return 1

    if resolved != port_name:
        print(f"Requested input-like port: {port_name}")
        print(f"Sending to paired output: {resolved}")
    else:
        print(f"Sending to output: {resolved}")

    with mido.open_output(resolved) as port:
        for i in range(count):
            note = base_note + i
            print(f"note_on  note={note} velocity={velocity}")
            port.send(mido.Message("note_on", note=note, velocity=velocity))
            time.sleep(hold_seconds)
            print(f"note_off note={note}")
            port.send(mido.Message("note_off", note=note, velocity=0))
            time.sleep(gap_seconds)

    print("Done.")
    return 0


def main():
    parser = argparse.ArgumentParser(
        description="Send test MIDI notes to a MIDI output port."
    )
    parser.add_argument(
        "--port",
        default="01. Internal MIDI 0",
        help="Output port name, or the app's input port name to resolve to its paired output.",
    )
    parser.add_argument("--base-note", type=int, default=48)
    parser.add_argument("--count", type=int, default=8)
    parser.add_argument("--velocity", type=int, default=110)
    parser.add_argument("--hold", type=float, default=0.25)
    parser.add_argument("--gap", type=float, default=0.08)
    parser.add_argument("--list", action="store_true", help="List MIDI ports and exit.")
    args = parser.parse_args()

    if args.list:
        print_ports()
        return 0

    return send_test_notes(
        args.port,
        args.base_note,
        args.count,
        args.velocity,
        args.hold,
        args.gap,
    )


if __name__ == "__main__":
    raise SystemExit(main())
