#!/bin/bash

# Install script for palette Linux binaries
# Installs to /usr/local/palette and creates symlinks in /usr/local/bin

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTE_SOURCE="$(cd "$SCRIPT_DIR/../.." && pwd)"
INSTALL_DIR="/usr/local/palette"
BIN_DIR="/usr/local/bin"

# Read version
VERSION=$(cat "$PALETTE_SOURCE/VERSION")
ZIP_FILE="$PALETTE_SOURCE/release/palette_${VERSION}_linux_amd64.zip"

# Check if zip file exists
if [ ! -f "$ZIP_FILE" ]; then
    echo "Error: $ZIP_FILE not found"
    echo "Run ./build.sh first to create the installer"
    exit 1
fi

# Check for root/sudo
if [ "$EUID" -ne 0 ]; then
    echo "This script requires root privileges. Re-running with sudo..."
    exec sudo "$0" "$@"
fi

echo "Installing Palette version $VERSION"

# Create palette user if it doesn't exist
if ! id -u palette >/dev/null 2>&1; then
    echo "Creating 'palette' user..."
    useradd --system --no-create-home --shell /usr/sbin/nologin palette
fi

# Create install directory
mkdir -p "$INSTALL_DIR"

# Extract zip file, only updating files that are newer than existing ones
# -u = update (only extract newer files or new files)
# -o = overwrite without prompting
echo "Extracting to $INSTALL_DIR (preserving newer existing files)..."
unzip -uoq "$ZIP_FILE" -d "$INSTALL_DIR"

# Make binaries executable
chmod +x "$INSTALL_DIR/bin/"*

# Create symlinks in /usr/local/bin
echo "Creating symlinks in $BIN_DIR..."
for bin in "$INSTALL_DIR/bin/"*; do
    name=$(basename "$bin")
    ln -sf "$bin" "$BIN_DIR/$name"
    echo "  $name -> $bin"
done

# Set ownership to palette user
echo "Setting ownership to 'palette' user..."
chown -R palette:palette "$INSTALL_DIR"

echo ""
echo "Installation complete!"
echo "  Install directory: $INSTALL_DIR"
echo "  Owner: palette"
echo "  Binaries linked to: $BIN_DIR"
echo ""
echo "You can now run: palette, palette_hub, palette_engine"
