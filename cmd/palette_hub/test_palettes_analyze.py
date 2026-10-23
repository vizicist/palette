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
                {},
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


class PopularPresetTests(unittest.TestCase):
    def test_analyze_day_counts_named_presets_loaded_outside_attract_mode(self):
        with tempfile.TemporaryDirectory() as directory:
            day_file = Path(directory) / "2026-10-13.json"
            write_events(
                day_file,
                [
                    {"msg": "Quad.Load", "filename": "Attract Preset", "time": "2026-10-13T10:00:00Z"},
                    {"msg": "setAttractMode", "onoff": False, "time": "2026-10-13T10:01:00Z"},
                    {"msg": "Quad.Load", "filename": "Nebula", "time": "2026-10-13T10:02:00Z"},
                    {"msg": "Quad.Load", "filename": "Nebula", "time": "2026-10-13T10:03:00Z"},
                    {"msg": "Quad.Load", "filename": "Sunrise", "time": "2026-10-13T10:04:00Z"},
                    {"msg": "Quad.Load", "filename": "_Current", "time": "2026-10-13T10:05:00Z"},
                    {"msg": "Quad.Load", "filename": "", "time": "2026-10-13T10:06:00Z"},
                ],
            )

            result = palettes_analyze.analyze_day_file(day_file, "PaletteOne", "2026-10-13")

            self.assertEqual(dict(result[5]), {"Nebula": 2, "Sunrise": 1})

    def test_analyze_all_groups_preset_counts_by_date_and_location(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            palettes_dir = root / "palettes"
            (root / "palettes.json").write_text(
                json.dumps({"location": "PaletteOne", "color": "#123456"}) + "\n",
                encoding="utf-8",
            )
            write_events(
                palettes_dir / "PaletteOne" / "2026-10-13.json",
                [
                    {"msg": "setAttractMode", "onoff": False, "time": "2026-10-13T10:00:00Z"},
                    {"msg": "Quad.Load", "filename": "Nebula", "time": "2026-10-13T10:01:00Z"},
                    {"msg": "Quad.Load", "filename": "Nebula", "time": "2026-10-13T10:02:00Z"},
                ],
            )

            result = palettes_analyze.analyze_all_palettes(palettes_dir, root)

            self.assertEqual(
                result[4],
                {"2026-10-13": {"PaletteOne": {"Nebula": 2}}},
            )

    def test_generated_report_contains_filterable_popular_presets_chart(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "report.html"
            palettes_analyze.generate_html(
                {"2026-10-13": {"PaletteOne": 3}},
                {"2026-10-13": {"PaletteOne": {10: 3}}},
                {"2026-10-13": {"PaletteOne": 120}},
                [],
                {"2026-10-13": {"PaletteOne": {"Nebula": 2, "Sunrise": 1}}},
                {"PaletteOne": "#123456"},
                {"2026-10-13": {"PaletteOne": 0}},
                {"PaletteOne": "2026-10-13T10:06:00+00:00"},
                output,
            )

            html = output.read_text(encoding="utf-8")
            self.assertIn("Most Popular Presets", html)
            self.assertIn('"presetLoadsByDate": {', html)
            self.assertIn('"Nebula": 2', html)
            self.assertIn("function updatePopularPresets()", html)
            self.assertIn("paletteVisibility[palette]", html)


if __name__ == "__main__":
    unittest.main()
