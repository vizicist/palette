## Initializing a Palette build and development environment.

- Please send email to me@timthompson.com if these instructions don't work for you.

- When given a choice between 64-bit and 32-bin installations, choose 64-bit.

- Install git from https://gitforwindows.org

- Install Go version 1.15 or later from https://golang.org/dl/

- Install Python 3.8.6 or later (BUT NOT FROM THE WINDOWS STORE!), and select the option for adding it to your PATH.

- Install Visual Studio Code from https://code.visualstudio.com/download

- Install Inno Setup from https://jrsoftware.org/isinfo.php

- Install Visual Studio 2019 Community Edition from https://visualstudio.microsoft.com/downloads

	- in the Workload tab, select "Desktop Development with C++"
	- In the Individual components tab, select "Msbuild"

- Install mingw64 using this installer:

 https://sourceforge.net/projects/mingw-w64/files/Toolchains%20targetting%20Win32/Personal%20Builds/mingw-builds/installer/mingw-w64-install.exe


```
 *** USE THESE SETTINGS WHEN INSTALLING mingw64 ***
 Version: 8.1.0
 Architecture: x86_64
 Threads: posix
 Exception: sjlj
```

- Open a new "cmd" window (so that changes to environment variables from the installed packages are reflected) and clone the repos by entering these lines

```
mkdir %USERPROFILE%\go\src\github.com\vizicist
cd %USERPROFILE%\go\src\github.com\vizicist
git clone https://github.com/vizicist/palette.git
git clone https://github.com/vizicist/portmidi.git
```

- Install a few Go packages by entering these lines in the "cmd" window:

```
go get github.com/hypebeast/go-osc/osc
go get github.com/nats-io/nats-server/server
go get github.com/nats-io/nats.go
go get github.com/nats-io/nuid
go get gopkg.in/mail.v2
```

- Make sure these directories are added to your PATH variable in System Properties.

```
 C:\Program Files\Git\bin
 C:\Program Files\Git\usr\bin
 C:\Program Files\mingw-w64\x86_64-8.1.0-posix-sjlj-rt_v6-rev0\mingw64\bin
 %USERPROFILE%\go\bin
 %USERPROFILE%\go\src\github.com\vizicist\palette\scripts
```

- Add a new environment variable in your System Properties:

```
 PALETTESOURCE=%USERPROFILE%\go\src\github.com\vizicist\palette
```

- In Visual Studio Code, click on the "extensions marketplace" icon (four little squares) on the left side.
  In the "Search Extensions" field, enter "go", and install the Go language support.
  Other pieces of the Go toolchain will be offered to you automatically within VSCode.

- To compile everything, use a newly-created "cmd" window (so that the changes to PATH and the environment variable are reflected) and enter these lines:

```
cd %PALETTESOURCE%\build\windows
build
```

- The result of this should be an installer executable in the release directory,
which you should execute to install Palette.
After that, the Windows Start menu should have a Palette folder under P,
where you'll find entries for "Start Palette" and "Stop Palette".
If you select "Start Palette", it will start palette_engine.exe and palette_gui.exe,
the later of which should pop up a GUI window with lots of buttons.
Congratulations, you've now compiled and installed Palette from scratch.
