## Building Palette on macOS

These notes cover a local macOS development build. The full performance stack
still has Windows-specific pieces, but the Go command binaries can be built and
the engine can be started by itself for web UI and API development.

## Install Go

Install a current Go toolchain with Homebrew:

```
brew install go
```

If an older Go from `/usr/local/go/bin` appears first in `PATH`, use the
Homebrew binary directly or move `/opt/homebrew/bin` earlier in `PATH`:

```
export PATH="/opt/homebrew/bin:$PATH"
go version
```

The module currently targets Go 1.25 or later.

## Build

For a local build in the command directories, build each command from its command
directory. The binaries are written into those same directories.

```
cd cmd/palette_engine && go build .
cd ../palette && go build .
cd ../palette_hub && go build .
cd ../palette_monitor && go build .
```

For a packaged macOS build, use:

```
cd build/macos
./build.sh
```

This builds the command binaries into `build/macos/ship`, copies the bundled
data directories, and creates a release zip under `release/`.

## Install

Install the most recent macOS build artifact with:

```
cd build/macos
./install.sh
```

By default this installs support files and bundled data under
`~/Library/Application Support/Palette` and creates command symlinks in `~/bin`.
To install somewhere else:

```
PALETTE_INSTALL_DIR="$HOME/Other Palette" PALETTE_BIN_DIR="$HOME/bin" ./install.sh
```

You can also pass an explicit zip or ship directory:

```
./install.sh ../../release/palette_VERSION_macos_ARCH.zip
./install.sh ./ship
```

## Runtime Environment

For an installed system, set `PALETTE` to the installed Palette root. For local
development on macOS, Palette detects the source checkout when run from inside
the repo and uses that as the Palette root. The `palette` CLI also looks for
sibling development binaries such as `cmd/palette_engine/palette_engine`.

For the default macOS user install:

```
export PALETTE="$HOME/Library/Application Support/Palette"
export PATH="$HOME/bin:$PATH"
```

`PALETTE_DATA` selects the data directory suffix. If unset, Palette uses
`data_default` under the detected Palette root.

## Engine-Only Startup

After building `cmd/palette` and `cmd/palette_engine`, start only the engine:

```
cd cmd/palette
./palette start engineonly
```

The web UI is served by the engine at:

```
http://127.0.0.1:3330/
```

The API endpoint is:

```
http://127.0.0.1:3330/api
```

## Current macOS Scope

The macOS process layer can launch and detect Unix-style command binaries. The
Windows-specific pieces, including Kinect/MMTT binaries, LoopBe30, and the
Windows Resolume/Bidule defaults, still need separate macOS configuration before
the complete instrument stack can auto-start cleanly.
