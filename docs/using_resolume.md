## Using Palette with Resolume 7 and the Sensel Morph

- These installation instructions will give you a visuals-only Palette.

- If you want to install both the visual and musical aspects of a full-blown
Space Palette Pro,
these are not the instructions for you.
Instead, use: <a href="https://github.com/vizicist/spacepalettepro/blob/main/doc/building_software.md">https://github.com/vizicist/spacepalettepro/blob/main/doc/building_software.md</a>

- These instructions assume that you:
  - are running Windows 10
  - have Resolume Avenue (or Arena) 7
  - and have a Sensel Morph.

## One-time installation steps

- First, download and execute the latest installer from the
<a href=https://github.com/vizicist/palette/tree/main/release>release directory</a>.

- If the installer asks you to reboot Windows, please do so.

- Start Resolume

- In Resolume's <i>Preferences->Video</i> section, add this directory to the list of FreeFrame (FFGL) plugin directories: <pre>C:\Program Files\Palette\ffgl</pre>

- In Resolume's <i>Preferences->OSC</i>, enable "OSC Input" with an incoming port of 7000.

- Use Resolume's <i>Composition->Open</i> to open: <pre>%LOCALAPPDATA%\Palette\config\PaletteABCD.avc</pre>
  That composition contains four layers, each with a <b>Palette</b> FFGL plugin followed by several dozen other FFGL effect plugins.
  Don't worry if there's a big yellow X in the layer, it should go away the next time Resolume starts.

- Quit Resolume.

- You're now ready to start using the Palette, as described in 
<a href="https://github.com/vizicist/spacepalettepro/blob/main/doc/starting_and_using.md">https://github.com/vizicist/spacepalettepro/blob/main/doc/starting_and_using.md</a>
