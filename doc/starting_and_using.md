## Using the Palette

- These instructions assume you have already <a href=installation.md>installed</a> the Palette software.

## Starting Up

- Make sure the Sensel Morph is plugged in, and that Resolume is not currently running.

- In a cmd window, execute: `palette start`

    This will start the Palette engine that runs in the background.  If configured appropriately, other programs will be automatically started - Bidule,
    Resolume, and the Palette GUI that you use to select and edit presets.

    The default configuration should automatically start Bidule, Resolume, and the Palette GUI.  If there are any issues, look at engine.log file in the
    `%CommonProgramFiles%\Palette\logs` directory.

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

- There are 2 modes - casual and advanced.  You can toggle between them in the GUI by first pressing the Help button to display the help page, and then sweeping (with the mouse or your finger) diagonally across the help page from one corner to the other.
In the Advanced Mode, you can select individual pads to enable per-pad control, and you can also edit the presets.

- In the advanced mode, clicking on the top category headers (Engine, Misc, Sound, Visual, Effect) will toggle between showing the presets of that category and showing and letting you edit the values of individual parameters.

## Looping

- The casual GUI interface allows looping to be turned on and off.  More detailed control of looping is only visible in the advanced mode of the GUI.  For example, engine parameters like engine.looping_fade let you control the looping behaviour, and the per-pad control in the advanced mode also applies to looping.

- Beware, if you leave looping on, you may get confused about what's going on, since you'll be seeing
the results of your live gestures as well as the looped gestures.

## Editing the presets

- In the Advanced mode, you will see buttons at the top of the GUI that let you access
separate category pages for Pad, Sound, Visual, and Effect presets.
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

- Execute `palette stop` to stop everything (including Resolume and Bidule).

## Configuring

- The Config directory - `%CommonProgramFiles%\Palette\data\config` -
contains various files you can edit to control things.

## Debugging hints

- To verify that the Palette FFGL plugin is being recognized and loaded by Resolume,
you should find an entry for <b>Palette</b> in Resolume's Sources tab, in the alphabetical list under <b>Generators</b>.
If you don't see this, look for clues in the Resolume log file in this directory: <pre>%APPDATA%\Resolume Avenue</pre>

- If for some reason Resolume crashes at startup,
you can look in the file `%CommonProgramFiles%\Palette\logs\ffgl.log`
for clues as to the reason.  If you can't resolve the issue,
you should either remove the ffgl directory from Resolume's <i>Preferences->Video</i> section or just uninstall Palette.

- When the Palette FFGL plugin is properly activated, this log file: `%CommonProgramFiles%\Palette\logs\ffgl.log`
should contain this line at the end: <pre>Palette: listening for OSC on port 3334</pre>

- In this logfile: `%CommonProgramFiles%\Palette\logs\engine.log`
you should see lines that indicate what has happened and/or failed during startup.