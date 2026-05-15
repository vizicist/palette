#!/usr/bin/env python3
"""
Patch Omnisphere multi files into the Palette Bidule configuration.

The default mapping is:
  Omnisphere_0  <- fourgo_1.mlt_omn
  Omnisphere_1  <- fourgo_2.mlt_omn
  ...
  Omnisphere_11 <- fourgo_12.mlt_omn

The script backs up the Bidule file before writing. It expects Bidule to be
closed, otherwise Bidule may overwrite the patched file when it exits.
"""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import os
import pathlib
import re
import shutil
import sys
import zlib


BIDULE_BLOCK_RE = re.compile(
    r'<Bidule\b(?=[^>]*displayName="(Omnisphere_(\d+))")[\s\S]*?</Bidule>'
)
CUSTOM_DATA_RE = re.compile(r'<CustomData name="([^"]+)">([\s\S]*?)</CustomData>')
VST3_HEADER_SIZE = 24
VST3_TRAILER = (b"\0" * 20) + b"JUCEPrivateData"


def default_config_dir() -> pathlib.Path:
    local_app_data = os.environ.get("LOCALAPPDATA")
    if not local_app_data:
        raise RuntimeError("LOCALAPPDATA is not set; pass --bidule and --multi-dir explicitly.")
    return pathlib.Path(local_app_data) / "Palette" / "data_default" / "config"


def signed_byte_sum(data: bytes) -> int:
    return sum(byte if byte < 128 else byte - 256 for byte in data)


def decompress_custom_data(block: str, name: str) -> bytes | None:
    cdata = dict(CUSTOM_DATA_RE.findall(block))
    encoded = cdata.get(name)
    if not encoded:
        return None
    raw = base64.b64decode("".join(encoded.split()))
    return zlib.decompress(raw)


def read_chunk(block: str) -> bytes | None:
    chunk = decompress_custom_data(block, "VSTChunk")
    if chunk is not None:
        return chunk

    state = decompress_custom_data(block, "VST3ComponentState")
    if state is None:
        return None
    xml_offset = state.find(b"<SynthMaster")
    if xml_offset < 0:
        return state
    trailer_offset = state.find(b"JUCEPrivateData", xml_offset)
    if trailer_offset < 0:
        return state[xml_offset:]
    return state[xml_offset:trailer_offset].rstrip(b"\0")


def entry_name(chunk: bytes) -> str | None:
    match = re.search(br'<ENTRYDESCR\s+name="([^"]+)"', chunk)
    if not match:
        return None
    return match.group(1).decode("utf-8", "replace")


def replace_custom_data(block: str, name: str, value: str) -> str:
    pattern = re.compile(rf'(<CustomData name="{re.escape(name)}">)([\s\S]*?)(</CustomData>)')
    if not pattern.search(block):
        raise ValueError(f"Omnisphere block is missing CustomData {name!r}")
    return pattern.sub(rf"\g<1>{value}\g<3>", block, count=1)


def encode_compressed(data: bytes) -> tuple[str, int, int]:
    compressed = zlib.compress(data)
    encoded = base64.b64encode(compressed).decode("ascii")
    return encoded, len(data), signed_byte_sum(compressed)


def make_vst_chunk(multi_path: pathlib.Path) -> tuple[str, int, int]:
    chunk = multi_path.read_bytes()
    if not chunk.endswith(b"\0"):
        chunk += b"\0"
    return encode_compressed(chunk)


def make_vst3_component_state(block: str, multi_path: pathlib.Path) -> tuple[str, int, int]:
    current_state = decompress_custom_data(block, "VST3ComponentState")
    if current_state is None or len(current_state) < VST3_HEADER_SIZE:
        raise ValueError("Omnisphere block is missing a valid VST3ComponentState")

    chunk = multi_path.read_bytes()
    if not chunk.endswith(b"\0"):
        chunk += b"\0"

    header = bytearray(current_state[:VST3_HEADER_SIZE])
    header[16:20] = len(chunk).to_bytes(4, "little")
    state = bytes(header) + chunk + VST3_TRAILER
    return encode_compressed(state)


