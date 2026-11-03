"""Package the Windows Spout receiver with Avenue's default GPU preference."""

import hashlib
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
from urllib.request import urlretrieve


PYINSTALLER_SOURCE = (
    "https://files.pythonhosted.org/packages/c8/63/"
    "fd62472b6371d89dc138d40c36d87a50dc2de18a035803bbdc376b4ffac4/"
    "pyinstaller-6.19.0.tar.gz"
)
PYINSTALLER_SHA256 = "ec73aeb8bd9b7f2f1240d328a4542e90b3c6e6fbc106014778431c616592a865"


def build(destination):
    receiver = Path(__file__).resolve().with_name("spout_receiver.py")
    with tempfile.TemporaryDirectory(prefix="palette-spout-") as directory:
        work = Path(directory)
        archive = work / "pyinstaller.tar.gz"
        urlretrieve(PYINSTALLER_SOURCE, archive)
        with archive.open("rb") as stream:
            if hashlib.file_digest(stream, "sha256").hexdigest() != PYINSTALLER_SHA256:
                raise RuntimeError("PyInstaller source archive has an unexpected checksum")
        with tarfile.open(archive) as source:
            source.extractall(work, filter="data")

        project = work / "pyinstaller-6.19.0"
        bootloader = project / "bootloader"
        entry = bootloader / "src" / "main.c"
        original = entry.read_text(encoding="utf-8")
        anchor = "#if defined(_WIN32)\n"
        if original.count(anchor) != 1:
            raise RuntimeError("Could not locate the Windows bootloader entry point")
        # The driver reads these exports on the executable that owns the OpenGL
        # context, which in a one-file build is the PyInstaller bootloader itself.
        entry.write_text(
            original.replace(
                anchor,
                anchor
                + "\n__declspec(dllexport) DWORD NvOptimusEnablement = 1;\n"
                + "__declspec(dllexport) int AmdPowerXpressRequestHighPerformance = 1;\n",
            ),
            encoding="utf-8",
        )
        subprocess.run([sys.executable, "waf", "all"], cwd=bootloader, check=True)
        subprocess.run(
            [
                sys.executable, "-m", "PyInstaller", "--noconfirm", "--onefile",
                "--collect-binaries", "glfw", "--name", "palette_spout_receiver",
                "--distpath", str(destination), "--workpath", str(work / "build"),
                "--specpath", str(work), str(receiver),
            ],
            cwd=project,
            check=True,
        )


if __name__ == "__main__":
    build(Path(sys.argv[1]).resolve())
