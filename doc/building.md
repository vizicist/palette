## Building Palette software from the source code

NOTE: these instructions describe how
to recompile the Palette software from scratch.
If instead you just want to install it using one of the
released installation packages, use
<a href=installation.md>these instructions</a>

In the list below, the === instructions are only appropriate for machines that will be maintained remotely.

- Please send email to me@timthompson.com if these instructions don't work for you.

- When given a choice between 64-bit and 32-bin installations, choose 64-bit.

=== In a fresh Windows install, use spacepalette@gmail.com for the Microsoft and Google accounts.

=== Install Chrome, log in as spacepalette@gmail.com

=== Set up remotedesktop.google.com

- Install github desktop from https://desktop.github.com

- Install git from https://gitforwindows.org

- Use the github desktop (or CLI if you prefer) to clone this repo:
      https://github.com/vizicist/palette.git into
  into this directory: 
	%USERPROFILE%\Github\palette
   may take a while, and you can continue on to the next steps while that cooks.

- In your Environment Variables:
	- Add %USERPROFILE%\Github\palette\scripts to your PATH
 	- Add C:\Program Files\Git\bin to your PATH
	- Add C:\Program Files\Git\usr\bin to your PATH
 	- Add %USERPROFILE%\mingw64\bin to your PATH (adjust path if installed elsewhere)
	- Add the following new environment variables:
```
 PALETTE=C:\Program Files\Palette
 PALETTE_SOURCE=%USERPROFILE%\Github\palette
```
Not sure whether %USERPROFILE%\Github\palette\SenselLib\x64 is needed, it depends on how you execute things during development.

- Install Go version 1.19 or later from https://golang.org/dl/

- Install Python 3.9.6 (64-bit) or later, using the dowload site on python.org,
  and select the option for adding it to your PATH.
  DO NOT INSTALL python FROM THE WINDOWS STORE!

- Install Visual Studio Code from https://code.visualstudio.com/download

- Install Inno Setup from https://jrsoftware.org/isinfo.php

- Install Visual Studio Build Tools 2017 (version 15.9) from https://visualstudio.microsoft.com/downloads

	- in the Workload tab, select "Visual C++ Build Tools"
<p>

- Install mingw64 to get the gcc compiler.
The last time I installed it from https://github.com/niXman/mingw-builds-binaries, and it may be necessary to download
the "online installer" and execute it from the Explorer, selecting "more info" to allow installation of an unsigned package.
The version I'm using is the 13.1.0 version, 64 bit architecture, posix thread model, and ucrt runtime.

- In Visual Studio Code, click on the "extensions marketplace" icon (four little squares) on the left side.
  In the "Search Extensions" field, enter "go", and install the Go language support.
  Also install Python language support.
  Other pieces of the Go toolchain will be offered to you automatically within VSCode.

- in a cmd window, cd to %PALETTE_SOURCE% and execute:

	go mod tidy

	go get gitlab.com/gomidi/midi/v2/drivers/rtmididrv

- install LoopBe30 from https://nerds.de/en/loopbe30.html, and use its systray item to
turn off Shortcut Detection and enable 16 ports (which requires a reboot).

- install Kinect Runtime v1.7, Kinect SDK v1.7, and Kinect Developer Toolkit v1.7.0

- To compile everything, use a newly-created "cmd" window (so that the changes to PATH and the environment variable are reflected) and enter these lines:

```
cd %PALETTE_SOURCE%\build\windows
build
```

- Install SenselApp0.19.32 (for the Morph)

- Other useful things to install are:  7zip, sharpkeys
- The result of this should be an installer executable in the release directory,
which you should execute to install Palette.

- If this is the first time you've run the Palette software, you should follow the instructions for one-time steps
<a href=installation.md>here</a>.

- After that, you're ready to start using the Palette, as described 
<a href=starting_and_using.md>here</a>.

