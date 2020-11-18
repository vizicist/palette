## Using Palette with Resolume 6 and the Sensel Morph

- These instructions assume that you are on Windows, have a licensed copy of Resolume, and have a Sensel Morph.

- Download and execute the latest installer from the
<a href=https://github.com/vizicist/palette/tree/main/release>release directory</a>.

- In Resolume's <i>Preferences->Video</i> section, add this to the list of FreeFrame (FFGL) plugin directories: <pre>C:\Program Files\Palette\ffgl</pre>

- In <i>Preferences->OSC</i>, enable "OSC Input" with an incoming port of 7000.

- Quit and restart Resolume.

- Use Composition->Open to open: <pre>C:\Program Files\Palette\config\Palette_1_Layer.avc</pre>

- That composition contains a single layer with a <b>Palette_1</b> plugin followed by a dozen or more other FFGL plugins.

- Click on the Palette_1 cell (the first one) in the layer to activate it.

- Make sure the Sensel Morph is plugged in.

- Start the Palette software by invoking
"Start Palette" in the Palette folder of the Windows Start menu.
Alternatively, you can open a cmd window and execute "palettestart".  You should see a window pop up with a GUI for selecting Palette presets.

- You should now be able to finger paint on the Morph and see something visual in the Resolume output.

- Use the GUI to select different Snapshot presets.  The Sound, Visual, and Effect tabs at the top of the GUI let you independently select presets for those things.

- If it's not working, see below for debugging hints.
## Debugging hints if you don't see any visual output

- If for some reason Resolume crashes at startup,
you can look in this file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
for clues as to the reason.  If you can't resolve the issue, you should uninstall Palette.

- To verify that the plugin is being recognized by Resolume,
you should find these entries in Resolume's Sources tab, under Generators:

  - Palette_1
  - Palette_2
  - Palette_3
  - Palette_4

- After you activate the plugin (by clicking on its cell in the layer),
working okay, this log file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
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

