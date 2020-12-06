## The Palette software consists of:

### bin\palette_engine.exe

A Go-based executable that

- Starts a NATS server for inter-process
   API and event communication both locally and to the internet,

- Monitors Morph and MIDI devices for input,

- Starts a realtime engine for looping and control of
   sound (MIDI) and visuals (OSC to Resolume and a FFGL plugin).

### bin\palette_gui.exe

- A python-based executable that provides a graphical interface
to select presets and edit their parameters.

### ffgl\palette.dll

- A C/C++-based FFGL plugin with an OSC interface for visual output
