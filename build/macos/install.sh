#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTE_SOURCE="$(cd "$SCRIPT_DIR/../.." && pwd)"
INSTALL_DIR="${PALETTE_INSTALL_DIR:-$HOME/Library/Application Support/Palette}"
BIN_DIR="${PALETTE_BIN_DIR:-$HOME/bin}"
SOURCE="${1:-}"
TMP_DIR=""

usage() {
    cat <<EOF
Usage: $0 [palette_VERSION_macos_ARCH.zip|ship-directory]

Environment:
  PALETTE_INSTALL_DIR  Install root, default: ~/Library/Application Support/Palette
  PALETTE_BIN_DIR      Symlink directory, default: ~/bin
EOF
}

find_default_source() {
    local match

    if [ -d "$SCRIPT_DIR/ship" ]; then
        echo "$SCRIPT_DIR/ship"
        return
    fi

    match="$(ls -t "$PALETTE_SOURCE"/release/palette_*_macos_*.zip 2>/dev/null | head -1 || true)"
    if [ -n "$match" ]; then
        echo "$match"
    fi
}

if [ "${SOURCE:-}" = "-h" ] || [ "${SOURCE:-}" = "--help" ]; then
    usage
    exit 0
fi

if [ -z "$SOURCE" ]; then
    SOURCE="$(find_default_source)"
fi

if [ -z "$SOURCE" ] || [ ! -e "$SOURCE" ]; then
    echo "Error: no macOS build artifact found." >&2
    usage >&2
    exit 1
fi

if [ "${SOURCE:0:1}" != "/" ]; then
    SOURCE="$(pwd)/$SOURCE"
fi

if [ -f "$SOURCE" ]; then
    TMP_DIR="$(mktemp -d /tmp/palette_macos_install_XXXXXX)"
    trap 'rm -rf "$TMP_DIR"' EXIT
    /usr/bin/ditto -x -k "$SOURCE" "$TMP_DIR"
    SOURCE="$TMP_DIR"
fi

if [ ! -d "$SOURCE/bin" ]; then
    echo "Error: install source does not contain a bin directory: $SOURCE" >&2
    exit 1
fi

VERSION="unknown"
if [ -f "$SOURCE/VERSION" ]; then
    VERSION="$(tr -d '\r\n' < "$SOURCE/VERSION")"
fi

echo "Installing Palette $VERSION"
echo "Install directory: $INSTALL_DIR"
echo "Symlink directory: $BIN_DIR"

mkdir -p "$INSTALL_DIR" "$BIN_DIR"

echo "Copying files..."
/usr/bin/ditto "$SOURCE" "$INSTALL_DIR"

echo "Making binaries executable..."
chmod +x "$INSTALL_DIR/bin/"*

echo "Creating symlinks..."
for bin in "$INSTALL_DIR/bin/"*; do
    name="$(basename "$bin")"
    ln -sf "$bin" "$BIN_DIR/$name"
    echo "  $name -> $bin"
done

echo ""
echo "Installation complete."
echo "To use this install in a shell:"
echo "  export PALETTE=\"$INSTALL_DIR\""
echo "  export PATH=\"$BIN_DIR:\$PATH\""
