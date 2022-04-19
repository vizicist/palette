## Initializing a Palette build and development environment.

NOTE: these instructions are only appropriate if you are wanting
to recompile the Palette software from scratch.
If instead you just want to install it using one of the
released installation packages, which is certainly a lot simpler and is recommended, use:
<a href="https://github.com/vizicist/palette/blob/main/doc/using_resolume.md">https://github.com/vizicist/palette/blob/main/doc/using_resolume.md</a>

- Please send email to me@timthompson.com if these instructions don't work for you.

- When given a choice between 64-bit and 32-bin installations, choose 64-bit.

- Install git from https://gitforwindows.org

- Install Go version 1.15 or later from https://golang.org/dl/

- Install Python 3.9.6 (64-bit) or later (BUT NOT FROM THE WINDOWS STORE!), and select the option for adding it to your PATH.

- Install Visual Studio Code from https://code.visualstudio.com/download

- Install Inno Setup from https://jrsoftware.org/isinfo.php

- Install Visual Studio 2013 Community Edition from https://visualstudio.microsoft.com/downloads

  - This will require downloading and mounting a .iso DVD image.
  - This version is required to get the v120 compiler tools for building older things,
  even though Visual Studio 2017 will be used to actually build things.

- Install Visual Studio 2017 Community Edition from https://visualstudio.microsoft.com/downloads

	- in the Workload tab, select "Desktop Development with C++"
	- In the Individual components tab, select "Msbuild"

- Install mingw64 to get the gcc compiler.  The online installers don't always work.  You'll want the 8.1.0 version, x86_64 architecture, posix threads, and sjlj exceptions.  Email me if you have any trouble getting it.

- Open a new "cmd" window (so that changes to environment variables from the installed packages are reflected) and clone the repos by entering these lines

```
mkdir %USERPROFILE%\Documents\Github
cd %USERPROFILE%\Documents\Github
git clone https://github.com/vizicist/palette.git
git clone https://github.com/vizicist/portmidi.git
```

- Make sure these directories are added to your PATH variable in System Properties.  If your gcc.exe is somewhere other than C:\Program Files\mingw64\bin, adjust that path.

```
 C:\Program Files\Git\bin
 C:\Program Files\Git\usr\bin
 C:\Program Files\mingw64\bin
```

Not sure whether %USERPROFILE%\Documents\Github\palette\SenselLib\x64 is needed, it depends on how you execute things during development.

- Add a new environment variable in your System Properties:

```
 PALETTESOURCE=%USERPROFILE%\Documents\Github\palette
```

- In Visual Studio Code, click on the "extensions marketplace" icon (four little squares) on the left side.
  In the "Search Extensions" field, enter "go", and install the Go language support.
  Also install Python language support.
  Other pieces of the Go toolchain will be offered to you automatically within VSCode.

- To compile everything, use a newly-created "cmd" window (so that the changes to PATH and the environment variable are reflected) and enter these lines:

```
cd %PALETTESOURCE%\build\windows
build
```

- The result of this should be an installer executable in the release directory,
which you should execute to install Palette.

- If this is the first time you've run the Palette software, you should follow the instructions for one-time steps in
<a href="https://github.com/vizicist/palette/blob/main/doc/using_resolume.md">https://github.com/vizicist/palette/blob/main/doc/using_resolume.md</a>.

- After that, you're ready to start using the Palette, as described in 
<a href="https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md">https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md</a>

