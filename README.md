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
   to generate MIDI (to a VST host) and OSC (to Resolume and the
   Palette FFGL plugin).

==================================================================
How to install and run in REMOTE mode
==================================================================

- Download ship/palette_win.zip from this repo
- Unzip it anywhere you like, it will produce a palette_win directory.
- Set a PALETTE environment variable to the full path of this palette_win
- Call %PALETTE%\bin\runall.bat

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

- instal Go version 1.15 or later from https://golang.org/dl/

- install visual studio code from https://code.visualstudio.com/download

- install visual studio 2019 community edition from https://visualstudio.microsoft.com/downloads

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

- in System Properties, add these directories to your PATH:

	C:\Program Files\Git\bin
	C:\Program Files\Git\usr\bin
	C:\Program Files\mingw-w64\x86_64-8.1.0-posix-sjlj-rt_v6-rev0\mingw64\bin
	%USERPROFILE%\go\bin
	%USERPROFILE%\go\src\github.com\vizicist\palette\scripts

  and add these environment variables:

	PALETTESOURCE=%USERPROFILE%\go\src\github.com\vizicist\palette
	PALETTE=%USERPROFILE%\go\src\github.com\vizicist\palette\ship\windows

- in Visual Studio Code, in the "extensions marketplace" section on the left side,
    type in "go" and install the Go language support.

- to compile everything on Windows:

	cd windows
        buildall


