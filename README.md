# Palette - a system for visual music instruments

Palette is the software used in the Space Palette Pro,
an instrument that lets you fingerpaint sound and visuals
using your fingers as 3D cursors on Sensel Morph pads.

Palette consists of:

* A realtime engine that accepts as input 3D cursors and MIDI,
      and generates as output MIDI (for softsynths) and OSC (for Resolume).

* A GUI that lets you control the presets and parameters of
      the musical and graphical output.

* A Freeframe plugin that runs inside Resolume.

There are several ways of running Palette:

Normally, input from the Sensel Morph is processed
to generate MIDI (sent to a VST host)
and OSC (sent to Resolume and Palette's FFGL plugin running inside it).

Optionally, Palette can run in <i>remote</a> mode, where you can remotely control
a Space Palette Pro running elsewhere in the network.
This allows you to collaborate and perform with other people in a
coordinated visual music environment.

# Installing on Windows

- Download and execute the latest release/palette_#.#_win_setup.exe from this repo

# Using with Resolume and a Sensel Morph

- After installing Palette, you can use it as a visual instrument in <a href=https://resolume.com>Resolume 6</a> by using the <a href=https://sensel.com>Sensel Morph</a> to fingerpaint.

# Other documentation

- <a href=docs/implementation.md>Implementation details</a>

- <a href=docs/building.md>Configuring a build/development environment on Windows</a>

