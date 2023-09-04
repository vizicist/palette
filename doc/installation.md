## Installing Palette

These are the instructions for installing a binary release of the Palette system.  If instead you want to rebuild it from the source code, see
<a href="https://github.com/vizicist/spacepalette/blob/main/doc/building.md">these instructions</a>.

The instructions below assume that you have:
  - Windows 10 or later
  - Resolume Avenue 7 or later (~$300)
  - Plogue Bidule (~$100), see https://www.plogue.com/products/bidule.html
  - LoopBe30 (~$20), see from https://www.nerds.de/en/order.html
  - at least one Sensel Morph, or some other 3D cursor-providing device or program

## One-time installation steps

- Install git from https://gitforwindows.org

- Use System Properties->Environment Variables to add `C:\Program Files\Git\bin` and `C:\Program Files\Git\usr\bin` to your Path variable.  You will need to log out (or reboot) to have this change reflected.

- All VST plugins should be put in `%CommonProgramFiles%\VST2` (for VST 2 plugins) and `%CommonProgramFiles%\VST3` (for VST 3 plugins).

 - Install the Dexed VST synth (free) from https://github.com/asb2m10/dexed . 

 - Install the Vital VST synth (free Basic version) from https://vital.audio

- After installing and starting Bidule, use Edit->Preferences->Plugins to set the VST plugin directory to
`%CommonProgramFiles%\VST2`.

- Install LoopBe30 from https://www.nerds.de/en/order.html
  - After installing, use the LoopBe30 tray item to expand "ports after reboot" to 16
  - Turn off "Enable Shortcut Detection"
  - Reboot (so that the 16 ports are recognized)
 <p>

- Start Bidule, and use this menu item - Tools->Osc Server - to enable OSC .

- Download and install the SenselApp (64 bit) from https://shop.sensel.com/pages/support#downloads

- Start the SenselApp, make sure the Morph is updated to the latest firmware, and then quit the SenselApp.

- Download and execute the latest Palette installer from the
<a href=https://github.com/vizicist/palette/tree/main/release>release directory</a>.

- If the Palette installer asks you to reboot Windows, please do so.

- Start Resolume, and
  - In the <i>Preferences->OSC</i> section, enable OSC Input on port 7000
  - In the <i>Preferences->Video</i> section, add this directory to the list of FreeFrame (FFGL) plugin directories: `C:\Program Files\Palette\ffgl`
<p>

- Quit and restart Resolume.
  - Use Resolume's <i>Composition->Open</i> to open: `%CommonProgramFiles%\Palette\data\config\PaletteDefault.avc`<br>
  That composition contains four layers, each with a <b>Palette</b> FFGL plugin followed by several dozen other FFGL effect plugins.
  If there's a big yellow X in the layer, Resolume hasn't been able to find or load the Palette plugin.  Verify that the Palette plugin is seen in Effects under Generators.
<p>

- Quit Resolume.

- You're now ready to start using the Palette, as described
<a href="https://github.com/vizicist/palette/blob/main/doc/starting_and_using.md">here</a>

- If there are any issues, log files can be found in `%CommonProgramFiles%\Palette\logs`,
with `engine.log` being the most important one.  Feel free to email me@timthompson with questions or suggestions for improving this documentation.