# gomorph

A pure-Go, cross-platform reader for the [Sensel Morph](https://sensel.com/).
It talks to the Morph directly over its USB CDC-ACM serial port, so it needs
**no LibSensel** native library (`.dll`/`.so`/`.a`), **no cgo**, and **no IOKit**
on macOS. It builds and runs unchanged on Windows, macOS, and Linux /
Raspberry Pi, and cross-compiles cleanly from any of them.

It reads contacts from one or more Morphs and prints them and/or sends them as
OSC `/cursor` messages. It is a drop-in reimplementation of
[`vizicist/gomorph`](https://github.com/vizicist/gomorph) that removes the
LibSensel dependency.

## Why

The official Sensel SDK ships prebuilt native libraries only for Windows and
macOS (x64 / Apple Silicon). There is no build for ARM Linux, so the Morph
could not be used on a Raspberry Pi via the SDK. The Morph's API channel is,
however, just a **USB CDC-ACM virtual serial port** — on the Pi it appears as
`/dev/ttyACM*` with the in-kernel `cdc_acm` driver and needs nothing installed.
This program speaks that serial protocol in pure Go.

## Build

```bash
# From the repo root:
go build -o cmd/gomorph/gomorph.exe ./cmd/gomorph/     # Windows
go build -o cmd/gomorph/gomorph      ./cmd/gomorph/     # macOS / Linux
```

Cross-compiling (all targets are cgo-free, so no C toolchain is required):

```bash
CGO_ENABLED=0 GOOS=linux  GOARCH=arm64 go build -o gomorph ./cmd/gomorph/   # Raspberry Pi (64-bit)
CGO_ENABLED=0 GOOS=linux  GOARCH=arm   go build -o gomorph ./cmd/gomorph/   # Raspberry Pi (32-bit)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o gomorph ./cmd/gomorph/   # Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o gomorph ./cmd/gomorph/   # Intel Mac
```

## Run

```bash
gomorph                       # open all Morphs, print cursors, send OSC to 127.0.0.1:4444
gomorph -list                 # just list the Morphs found, then exit
gomorph -quiet                # send OSC but don't print cursor lines
gomorph -verbose              # log the serial number on every contact
gomorph -serial SM01174213529 # only use the Morph with this firmware serial
gomorph -device COM8          # open one serial device directly, skipping discovery
gomorph -listen               # act as an OSC server and print received /cursor messages
```

### Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `-list` | false | List Morphs and exit |
| `-quiet` | false | Don't print cursor lines (still sends OSC) |
| `-verbose` | false | Verbose per-contact logging |
| `-serial` | `*` | Only open the Morph with this firmware serial (`SM…`); `*` = all |
| `-device` | (none) | Open this exact serial device (e.g. `COM7`, `/dev/ttyACM0`, `/dev/cu.usbmodemXXXX`), bypassing enumeration |
| `-ip` | `127.0.0.1` | OSC target/listen IP |
| `-port` | `4444` | OSC UDP port |
| `-listen` | false | Run as an OSC receiver instead of reading Morphs |

Cursor output format (matches the original gomorph):

```
Morph: cursor <down|drag|up> <contact-id> <x> <y> <z>
```

`x` and `y` are normalized to `0..1` (y is flipped to match OpenGL/Freeframe);
`z` is force scaled by `MaxForce` (1500).

## Identifying Morphs (serial numbers)

A Morph has **two different serial identifiers** — this trips people up:

1. **USB descriptor iSerial** (e.g. `2031B1424B34`) — the low-level USB/MCU
   hardware ID exposed by the OS.
2. **Firmware / product serial** (e.g. `SM01174213529`, format `SM` + digits,
   printed on the unit) — read from device register `0x0F`.

The Sensel SDK and the palette engine use **#2**, and `morphs.json` keys on it
to map each Morph to A/B/C/D. gomorph therefore reads register `0x0F` and
reports the `SM…` serial (matching `engine.log`). The `-serial` filter matches
this value.

## Device names by platform

Each Morph is a separate USB device, so with four connected you get four nodes:

| Platform | Device names | Notes |
|----------|--------------|-------|
| Windows | `COM7`, `COM8`, … | Assigned by Windows |
| Linux / Pi | `/dev/ttyACM0`, `/dev/ttyACM1`, … | Numbered by enumeration order — **not stable** across reboots/replug |
| macOS | `/dev/cu.usbmodem…` | Use the `cu.` (callout) node, not `tty.` |

Because gomorph identifies devices by their `SM…` firmware serial (read from
each port), the OS device-node numbering does not matter — the `morphs.json`
mapping works regardless of which node the kernel assigned.

**Raspberry Pi power note:** four Morphs draw meaningful USB current. Use a
*powered* USB hub rather than hanging all four directly off the Pi to avoid
brown-outs and disconnects.

## How it works (for maintainers)

The Morph's API interface is USB CDC-ACM at **115200 8N1** (baud is nominal for
CDC). The protocol is a simple register read/write scheme; this implementation
was written from Sensel's MIT-licensed SDK
([sensel/sensel-api](https://github.com/sensel/sensel-api)) and verified
byte-for-byte against firmware 0.19 build 298.

Wire framing (`checksum = sum(bytes) & 0xFF`):

```
Register read   TX (3B): [0x81, reg, size]           (0x81 = board 0x01 | read-bit 0x80)
                RX     : ack(1)=PT_READ_ACK  reg(1)  size(2 LE)  payload(size)  checksum(1)
Register write  TX     : [0x01, reg, size]  payload  checksum(1)
                RX     : ack(1)=PT_WRITE_ACK  reg_echo(1)
Var-size read   TX (3B): [0x81, reg, 0x00]
                RX     : ack(3)  size(2 LE)  payload(size)  checksum(1)
Frame read      TX (3B): [0x81, 0x26, 0x00]           (0x26 = SCAN_READ_FRAME)
                RX     : ack(1)=PT_RVS_ACK  reg(1)  header(1)  size(2 LE)  payload(size)  checksum(1)
```

Open / start sequence (per device), mirroring the SDK:

1. Open the serial port, DTR/RTS disabled.
2. Read register `0x00` (magic) → must be `"S3NS31"`.
3. Read the `SM…` serial (variable-size read of register `0x0F`).
4. Soft reset (write `0xE0` = 1), then read firmware, sensor dimensions and
   unit scales.
5. Configure: disable timeouts (`0xD0` = 255), set frame content to contacts
   only (`0x24`), start synchronous scanning (`0x25`), turn LEDs off.
6. Poll: request one frame at a time (`0x26`), parse contacts.

Frame payload layout: `content_bit_mask(1) rolling_counter(1) timestamp(4)`
followed, when the contacts bit is set, by
`contact_bit_mask(1) num_contacts(1)` and then each contact's 10-byte default
block (`id, state, x, y, force, area`), plus any optional per-contact fields
selected by the contact bit mask.

### Source layout

| File | Role |
|------|------|
| `main.go` | CLI, OSC output, `-listen` server |
| `morph/morph.go` | Public API: `OneMorph`, `Init`, `InitPort`, `Start`, event types |
| `morph/sensel.go` | Serial protocol: register read/write/VS, setup, frame parsing |
| `morph/enumerate.go` | Shared `morphPort` type |
| `morph/enumerate_default.go` | Windows/Linux discovery via USB VID `2C2F` (build tag `!darwin`) |
| `morph/enumerate_darwin.go` | macOS discovery by globbing `/dev/cu.usbmodem*` (build tag `darwin`, no IOKit) |

On Windows and Linux, discovery uses `go.bug.st/serial/enumerator` (pure Go on
those platforms) to filter by Sensel's USB vendor ID. On macOS, reading USB
metadata would require Apple's IOKit framework (and thus cgo), so discovery
instead globs `/dev/cu.usbmodem*` and confirms each candidate is a Morph via
the magic + serial reads above — keeping the macOS build pure Go as well.
