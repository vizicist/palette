#!/bin/bash

# Self-extracting installer for Palette Linux binaries
# Installs to /usr/local/palette and creates symlinks in /usr/local/bin
#
# Usage:
#   sudo ./palette_8.20_linux_amd64.sh
#
# Can also be used standalone with a zip file:
#   sudo ./install.sh palette_8.20_linux_amd64.zip

set -e  # Exit on error

INSTALL_DIR="/usr/local/palette"
BIN_DIR="/usr/local/bin"

# Check if this script has a zip payload appended (self-extracting mode)
ARCHIVE_MARKER="__ARCHIVE_BELOW__"
ARCHIVE_LINE=$(grep -an "$ARCHIVE_MARKER" "$0" | tail -1 | cut -d: -f1)

if [ -n "$ARCHIVE_LINE" ]; then
    # Self-extracting mode: extract payload from this script
    ZIP_FILE=$(mktemp /tmp/palette_install_XXXXXX.zip)
    trap "rm -f '$ZIP_FILE'" EXIT
    tail -n +$((ARCHIVE_LINE + 1)) "$0" > "$ZIP_FILE"
    VERSION=$(unzip -p "$ZIP_FILE" VERSION 2>/dev/null || echo "unknown")
else
    # Standalone mode: find zip file from argument or auto-detect
    ZIP_FILE="${1:-}"
    if [ -z "$ZIP_FILE" ]; then
        SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        for dir in "." "$SCRIPT_DIR"; do
            match=$(ls "$dir"/palette_*_linux_amd64.zip 2>/dev/null | head -1)
            if [ -n "$match" ]; then
                ZIP_FILE="$match"
                break
            fi
        done
    fi

    # Convert to absolute path
    if [ -n "$ZIP_FILE" ] && [ "${ZIP_FILE:0:1}" != "/" ]; then
        ZIP_FILE="$(pwd)/$ZIP_FILE"
    fi

    if [ -z "$ZIP_FILE" ] || [ ! -f "$ZIP_FILE" ]; then
        echo "Error: No palette zip file found"
        echo "Usage: $0 [palette_VERSION_linux_amd64.zip]"
        exit 1
    fi
    VERSION=$(basename "$ZIP_FILE" | sed 's/palette_\(.*\)_linux_amd64.zip/\1/')
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
echo "Extracting to $INSTALL_DIR..."
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
exit 0
