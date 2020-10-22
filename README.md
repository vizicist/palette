THIS DOCUMENT IS UNDER CONSTRUCTION

Palette - a visual music instrument

Palette is the engine used in the Space Palette Pro,
an instrument that lets you fingerpaint sound and visuals
using your fingers as 3D cursors on Sensel Morph pads.

It consists of:

   1) A realtime engine that accepts 3D Cursor and MIDI input
      and generates as output MIDI (for softsynths) and OSC (for Resolume).

   2) A GUI that lets you control sound and visual presets for the
      musical and graphical reactions to your finger gestures.

There are several ways of running Palette:

   In REMOTE mode, all input (from the GUI and 3D Cursor input) is
   broadcast to the internet, to be received by a remote host
   that is running Palette in LOCAL mode.

   In LOCAL mode, input (local or remotely-generated) is processed
   to generate MIDI (typicall to a VST host) and OSC (typically
   to Resolume and Palette's FFGL plugin running in it).

==================================================================
How to install
==================================================================

- Download and execute ship/palette_setup_win.exe from this repo

==================================================================
How to run in REMOTE mode
==================================================================
- Execute C:\Program Files\Palette\bin\runall.bat

==================================================================
IMPLEMENTATION DETAILS
==================================================================

palette.exe ======================================

	A Go-based executable that

	1) Starts a NATS server for inter-process
	   API and event communication both locally and to the internet,

	2) Monitors Morph and MIDI devices for input,

	3) Starts a realtime engine for looping and control of
	   sound (MIDI) and visuals (OSC to Resolume and a FFGL plugin).

gui.exe ==========================================

	A python-based executable that provides a graphical interface
	to select presets and edit their parameters.

palette.dll ======================================

	A C/C++-based FFGL plugin with an OSC interface for visual output


==================================================================
Compiling on a brand new Windows machine for development
==================================================================

- install git from https://gitforwindows.org

- install Go version 1.15 or later from https://golang.org/dl/

- install Python 3.8.6 or later (BUT NOT FROM THE WINDOWS STORE!), in C:\Python38

- install Visual Studio Code from https://code.visualstudio.com/download

- install Inno Setup from https://jrsoftware.org/isinfo.php

- install Visual Studio 2019 Community Edition from https://visualstudio.microsoft.com/downloads

	- in the Workload tab, select "Desktop Development with C++"
	- In the Individual components tab, select "Msbuild"

- install mingw64 using this installer:

        https://sourceforge.net/projects/mingw-w64/files/Toolchains%20targetting%20Win32/Personal%20Builds/mingw-builds/installer/mingw-w64-install.exe

        *** USE THESE SETTINGS WHEN INSTALLING mingw64 ***
        Version: 8.1.0
        Architecture: x86_64
        Threads: posix
        Exception: sjlj

- set up the repos as follows

	mkdir %USERPROFILE%\go\src\github.com\vizicist
	cd %USERPROFILE%\go\src\github.com\vizicist
	git clone https://github.com/vizicist/palette.git
	git clone https://github.com/vizicist/portmidi.git

- go get a few packages:

	go get github.com/hypebeast/go-osc/osc
	go get github.com/nats-io/nats-server/server
	go get github.com/nats-io/nats.go
	go get github.com/nats-io/nuid
	go get gopkg.in/mail.v2

- Make sure these directories are added to your PATH:

	C:\Program Files\Git\bin
	C:\Program Files\Git\usr\bin
	C:\Program Files\mingw-w64\x86_64-8.1.0-posix-sjlj-rt_v6-rev0\mingw64\bin
	%USERPROFILE%\go\bin
	%USERPROFILE%\go\src\github.com\vizicist\palette\scripts

  and add this environment variables:

	PALETTESOURCE=%USERPROFILE%\go\src\github.com\vizicist\palette

- in Visual Studio Code, in the "extensions marketplace" section on the left side,
    type in "go" and install the Go language support.

- to compile everything on Windows:

	cd windows
        build
