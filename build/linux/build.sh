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

# Copy the default data tree.
#
# The release used to package only binaries, but the engine resolves its data
# path to /usr/local/palette/data_default (see PaletteDataPath in kit/misc.go),
# so an install had no paramdefs.json, no presets and no config at all - the
# engine came up unable to do anything. Sanitized the same way build_data.bat
# does for Windows: the working state and the per-installation attract videos
# are not part of a release.
echo "================ Copying data_default"
cp -R "$PALETTE_SOURCE/data_default" "$SHIP/data_default"
rm -f "$SHIP/data_default/saved/global/_Current.json"
rm -f "$SHIP/data_default/saved/global/_Boot.json"
rm -rf "$SHIP/data_default/config/chrome"
find "$SHIP/data_default/logs" -type f ! -name '.gitignore' -delete 2>/dev/null || true
# Attract videos are per-installation content; ship the README that explains
# how to add them, never the files themselves.
if [ -d "$SHIP/data_default/config/attractmode_videos" ]; then
    find "$SHIP/data_default/config/attractmode_videos" -type f ! -name 'README.md' -delete
fi

if [ -f "$PALETTE_SOURCE/pkg/samplesplitter/assets/static/index.html" ]; then
    echo "================ Copying samplesplitter"
    cp -R "$PALETTE_SOURCE/pkg/samplesplitter/assets" "$SHIP/samplesplitter"
    rm -rf "$SHIP/samplesplitter/.git" "$SHIP/samplesplitter/__pycache__"
    if [ -d "$PALETTE_SOURCE/data_default/samplesplitter" ]; then
        cp -R "$PALETTE_SOURCE/data_default/samplesplitter/." "$SHIP/samplesplitter/"
    fi
else
    echo "Error: samplesplitter static UI is missing under pkg/samplesplitter/assets/static" >&2
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
