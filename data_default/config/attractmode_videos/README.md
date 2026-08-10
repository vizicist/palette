# Attract mode videos

Place video files in this directory and they will play on the Resolume output
while Palette is in attract mode, instead of showing only the generated
graphics. The videos play in filename order and loop back to the first one.

This directory is created by the installer, but the videos in it are not part of
Palette - they are whatever suits the installation. Videos you put here are left
alone when you install a newer version of Palette.

**An empty directory does nothing.** This README on its own is not enough to
turn the feature on: attract mode only uses videos when there is at least one
actual video file here. Recognized extensions are `.mp4`, `.mov`, `.avi`,
`.mkv`, `.webm`, `.m4v`, `.mpg`, and `.mpeg`. Anything else in this directory,
including this file, is ignored.

## Turning it off

Videos play by default whenever this directory has something in it. To keep the
files here but stop them being used, turn off `global.attractvideos`:

```
palette set global.attractvideos false
```

The change takes effect immediately, even if attract mode is already running.

## Resolume setup

Palette loads these files into Resolume through Resolume's REST API, because the
OSC interface can only trigger clips that are already in the composition. That
API has to be switched on once, in Resolume under **Preferences > Webserver**.
If it is off, attract mode still works normally, the videos simply don't play,
and the engine log says so.

Related parameters:

| Parameter | Default | Meaning |
| --- | --- | --- |
| `global.attractvideos` | `true` | Whether videos in this directory are used |
| `global.attractvideolayer` | `6` | Resolume layer the videos play on |
| `global.resolumerestport` | `8080` | Port of Resolume's webserver |

Palette adds layers to the composition if the layer above doesn't exist yet.
Because the video layer sits above the four patch layers and the text layer, the
videos cover them while they play.

The audio of these clips is whatever Resolume does with it - Palette doesn't
mute or unmute them. If a video's soundtrack shouldn't be heard, turn the layer
volume down in Resolume.
