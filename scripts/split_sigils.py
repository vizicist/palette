"""Split data_default/shapes/sigils.png into four cropped PNG tiles and
trace each one to an SVG with potrace.

The source PNG is a 2x2 grid of circular sigils over captions (SACRED,
CHAOS, ORACLE, DIRECTIVE). For each quadrant we:

  1. Isolate the upper (sigil) region, excluding the caption text below.
  2. Threshold to pure black/white and compute a tight bounding box
     around the light-colored sigil pixels.
  3. Save the cropped PNG alongside sigils.png.
  4. Emit a PBM of the same bitmap and invoke potrace to produce an SVG
     with cubic Bezier path data.

Run:
    python scripts/split_sigils.py
Requires: Pillow, potrace on PATH.
"""
from __future__ import annotations

import subprocess
from pathlib import Path

from PIL import Image, ImageOps

REPO = Path(__file__).resolve().parent.parent
SRC = REPO / "data_default" / "shapes" / "sigils.png"
OUT_DIR = SRC.parent

NAMES = ["sacred", "chaos", "oracle", "directive"]
THRESHOLD = 96
CAPTION_EXCLUDE_FRACTION = 0.62
PAD = 8


def quadrant_boxes(width: int, height: int) -> list[tuple[int, int, int, int]]:
    hw, hh = width // 2, height // 2
    return [
        (0, 0, hw, hh),
        (hw, 0, width, hh),
        (0, hh, hw, height),
        (hw, hh, width, height),
    ]


def tight_bbox(gray: Image.Image, limit_y: int) -> tuple[int, int, int, int]:
    mask = gray.point(lambda p: 255 if p > THRESHOLD else 0, mode="L")
    upper = mask.crop((0, 0, mask.width, limit_y))
    bbox = upper.getbbox()
    if bbox is None:
        raise RuntimeError("no sigil pixels found in quadrant")
    x0, y0, x1, y1 = bbox
    x0 = max(0, x0 - PAD)
    y0 = max(0, y0 - PAD)
    x1 = min(mask.width, x1 + PAD)
    y1 = min(limit_y, y1 + PAD)
    return (x0, y0, x1, y1)


def process_one(full: Image.Image, box: tuple[int, int, int, int], name: str) -> None:
    quad = full.crop(box).convert("L")
    limit_y = int(quad.height * CAPTION_EXCLUDE_FRACTION)
    bbox = tight_bbox(quad, limit_y)
    sigil = quad.crop(bbox)

    png_path = OUT_DIR / f"{name}.png"
    sigil.save(png_path)
    print(f"wrote {png_path}  ({sigil.size[0]}x{sigil.size[1]})")

    # Binarize for potrace. Sigils are light on dark, but potrace traces
    # the BLACK regions, so we invert.
    binary = sigil.point(lambda p: 0 if p > THRESHOLD else 255, mode="L")
    pbm_path = OUT_DIR / f"{name}.pbm"
    binary.convert("1").save(pbm_path)

    svg_path = OUT_DIR / f"{name}.svg"
    subprocess.run(
        ["potrace", "-s", "-o", str(svg_path), str(pbm_path)],
        check=True,
    )
    pbm_path.unlink()
    print(f"wrote {svg_path}")


def main() -> None:
    if not SRC.exists():
        raise SystemExit(f"missing source: {SRC}")
    full = Image.open(SRC).convert("RGB")
    for name, box in zip(NAMES, quadrant_boxes(*full.size)):
        process_one(full, box, name)


if __name__ == "__main__":
    main()
