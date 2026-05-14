#!/bin/bash

# Build script for palette Linux binaries
# Creates a zip installer with palette and palette_hub

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTE_SOURCE="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Read version
VERSION=$(cat "$PALETTE_SOURCE/VERSION")
echo "Building Palette version $VERSION for Linux"

# Create ship directory
SHIP="$SCRIPT_DIR/ship"
BIN="$SHIP/bin"
rm -rf "$SHIP"
mkdir -p "$BIN"

# Copy VERSION file
cp "$PALETTE_SOURCE/VERSION" "$SHIP/"

if [ -f "$PALETTE_SOURCE/cmd/samplesplitter/samplesplitter.py" ]; then
    echo "================ Copying samplesplitter"
    cp -R "$PALETTE_SOURCE/cmd/samplesplitter" "$SHIP/samplesplitter"
    rm -rf "$SHIP/samplesplitter/.git" "$SHIP/samplesplitter/__pycache__"
    if [ -d "$PALETTE_SOURCE/data_default/samplesplitter" ]; then
        cp -R "$PALETTE_SOURCE/data_default/samplesplitter/." "$SHIP/samplesplitter/"
    fi
else
    echo "Error: samplesplitter source is missing under cmd/samplesplitter" >&2
    exit 1
fi

echo "================ Building palette"
pushd "$PALETTE_SOURCE/cmd/palette" > /dev/null
go build -o "$BIN/palette" .
popd > /dev/null

echo "================ Building palette_hub"
pushd "$PALETTE_SOURCE/cmd/palette_hub" > /dev/null
go build -o "$BIN/palette_hub" .
popd > /dev/null

echo "================ Building palette_engine"
pushd "$PALETTE_SOURCE/cmd/palette_engine" > /dev/null
go build -o "$BIN/palette_engine" .
popd > /dev/null

echo "================ Building samplesplitter"
pushd "$PALETTE_SOURCE/cmd/samplesplitter" > /dev/null
go build -o "$BIN/samplesplitter" .
popd > /dev/null

# Create release directory if it doesn't exist
RELEASE_DIR="$PALETTE_SOURCE/release"
mkdir -p "$RELEASE_DIR"

# Create self-extracting installer
INSTALLER_NAME="palette_${VERSION}_linux_amd64.sh"
ZIP_TMP="/tmp/palette_build_$$.zip"
rm -f "$ZIP_TMP"
echo "================ Creating $INSTALLER_NAME"
pushd "$SHIP" > /dev/null
zip -rq "$ZIP_TMP" .
popd > /dev/null
cat "$SCRIPT_DIR/install.sh" > "$RELEASE_DIR/$INSTALLER_NAME"
echo "__ARCHIVE_BELOW__" >> "$RELEASE_DIR/$INSTALLER_NAME"
cat "$ZIP_TMP" >> "$RELEASE_DIR/$INSTALLER_NAME"
chmod +x "$RELEASE_DIR/$INSTALLER_NAME"
rm -f "$ZIP_TMP"

echo "================ Done"
echo "Installer created: $RELEASE_DIR/$INSTALLER_NAME"

rm -rf "$SHIP"
