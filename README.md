# Palette - a system for visual music instruments

Palette is the engine used in the Space Palette Pro,
an instrument that lets you fingerpaint sound and visuals
using your fingers as 3D cursors on Sensel Morph pads.

Palette consists of:

   1) A realtime engine that accepts as input 3D cursors and MIDI,
      and generates as output MIDI (for softsynths) and OSC (for Resolume).

   2) A GUI that lets you control the presets and parameters of
      the musical and graphical output.

   3) A Freeframe plugin that runs inside Resolume.

There are several ways of running Palette:

   In REMOTE mode, Resolume is not required - all input (from the
   GUI and 3D Cursor input) is broadcast to the internet,
   to be received by a remote host.

   In LOCAL mode, input (both local or remotely-generated) is processed
   to generate MIDI (typically sent to a VST host) and OSC (typically
   sent to Resolume and Palette's FFGL plugin running inside it).

# How to install on Windows

- Download and execute the latest release/palette_#.#_win_setup.exe from this repo

# Other documentation

* <a href=docs/implementation.md>Implementation details</a>

* <a href=docs/building.md>Configuring a build/development environment on Windows</a>

