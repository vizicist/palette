## Using Palette with Resolume 7 and the Sensel Morph

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

- Use Resolume's <i>Composition->Open</i> to open: <pre>C:\Program Files\Palette\config\PaletteA.avc</pre>
  That composition contains a single layer with a <b>Palette</b> plugin followed by several dozen FFGL plugins.
  Don't worry if there's a big yellow X in the layer, it should go away the next time Resolume starts.

- Quit Resolume.

## Every time you want to start the Palette

- Make sure the Sensel Morph is plugged in.

- Invoke <b>Start Palette and Resolume</b> found in the Palette folder of the Windows Start menu.
The Palette software will be started, along with Resolume.

- If you want to start the Palette software and Resolume independently, you can use <b>Start Palette</b>.
In that case, you'll need to manually activate the Palette clip inside Resolume.  This is done automatically
if you use <b>Start Palette and Resolume</b>.

- You should now be able to finger paint on the Morph and see results in the Resolume output.

- Use the Palette GUI to select different presets for your finger painting.
If you don't see the GUI, it may be hiding behind other windows (e.g. Resolume).

- To get more control and edit the presets, click on the Preset button at the very top of the GUI.
This will reveal the buttons to access separate pages for Snapshot, Visual, and Effect presets.
A Snapshot preset contains both visual and effect settings, and the Visual and Effect
presets let you control these things independently.
If you click a second time on any of these three buttons,
you will be able to access the individual parameters
of a preset and save it has a new preset.

- If it's not working, see below for debugging hints.

## Stopping the Palette

- Use <b>Stop Palette</b> or <b>Stop Palette and Resolume</b> from the Palette folder of the Windows Start menu.

## Debugging hints if you don't see any visual output

- Make sure that you've clicked on the <b>Palette</b>
cell (the first one) in the layer to activate it.
This is the most common mistake.
Every time you start Resolume manually, you need to do this.
If you want to eliminate the need for this, use can use this script: <pre>C:\Program Files\Palette\bin\palettestartall.bat</pre>
That script will start the Palette software and start Resolume.

- If for some reason Resolume crashes at startup,
you can look in this file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
for clues as to the reason.  If you can't resolve the issue,
you should either remove the ffgl directory from the <i>Preferences->Video</i> section or just uninstall Palette.

- To verify that the plugin is being recognized by Resolume,
you should find an entry for <b>Palette</b> in Resolume's Sources tab, in the alphabetical list under <b>Generators</b>.  If you don't see this, look at the Resolume log file in this directory: <pre>%APPDATA%\Resolume Avenue</pre>

- After you activate the plugin, this log file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
should contain this line at the end: <pre>Palette: listening for OSC on port 3334</pre>

- In this logfile: <pre>%LOCALAPPDATA%\Palette\logs\engine.log</pre>
you should see lines like these: <pre>2020/11/17 12:36:16 ====================== Palette Engine is starting
2020/11/17 12:36:17.030899 MIDI devices (18 inputs, 20 outputs) have been initialized
2020/11/17 12:36:17.031868 StartRealtime begins
2020/11/17 12:36:17.039870 NewReactor: pad=A resolumeLayer=1
2020/11/17 12:36:17.039870 NewReactor: pad=B resolumeLayer=2
2020/11/17 12:36:17.039870 NewReactor: pad=C resolumeLayer=3
2020/11/17 12:36:17.039870 NewReactor: pad=D resolumeLayer=4
2020/11/17 12:36:17.416867 Morph Opened and Started: idx=0 serial=SM01180216801 firmware=0.19.216 suceeded
2020/11/17 12:36:17.533556 StartNATS: Subscribing to palette.api
2020/11/17 12:36:17.533556 StartNATS: subscribing to palette.event
</pre>

- In this logfile: <pre>%LOCALAPPDATA%\Palette\logs\engine.log</pre>