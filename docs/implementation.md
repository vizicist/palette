# Implementation details

## bin\palette_engine.exe

	A Go-based executable that

	1) Starts a NATS server for inter-process
	   API and event communication both locally and to the internet,

	2) Monitors Morph and MIDI devices for input,

	3) Starts a realtime engine for looping and control of
	   sound (MIDI) and visuals (OSC to Resolume and a FFGL plugin).

## bin\palette_gui_*.exe

	A python-based executable that provides a graphical interface
	to select presets and edit their parameters.

## ffgl\palette.dll

	A C/C++-based FFGL plugin with an OSC interface for visual output
