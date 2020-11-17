## Configuring Resolume 6 to use the Palette software on Windows.

- In Resolume's Preferences->Video, add this to the list of FreeFrame (FFGL) plugin directories: <pre>C:\Program Files\Palette\ffgl</pre>

- In Preferences->OSC, enable "OSC Input" with an incoming port of 7000.

- Quit and restart Resolume.

- If for some reason Resolume crashes, indicating something amiss in the
Palette FFGL plugin, you can look in %LOCALAPPDATA%\Palette\logs\ffgl.log
to look for clues.  If you can't resolve the issue, you should uninstall Palette.

- In Resolume's Sources tab, under Generators, you should now find entries for:

  - Palette_1
  - Palette_2
  - Palette_3
  - Palette_4

- Use Composition->Open to open: <pre>C:\Program Files\Palette\config\Palette_1_Layer.avc</pre>

- You should find a single layer with a <b>Palette_1</b> plugin followed by a dozen or more other FFGL plugins.

- Click on the Palette_1 cell (the first one) in the layer to activate it.

- If the Palette FFGL is working okay, the %LOCALAPPDATA%\Palette\logs\ffgl.log should contain this line at the end: <pre>Palette: listening for OSC on port 3334</pre>

- You can now start the Palette software by invoking the "Start Palette" entry in the Palette folder of the Windows Start menu.  Alternatively, you can open a cmd window and execute "palettestart".  You should then see a window pop up with a GUI for selecting Palette presets.

- If you have a Sensel Morph plugged in, you should be able to finger paint on the Morph and see something visual in the Resolume output.

- If you don't see anything in Resolume, take a look at this logfile: <pre>%LOCALAPPDATA%\Palette\logs\engine.log</pre> where you should see lines like this: <pre>2020/11/17 12:36:16 ====================== Palette Engine is starting
2020/11/17 12:36:16 Merging settings from C:\Users\tjt\AppData\Local\Palette\config\settings.json
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

