## Using the Palette

- These instructions assume you have already installed and configured the Palette software.

## Starting Up

- Make sure the Sensel Morph is plugged in, and that Resolume is not currently running.

- Invoke <b>Start Palette</b> found in the Palette folder of the Windows Start menu.  This will start up a few things: the Palette engine that runs in the background, the Palette GUI that you use to select and edit presets, and Resolume.  If you've installed Plogue Bidule for doing music, Bidule will also be started.

- If you have an actual touchscreen (as in the Space Palette Pro controller), you can set the guisize value in %LOCALAPPDATA%\Palette\config\settings.json to have the GUI show up on that touchscreen.

## Basic Usage

- Finger paint on the Morph and you should see results immediately in the Resolume output.

- Use the Palette GUI to select different presets for your
finger painting.  If you don't see the GUI, it may be hiding behind other windows (e.g. Resolume).

- Have fun finger painting and exploring the presets.  Take advantage of the fact
that finger pressure on the Morph affects the size of the visuals you are painting.
The Morph is extremely sensitive with a large dynamic range.

- If it's not working, see the section below on debugging hints.

## Using Four Morphs - Virtual or Real

- If you are using only a single Morph, touching the four corners of the Morph
allows you to switch between four virtual Morphs.
If you have 4 Morphs (e.g. a real Space Palette Pro controller),
you don't need to switch - you have simultaneous access to all four.

- Each of the four Morphs (real or virtual) can have different visual and sound settings.

- In the default "casual instrument" mode, each preset changes the settings of all four Morphs simultaneously.

## Advanced Mode

- There are 2 modes - casual and advanced.  You can switch between them by pressing the "Clear" button (at the bottom of the Palette GUI) four times in a row, quickly.
In the Advanced Mode, you have more per-pad control and can edit the presets as described below.

## Looping

- Looping operations are only visible in the more advanced modes of the GUI.

- The buttons at the bottom of the GUI let you turn on Looping, which will loop your
gestures on the Morph.  You can control the length of the loops and how quickly they fade out.
The effects of the looping controls are per-pad, depending (like parameter changes) on which pads are currently highlighted in the GUI.

- Beware, if you leave looping on, you may get confused about what's going on, since you'll be seeing
the results of your live gestures as well as the looped gestures.

## Editing the presets

- In the Advanced mode, you will see buttons at the top of the GUI that let you access
separate pages for Pad, Sound, Visual, and Effect presets.
A Pad preset combines the Sound, Visual and Effect settings.

- Clicking the buttons at the top of the page for Pad, Sound, Visual, and Effect will toggle
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

- Use <b>Stop Palette</b> from the Palette folder of the Windows Start menu.

## Debugging hints

- To verify that the Palette FFGL plugin is being recognized and loaded by Resolume,
you should find an entry for <b>Palette</b> in Resolume's Sources tab, in the alphabetical list under <b>Generators</b>.
If you don't see this, look for clues in the Resolume log file in this directory: <pre>%APPDATA%\Resolume Avenue</pre>

- If for some reason Resolume crashes at startup,
you can look in this file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
for clues as to the reason.  If you can't resolve the issue,
you should either remove the ffgl directory from Resolume's <i>Preferences->Video</i> section or just uninstall Palette.

- When the Palette FFGL plugin is properly activated, this log file: <pre>%LOCALAPPDATA%\Palette\logs\ffgl.log</pre>
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