def patch_block(block: str, multi_path: pathlib.Path) -> str:
    cdata = dict(CUSTOM_DATA_RE.findall(block))
    if "VSTChunk" in cdata:
        encoded, size, checksum = make_vst_chunk(multi_path)
        block = replace_custom_data(block, "VSTChunk", encoded)
        block = replace_custom_data(block, "VSTChunkCompressed", "yes")
        block = replace_custom_data(block, "VSTChunkSize", str(size))
        block = replace_custom_data(block, "VSTChunkSum", str(checksum))
        return block

    if "VST3ComponentState" in cdata:
        encoded, size, checksum = make_vst3_component_state(block, multi_path)
        block = replace_custom_data(block, "VST3ComponentState", encoded)
        block = replace_custom_data(block, "VST3ComponentStateCheckSum", str(checksum))
        block = replace_custom_data(block, "VST3ComponentStateSize", str(size))
        return block

    raise ValueError("Omnisphere block is missing VSTChunk or VST3ComponentState")


def is_process_running(name: str) -> bool:
    try:
        import subprocess

        result = subprocess.run(
            ["tasklist", "/FI", f"IMAGENAME eq {name}"],
            check=False,
            capture_output=True,
            text=True,
        )
    except Exception:
        return False
    return name.lower() in result.stdout.lower()


def parse_args() -> argparse.Namespace:
    config_dir = default_config_dir()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--bidule",
        default=str(config_dir / "default.bidule"),
        help="Bidule file to patch.",
    )
    parser.add_argument(
        "--multi-dir",
        default=str(config_dir / "omnisphere" / "Multis" / "User" / "SPPro"),
        help="Directory containing fourgo_1.mlt_omn through fourgo_12.mlt_omn.",
    )
    parser.add_argument("--dry-run", action="store_true", help="Report changes without writing.")
    parser.add_argument(
        "--force",
        action="store_true",
        help="Rewrite blocks even when they already appear to contain the expected multi.",
    )
    parser.add_argument(
        "--allow-bidule-running",
        action="store_true",
        help="Patch even if Bidule appears to be running.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    bidule_path = pathlib.Path(args.bidule)
    multi_dir = pathlib.Path(args.multi_dir)

    if not bidule_path.exists():
        print(f"Bidule file does not exist: {bidule_path}", file=sys.stderr)
        return 2
    if not multi_dir.exists():
        print(f"Multi directory does not exist: {multi_dir}", file=sys.stderr)
        return 2
    if is_process_running("bidule.exe") and not args.allow_bidule_running:
        print("Bidule appears to be running. Close it first, or pass --allow-bidule-running.", file=sys.stderr)
        return 2

    missing = [multi_dir / f"fourgo_{idx}.mlt_omn" for idx in range(1, 13)]
    missing = [path for path in missing if not path.exists()]
    if missing:
        for path in missing:
            print(f"Missing multi: {path}", file=sys.stderr)
        return 2

    text = bidule_path.read_text(errors="replace")
    matches = list(BIDULE_BLOCK_RE.finditer(text))
    if len(matches) != 12:
        print(f"Expected 12 Omnisphere blocks, found {len(matches)}.", file=sys.stderr)
        return 2

    replacements: dict[tuple[int, int], str] = {}
    changed = 0
    for match in matches:
        display_name = match.group(1)
        omni_index = int(match.group(2))
        multi_index = omni_index + 1
        if not 1 <= multi_index <= 12:
            print(f"Unexpected Omnisphere index in {display_name}; expected 0 through 11.", file=sys.stderr)
            return 2

        block = match.group(0)
        current = read_chunk(block)
        current_name = entry_name(current) if current is not None else None
        expected_name = f"fourgo_{multi_index}"
        multi_path = multi_dir / f"{expected_name}.mlt_omn"

        if current_name == expected_name and not args.force:
            print(f"skip  {display_name}: already contains {expected_name}")
            continue

        new_block = patch_block(block, multi_path)
        replacements[(match.start(), match.end())] = new_block
        changed += 1
        print(f"patch {display_name}: {current_name or 'unknown'} -> {expected_name}")

    if changed == 0:
        print("No changes needed.")
        return 0

    patched = []
    cursor = 0
    for (start, end), new_block in sorted(replacements.items()):
        patched.append(text[cursor:start])
        patched.append(new_block)
        cursor = end
    patched.append(text[cursor:])
    patched_text = "".join(patched)

    if args.dry_run:
        print(f"Dry run: would patch {changed} Omnisphere block(s).")
        return 0

    stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    backup_path = bidule_path.with_name(f"{bidule_path.name}.before-omni-patch-{stamp}")
    shutil.copy2(bidule_path, backup_path)
    bidule_path.write_text(patched_text, newline="", encoding="utf-8")
    print(f"Patched {changed} Omnisphere block(s).")
    print(f"Backup: {backup_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
