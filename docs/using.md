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
