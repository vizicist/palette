# Palette

Palette is the software used in the <a href=https://youtu.be/HDtxEyCI_zc>Space Palette Pro</a>,
an instrument that lets you fingerpaint sound and visuals
using your fingers as 3D cursors on Sensel Morph pads.

# Open Source software

### bin/palette_engine

   palette_engine is the Go-based central brain of the system that does the following:

   - Starts a realtime engine for looping and control of
      sound (via MIDI output) and visuals (via OSC to Resolume).

   - Monitors Morph and MIDI devices for input,

   - Has an API for 1) managing presets of sound and visual parameters, 2) starting and stopping various programs (Bidule, Resolume, etc), and 3) other things.

### bin/palette_gui

   A python-based executable that provides a graphical interface to select presets and edit their parameters.  It works for either a single Morph or four of them (as in the Space Palette Pro).

### bin/palette_monitor

   A Go-based executable that monitors the palette_engine process, so that if palette_engine crashes for any reason, it will be automatically restarted.

### ffgl/Palette.dll

   This Freeframe plugin draws basic visual shapes (sprites) in response to OSC messages sent from palette_engine.  The sprites are animated (moving, resizing, color fades) by the Freeframe plugin, but the position and all other parameters are sent by the palette_engine.

# Commercial software

### Resolume

   Resolume is the host for Freeframe plugins that generate and process visuals.  The output of the ffgl/Palette.dll plugin is processed by a pipeline of Freeframe effect plugins - the standard effects that come with Resolume.  There are four independent layers in Resolume - A, B, C, D - and the palette_engine controls all of the Freeframe effect parameters independently for each layer.

### Bidule

   Bidule is the VST host that listens for MIDI ouput from the engine.  Typically each MIDI input port in Bidule is distributed to 16 instances of a particular synth.
