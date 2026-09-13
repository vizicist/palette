import json
import tempfile
import unittest
from pathlib import Path

import palettes_analyze


def write_events(path: Path, events):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("".join(json.dumps(event) + "\n" for event in events), encoding="utf-8")


class LastSeenTests(unittest.TestCase):
    def test_collect_last_seen_uses_newest_valid_event_timestamp(self):
        with tempfile.TemporaryDirectory() as directory:
            palettes_dir = Path(directory)
            write_events(
                palettes_dir / "PaletteOne" / "2026-07-27.json",
                [
                    {"msg": "Quad.Load", "time": "2026-07-27T15:00:00Z"},
                    {"msg": "setAttractMode", "time": "2026-07-27T18:30:00Z"},
                ],
            )
            write_events(
                palettes_dir / "PaletteOne" / "2026-07-28.json",
                [
                    {"msg": "Quad.Load", "time": "not-a-timestamp"},
                    {"msg": "Quad.Load"},
                ],
            )

            result = palettes_analyze.collect_last_seen(
                palettes_dir, ["PaletteOne", "NeverSeen"]
            )

            self.assertEqual(
                result,
                {
                    "PaletteOne": "2026-07-27T18:30:00+00:00",
                    "NeverSeen": None,
                },
            )
    def test_generated_palette_list_includes_last_seen_data_and_label(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "report.html"
            palettes_analyze.generate_html(
                {"2026-07-27": {"PaletteOne": 1}},
                {"2026-07-27": {"PaletteOne": {12: 1}}},
                {"2026-07-27": {"PaletteOne": 60}},
                [],
                {"PaletteOne": "#123456"},
                {"2026-07-27": {"PaletteOne": 0}},
                {"PaletteOne": "2026-07-27T18:30:00+00:00"},
                output,
            )

            html = output.read_text(encoding="utf-8")
            self.assertIn('"lastSeen": {', html)
            self.assertIn('"PaletteOne": "2026-07-27T18:30:00+00:00"', html)
            self.assertIn("Last seen ${daysAgo} ${unit} ago", html)
            self.assertIn("grid-template-columns: auto 12px 1fr max-content", html)
            self.assertIn("font-variant-numeric: tabular-nums", html)


if __name__ == "__main__":
    unittest.main()
