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

echo "================ Copying data_default"
DATA_DEFAULT="$SHIP/data_default"
cp -r "$PALETTE_SOURCE/data_default" "$SHIP/"
# Create logs directory
mkdir -p "$DATA_DEFAULT/logs"
# Remove user-specific files that shouldn't be distributed
rm -f "$DATA_DEFAULT/saved/global/_Current.json"
rm -f "$DATA_DEFAULT/saved/global/_Boot.json"
rm -f "$DATA_DEFAULT/logs/"*.log 2>/dev/null || true

# Create release directory if it doesn't exist
RELEASE_DIR="$PALETTE_SOURCE/release"
mkdir -p "$RELEASE_DIR"

# Create zip file
ZIP_NAME="palette_${VERSION}_linux_amd64.zip"
echo "================ Creating $ZIP_NAME"
pushd "$SHIP" > /dev/null
zip -r "$RELEASE_DIR/$ZIP_NAME" .
popd > /dev/null

echo "================ Done"
echo "Installer created: $RELEASE_DIR/$ZIP_NAME"
echo ""
echo "Contents:"
unzip -l "$RELEASE_DIR/$ZIP_NAME"
