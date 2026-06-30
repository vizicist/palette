#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTE_SOURCE="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERSION="$(tr -d '\r\n' < "$PALETTE_SOURCE/VERSION")"
ARCH="$(uname -m)"
SHIP="$SCRIPT_DIR/ship"
BIN="$SHIP/bin"
RELEASE_DIR="$PALETTE_SOURCE/release"
GO_BIN="${GO_BIN:-}"

if [ -z "$GO_BIN" ]; then
    if [ -x /opt/homebrew/bin/go ]; then
        GO_BIN="/opt/homebrew/bin/go"
    elif [ -x /usr/local/bin/go ]; then
        GO_BIN="/usr/local/bin/go"
    elif command -v go >/dev/null 2>&1; then
        GO_BIN="$(command -v go)"
    else
        echo "Error: Go was not found. Install it with: brew install go" >&2
        exit 1
    fi
fi

echo "Building Palette $VERSION for macOS $ARCH"
echo "Using Go: $("$GO_BIN" version)"

rm -rf "$SHIP"
mkdir -p "$BIN" "$RELEASE_DIR"

cp "$PALETTE_SOURCE/VERSION" "$SHIP/"

build_cmd() {
    local name="$1"
    shift
    local package_dir="$PALETTE_SOURCE/cmd/$name"

    echo "================ Building $name"
    (
        cd "$package_dir"
        "$GO_BIN" build "$@" -o "$BIN/$name" .
    )
}

build_cmd palette
build_cmd palette_engine
build_cmd palette_monitor
build_cmd palette_chat
build_cmd palette_hub
build_cmd palette_remote -ldflags "-extldflags=-Wl,-no_warn_duplicate_libraries"
build_cmd palette_natsmon

copy_data_dir() {
    local name="$1"
    local source_dir="$PALETTE_SOURCE/$name"

    if [ -d "$source_dir" ]; then
        echo "================ Copying $name"
        /usr/bin/ditto "$source_dir" "$SHIP/$name"
        rm -rf "$SHIP/$name/config/chrome"
        rm -rf "$SHIP/$name/logs"
        rm -f "$SHIP/$name/saved/global/_Boot.json"
    fi
}

copy_data_dir data_default

if [ -f "$PALETTE_SOURCE/pkg/samplesplitter/assets/static/index.html" ]; then
    echo "================ Copying samplesplitter"
    /usr/bin/ditto "$PALETTE_SOURCE/pkg/samplesplitter/assets" "$SHIP/samplesplitter"
    rm -rf "$SHIP/samplesplitter/.git" "$SHIP/samplesplitter/__pycache__"
    if [ -d "$PALETTE_SOURCE/data_default/samplesplitter" ]; then
        /usr/bin/ditto "$PALETTE_SOURCE/data_default/samplesplitter" "$SHIP/samplesplitter"
    fi
else
    echo "Error: samplesplitter static UI is missing under pkg/samplesplitter/assets/static" >&2
    exit 1
fi

echo "================ Building FFGL bundle"
PALETTE_FFGL_INSTALL=0 "$SCRIPT_DIR/build_ffgl.sh"
mkdir -p "$SHIP/ffgl"
/usr/bin/ditto "$PALETTE_SOURCE/ffgl/binaries/release/Palette.bundle" "$SHIP/ffgl/Palette.bundle"

ZIP_NAME="palette_${VERSION}_macos_${ARCH}.zip"
ZIP_PATH="$RELEASE_DIR/$ZIP_NAME"
rm -f "$ZIP_PATH"

echo "================ Creating $ZIP_NAME"
(
    cd "$SHIP"
    /usr/bin/zip -qry "$ZIP_PATH" .
)

echo "================ Cleaning build artifacts"
rm -rf "$SHIP" "$SCRIPT_DIR/ffgl-build"

echo "================ Done"
echo "Release zip: $ZIP_PATH"
