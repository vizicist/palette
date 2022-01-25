## Using Palette with Resolume 7 and the Sensel Morph

- These installation instructions will give you a visuals-only Palette.

- Installing the musical aspects of a full-blown Space Palette Pro is a much more
involved process involving lots of additional software and configuration.
If you want to attempt it, these are the instructions to use:
<a href="https://github.com/vizicist/spacepalettepro/blob/main/doc/building_software.md">https://github.com/vizicist/spacepalettepro/blob/main/doc/building_software.md</a>

- The instructions below assume that you:
  - are running Windows 10
  - have Resolume Avenue (or Arena) 7
  - and have a Sensel Morph.

## One-time installation steps

- Download and install the SenselApp (64 bit) from https://shop.sensel.com/pages/support#downloads

- Start the SenselApp, make sure the Morph is updated to the latest firmware, and then quit the SenselApp.

- Download and execute the latest Palette installer from the
<a href=https://github.com/vizicist/palette/tree/main/release>release directory</a>.

- If the Palette installer asks you to reboot Windows, please do so.

- Start Resolume

- In Resolume's <i>Preferences->Video</i> section, add this directory to the list of FreeFrame (FFGL) plugin directories: <pre>C:\Program Files\Palette\ffgl</pre>

- In Resolume's <i>Preferences->OSC</i>, enable "OSC Input" with an incoming port of 7000.

- Use Resolume's <i>Composition->Open</i> to open: <pre>%LOCALAPPDATA%\Palette\config\PaletteABCD.avc</pre>
  That composition contains four layers, each with a <b>Palette</b> FFGL plugin followed by several dozen other FFGL effect plugins.
  Don't worry if there's a big yellow X in the layer, it should go away the next time Resolume starts.

- Quit Resolume.

- You're now ready to start using the Palette, as described in 
<a href="https://github.com/vizicist/spacepalettepro/blob/main/doc/starting_and_using.md">https://github.com/vizicist/spacepalettepro/blob/main/doc/starting_and_using.md</a>
