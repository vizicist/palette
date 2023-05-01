## Initializing a Palette build and development environment.

NOTE: these instructions are only appropriate if you are wanting
to recompile the Palette software from scratch.
If instead you just want to install it using one of the
released installation packages, which is certainly a lot simpler and is recommended, use:
<a href="https://github.com/vizicist/palette/blob/main/doc/using_resolume.md">https://github.com/vizicist/palette/blob/main/doc/using_resolume.md</a>

In the list below, the === instructions are only appropriate for machines that will be maintained by Tim.

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
	%USERPROFILE%\Documents\Github\palette
  That may take a while, and you can continue on to the next steps while that cooks.

- In your Environment Variables:
	- Add %USERPROFILE%\Documents\Github\palette\scripts to your PATH
 	- Add C:\Program Files\Git\bin to your PATH
	- Add C:\Program Files\Git\usr\bin to your PATH
 	- Add %USERPROFILE%\mingw64\bin to your PATH (adjust path if installed elsewhere)
	- Add the following new environment variables:
```
 PALETTE=C:\Program Files\Palette
 PALETTESOURCE=%USERPROFILE%\Documents\Github\palette
 PALETTE_DATA_PATH=%USERPROFILE%\Documents\Github\palette\data_omnisphere
```

	Not sure whether %USERPROFILE%\Documents\Github\palette\SenselLib\x64 is needed, it depends on how you execute things during development.

- Install Go version 1.19 or later from https://golang.org/dl/

- Install Python 3.9.6 (64-bit) or later, using the dowload site on python.org,
  and select the option for adding it to your PATH.
  DO NOT INSTALL python FROM THE WINDOWS STORE!

- Install Visual Studio Code from https://code.visualstudio.com/download

- Install Inno Setup from https://jrsoftware.org/isinfo.php

- Install Visual Studio Build Tools 2017 from https://visualstudio.microsoft.com/downloads

	- in the Workload tab, select "Desktop Development with C++"

- Install mingw64 to get the gcc compiler.  The online installers don't always work.  You'll want the 8.5.0 version, x86_64 architecture, posix threads, and sjlj exceptions.
The last time I installed it from https://github.com/niXman/mingw-builds-binaries

- In Visual Studio Code, click on the "extensions marketplace" icon (four little squares) on the left side.
  In the "Search Extensions" field, enter "go", and install the Go language support.
  Also install Python language support.
  Other pieces of the Go toolchain will be offered to you automatically within VSCode.

- in a cmd window, cd to %PALETTESOURCE% and execute:
	go get gitlab.com/gomidi/midi/v2/drivers/rtmididrv

- install LoopBe30 from https://nerds.de/en/loopbe30.html and enable 16 ports (requires reboot).

- install Kinect Developer Toolkit v1.7.0
- install Kinect Runtime v1.7
- install Kinect SDK v1.7

- To compile everything, use a newly-created "cmd" window (so that the changes to PATH and the environment variable are reflected) and enter these lines:

```
cd %PALETTESOURCE%\build\windows
build
```

- Install SenselApp0.19.32 (for the Morph)

- Other useful things to install are:  7zip, sharpkeys
- The result of this should be an installer executable in the release directory,
which you should execute to install Palette.

- If this is the first time you've run the Palette software, you should follow the instructions for one-time steps in
<a href="https://github.com/vizicist/palette/blob/main/doc/using_resolume.md">https://github.com/vizicist/palette/blob/main/doc/using_resolume.md</a>.

- After that, you're ready to start using the Palette, as described in 
<a href="https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md">https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md</a>

