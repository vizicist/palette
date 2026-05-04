#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PALETTE_SOURCE="$(cd "$SCRIPT_DIR/../.." && pwd)"
FFGL_DIR="$PALETTE_SOURCE/ffgl"
OUT_DIR="$FFGL_DIR/binaries/release"
BUNDLE="$OUT_DIR/Palette.bundle"
INSTALL_FFGL_DIR="${PALETTE_FFGL_DIR:-$HOME/Library/Application Support/Palette/ffgl}"
INSTALL_BUNDLE="${PALETTE_FFGL_INSTALL:-1}"
BUILD_DIR="$SCRIPT_DIR/ffgl-build"

CXX="${CXX:-clang++}"
CC="${CC:-clang}"
SDKROOT="$(xcrun --sdk macosx --show-sdk-path)"
ARCHS="${ARCHS:-x86_64 arm64}"

COMMON_FLAGS=(
    -isysroot "$SDKROOT"
    -fPIC
    -DNDEBUG=1
    -D_NDEBUG=1
    -DTARGET_OS_MAC=1
    -I"$FFGL_DIR/source/lib/macos_compat"
    -I"$FFGL_DIR/source/lib"
    -I"$FFGL_DIR/source/lib/ffgl"
    -I"$FFGL_DIR/source/lib/ffglex"
    -I"$FFGL_DIR/source/lib/ffglquickstart"
    -I"$FFGL_DIR/source/lib/glm"
    -I"$FFGL_DIR/source/lib/palette"
    -I"$FFGL_DIR/source/lib/nosuch"
    -I"$FFGL_DIR/source/lib/oscpack"
    -I"$FFGL_DIR/source/lib/osc/include"
    -I"$FFGL_DIR/source/lib/cJSON"
    -I"$FFGL_DIR/source/plugins/Palette"
)

SOURCES_CPP=(
    "$FFGL_DIR/source/lib/FFGLSDK.cpp"
    "$FFGL_DIR/source/lib/cJSON/cJSON.cpp"
    "$FFGL_DIR/source/lib/nosuch/NosuchDebug.cpp"
    "$FFGL_DIR/source/lib/nosuch/NosuchOscInput.cpp"
    "$FFGL_DIR/source/lib/nosuch/NosuchOscUdpInput.cpp"
    "$FFGL_DIR/source/lib/nosuch/NosuchUtil.cpp"
    "$FFGL_DIR/source/lib/nosuch/UriCodec.cpp"
    "$FFGL_DIR/source/lib/oscpack/ip/IpEndpointName.cpp"
    "$FFGL_DIR/source/lib/oscpack/ip/posix/NetworkingUtils.cpp"
    "$FFGL_DIR/source/lib/oscpack/ip/posix/UdpSocket.cpp"
    "$FFGL_DIR/source/lib/oscpack/osc/OscOutboundPacketStream.cpp"
    "$FFGL_DIR/source/lib/oscpack/osc/OscPrintReceivedElements.cpp"
    "$FFGL_DIR/source/lib/oscpack/osc/OscReceivedElements.cpp"
    "$FFGL_DIR/source/lib/oscpack/osc/OscTypes.cpp"
    "$FFGL_DIR/source/lib/osc/src/OscBundle.cpp"
    "$FFGL_DIR/source/lib/osc/src/OscMessage.cpp"
    "$FFGL_DIR/source/lib/osc/src/OscSender.cpp"
    "$FFGL_DIR/source/lib/palette/Palette.cpp"
    "$FFGL_DIR/source/lib/palette/PaletteDrawer.cpp"
    "$FFGL_DIR/source/lib/palette/PaletteHost.cpp"
    "$FFGL_DIR/source/lib/palette/PaletteOscInput.cpp"
    "$FFGL_DIR/source/lib/palette/PaletteUtil.cpp"
    "$FFGL_DIR/source/lib/palette/Layer.cpp"
    "$FFGL_DIR/source/lib/palette/Scheduler.cpp"
    "$FFGL_DIR/source/lib/palette/Sprite.cpp"
    "$FFGL_DIR/source/lib/palette/SvgSprite.cpp"
    "$FFGL_DIR/source/lib/palette/TrackedCursor.cpp"
    "$FFGL_DIR/source/plugins/Palette/FFGLPalette.cpp"
)

SOURCES_C=(
    "$FFGL_DIR/source/lib/nosuch/sha1.c"
)

rm -rf "$BUILD_DIR" "$BUNDLE"
mkdir -p "$BUILD_DIR" "$BUNDLE/Contents/MacOS" "$OUT_DIR"
if [ "$INSTALL_BUNDLE" != "0" ]; then
    mkdir -p "$INSTALL_FFGL_DIR"
fi

OBJECTS=()

BINARIES=()

for arch in $ARCHS; do
    arch_build_dir="$BUILD_DIR/$arch"
    arch_binary="$arch_build_dir/Palette"
    mkdir -p "$arch_build_dir"
    OBJECTS=()

    echo "================ Compiling Palette FFGL for macOS $arch"
    for src in "${SOURCES_CPP[@]}"; do
        obj="$arch_build_dir/$(basename "$src").o"
        "$CXX" -std=c++17 -stdlib=libc++ "${COMMON_FLAGS[@]}" -target "$arch-apple-macos10.15" -c "$src" -o "$obj"
        OBJECTS+=("$obj")
    done

    for src in "${SOURCES_C[@]}"; do
        obj="$arch_build_dir/$(basename "$src").o"
        "$CC" "${COMMON_FLAGS[@]}" -target "$arch-apple-macos10.15" -c "$src" -o "$obj"
        OBJECTS+=("$obj")
    done

    echo "================ Linking Palette.bundle for $arch"
    "$CXX" -bundle -stdlib=libc++ -isysroot "$SDKROOT" -target "$arch-apple-macos10.15" \
        -framework OpenGL -framework Carbon -framework AppKit \
        "${OBJECTS[@]}" -o "$arch_binary"
    BINARIES+=("$arch_binary")
done

if [ "${#BINARIES[@]}" -eq 1 ]; then
    cp "${BINARIES[0]}" "$BUNDLE/Contents/MacOS/Palette"
else
    echo "================ Creating universal Palette.bundle"
    lipo -create "${BINARIES[@]}" -output "$BUNDLE/Contents/MacOS/Palette"
fi

cat > "$BUNDLE/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>English</string>
	<key>CFBundleExecutable</key>
	<string>Palette</string>
	<key>CFBundleIdentifier</key>
	<string>com.vizicist.palette.ffgl</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundlePackageType</key>
	<string>BNDL</string>
	<key>CFBundleSignature</key>
	<string>????</string>
	<key>CFBundleVersion</key>
	<string>1.0</string>
	<key>CSResourcesFileMapped</key>
	<string>yes</string>
</dict>
</plist>
PLIST

echo "Built: $BUNDLE"
echo "================ Signing Palette.bundle"
codesign --force --deep --sign - "$BUNDLE"
if [ "$INSTALL_BUNDLE" != "0" ]; then
    echo "================ Installing Palette.bundle"
    /usr/bin/ditto "$BUNDLE" "$INSTALL_FFGL_DIR/Palette.bundle"
    echo "Installed: $INSTALL_FFGL_DIR/Palette.bundle"
fi
