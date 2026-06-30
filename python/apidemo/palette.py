from pathlib import Path
import importlib.util

_palette_path = Path(__file__).resolve().parents[1] / "palette.py"
_spec = importlib.util.spec_from_file_location("_palette_api_helper", _palette_path)
if _spec is None or _spec.loader is None:
    raise ImportError(f"Unable to load palette helper from {_palette_path}")

_module = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_module)

for _name in dir(_module):
    if not _name.startswith("__"):
        globals()[_name] = getattr(_module, _name)
