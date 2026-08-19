# Attract mode videos

Place video files in this directory and they will play while Palette is in
attract mode, instead of showing only the generated graphics. The videos play in
filename order and loop back to the first one.

By default they play on the Resolume output - the projection the audience sees.
They can play on the GUI screen instead; see **Where they play** below.

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

## Where they play

`global.attractvideodestination` chooses between two places:

| Value | Where the videos play |
| --- | --- |
| `main` (default) | The Resolume output, over the four patch layers |
| `gui` | The GUI screen, in the browser, in place of the attract screen |

```
palette set global.attractvideodestination gui
```

The change takes effect immediately, even mid-show: the videos come down from
wherever they were and go up at the new destination.

The `gui` destination doesn't involve Resolume at all - the engine serves this
directory over its own web server and the browser plays the files itself, so
none of the Resolume setup below applies to it. It needs nothing configured
beyond this parameter; in particular it does not need `global.attractallowgui`,
which governs showing the attract *screen* on the GUI. If this directory turns
out to have no videos in it, the GUI falls back to that attract screen.

By default a video that isn't the shape of the screen is fitted inside it, with
black bars. `global.attractvideoresize` changes the layout:

```
palette set global.attractvideoresize true
```

The video then fills the top two thirds of the screen - keeping its aspect
ratio and cropping whatever hangs over the edges, usually the left and right
sides - and the bottom third is given over to the venue logo, which is read
from `shapes/dirtygoat.svg`.

**The GUI destination is always silent.** The GUI screen is right next to
whoever walks up to the instrument, so its videos are muted regardless of what
soundtrack they carry. The `main` destination is the one that plays their audio
(see the note at the end about layer volume).

## Resolume setup

This section is about the `main` destination only.

Palette loads these files into Resolume through Resolume's REST API, because the
OSC interface can only trigger clips that are already in the composition. That
API has to be switched on once, in Resolume under **Preferences > Webserver**.
If it is off, attract mode still works normally, the videos simply don't play,
and the engine log says so.

Related parameters:

| Parameter | Default | Meaning |
| --- | --- | --- |
| `global.attractvideos` | `true` | Whether videos in this directory are used |
| `global.attractvideodestination` | `main` | `main` (Resolume) or `gui` (GUI screen) |
| `global.attractvideoresize` | `false` | GUI screen: fill the top 2/3, venue logo below |
| `global.attractvideolayer` | `6` | Resolume layer the videos play on |
| `global.resolumerestport` | `8080` | Port of Resolume's webserver |

Palette adds layers to the composition if the layer above doesn't exist yet.
Because the video layer sits above the four patch layers and the text layer, the
videos cover them while they play.

The audio of these clips is whatever Resolume does with it - Palette doesn't
mute or unmute them. (This applies to the `main` destination; the GUI one is
always muted.) If a video's soundtrack shouldn't be heard, turn the layer
volume down in Resolume.
