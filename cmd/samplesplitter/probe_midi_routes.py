import argparse
import time

import mido


def main():
    parser = argparse.ArgumentParser(
        description="Probe which MIDI output reaches a selected MIDI input."
    )
    parser.add_argument("--input", default="01. Internal MIDI 0")
    parser.add_argument("--note", type=int, default=60)
    parser.add_argument("--velocity", type=int, default=100)
    parser.add_argument("--wait", type=float, default=0.2)
    args = parser.parse_args()

    outputs = mido.get_output_names()
    print(f"Listening on input: {args.input}")
    print("Testing outputs:")
    for output_name in outputs:
        print(f"  -> {output_name}")
        received = []
        try:
            with mido.open_input(args.input) as input_port:
                with mido.open_output(output_name) as output_port:
                    output_port.send(
                        mido.Message("note_on", note=args.note, velocity=args.velocity)
                    )
                    time.sleep(args.wait)
                    output_port.send(
                        mido.Message("note_off", note=args.note, velocity=0)
                    )
                    time.sleep(args.wait)
                    received.extend(input_port.iter_pending())
        except Exception as exc:
            print(f"     error: {exc}")
            continue

        if received:
            print("     received:")
            for msg in received:
                print(f"       {msg}")
        else:
            print("     no messages")


if __name__ == "__main__":
    main()
