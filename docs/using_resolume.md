## Using Palette with Resolume 7 and the Sensel Morph

- These instructions will give you a visuals-only Palette.

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

- Use Resolume's <i>Composition->Open</i> to open: <pre>%LOCALAPPDATA%\Palette\config\PaletteA.avc</pre>
  That composition contains a single layer with a <b>Palette</b> plugin followed by several dozen FFGL plugins.
  Don't worry if there's a big yellow X in the layer, it should go away the next time Resolume starts.

- Quit Resolume.

## Starting the Palette

- Make sure the Sensel Morph is plugged in, and that Resolume is not currently running.

- Invoke <b>Start Palette (small GUI)</b> found in the Palette folder of the Windows Start menu.  This will start up the Palette software, including Resolume.  The smaller-than-normal GUI is appropriate when you only have one Morph.

- You should now be able to finger paint on the Morph and see results in the Resolume output.

- Use the Palette GUI to select different presets for your finger painting.
If you don't see the GUI, it may be hiding behind other windows (e.g. Resolume).

- Now, have fun finger painting and exploring the presets.  Take advantage of the fact
that finger pressure on the Morph affects the size of the visuals you are painting.
The Morph is extremely sensitive with a large dynamic range.

- If it's not working, see the section below on debugging hints.

## Looping

- Looping operations are only visible in the more advanced modes of the GUI.

- The buttons at the bottom of the GUI let you turn on Looping, which will loop your
gestures on the Morph.  You can control the length of the loops and how quickly they fade out.
The effects of the looping controls are per-pad, depending (like parameter changes) on which pads are currently highlighted in the GUI.

- Beware, if you leave looping on, you may get confused about what's going on, since you'll be seeing
the results of your live gestures as well as the looped gestures.

## Editing the presets

- To get more control and edit the presets, click on the <b>Preset</b> button
at the very top of the GUI.
This will reveal the buttons to access separate pages for Snapshot, Visual, and Effect presets.
A Snapshot preset contains both visual and effect settings, and the Visual and Effect
presets let you control these things independently.

- Clicking the buttons at the top of the page for Snapshot, Visual, and Effect will toggle
between 1) seeing the page of named presets and 2) seeing the list of parameters which you can 
then edit.  When editing, you'll also see buttons that allow you to save it, either with the same
name or as a newly named preset.  Any presets that you edit (newly named or not) will be saved
in a local directory that will (should) not be overwritten by newer versions of Palette.

- When editing parameters, the <b>Rnd</b> button will randomize many of the parameters.
When using this on Visual parameters, it can be useful to set the Effect preset to <b>None</b>,
so you can clearly see what's being created.  Likewise, when randomizing the Effect parameters,
make sure the Visual settings are showing something clearly visible.  It will sometimes be the
case that randomized parameters will produce no output.

## Stopping the Palette

- Use <b>Stop Palette</b> or <b>Stop Palette and Resolume</b> from the Palette folder of the Windows Start menu.

## Debugging hints

- If you start Resolume manually, make sure that you've clicked on the <b>Palette</b>
cell (the first one) in the Resolume layer to activate it. Every time you start Resolume manually, you need to do this.
Using <b>Start Palette and Resolume</b> eliminates the need for this, and is equivalent to invoking this script: <pre>C:\Program Files\Palette\bin\palettestartall.bat</pre>

- To verify that the plugin is being recognized by Resolume,
you should find an entry for <b>Palette</b> in Resolume's Sources tab, in the alphabetical list under <b>Generators</b>.
If you don't see this, look for clues in the Resolume log file in this directory: <pre>%APPDATA%\Resolume Avenue</pre>

- If for some reason Resolume crashes at startup,
you can look in this file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
for clues as to the reason.  If you can't resolve the issue,
you should either remove the ffgl directory from Resolume's <i>Preferences->Video</i> section or just uninstall Palette.

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