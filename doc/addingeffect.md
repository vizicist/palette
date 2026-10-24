# Adding Resolume effects to Palette

To add an effect, install it in Resolume, add two instances to each Palette
clip, and register its controls in `paramdefs.json` and `resolume.json`. The
engine and web UI discover ordinary effect controls from those JSON files;
adding an effect normally needs no Go, JavaScript, or Palette FFGL build.

This guide describes the current implementation in
[kit/resolume.go](../kit/resolume.go),
[kit/params_defs.go](../kit/params_defs.go), and
[kit/patch.go](../kit/patch.go). It replaces the older instructions referring
to `PaletteABCD.avc`.

## Files and naming

| Item | Purpose |
| --- | --- |
| `data_default/config/PaletteDefault.avc` | Standard Resolume composition shipped in this repository |
| `data_default/config/PaletteDefaultSP.avc` | Alternate shipped composition with the same A–D layer layout |
| `data_default/config/PaletteKora.avc` | Legacy alternate layout; see the routing limitation below |
| `data_default/config/paramdefs.json` | Palette controls, types, initial values, and randomization settings |
| `data_default/config/resolume.json` | Effect bypass messages and parameter OSC address suffixes |
| `data_default/config/paramenums.json` | Only needed if you introduce a string control with a named list of choices |
| `data_default/saved/effect/` | Optional saved effect presets |

Source files and installed runtime files can be different copies. The engine
reads `ConfigDir()`, which is `PaletteDataPath()/config`:

- `PALETTE_DATA` selects `data_<name>` and defaults to `data_default`.
- `PALETTE_DATAROOT`, if set, selects the parent of that data directory.
- On Windows, the usual runtime config is
  `%LOCALAPPDATA%\Palette\data_default\config`.
- On other platforms, the default is `data_default/config` under the detected
  Palette root. See [macOS setup](building_macos.md).

Keep the repository files as the distributable source, then copy the changed
files to the installation you are testing. Back up its composition and config
before replacing them. Do not assume changing this checkout changes a running
Windows installation.

An effect such as Goo is defined once as `effect.goo` and
`effect.goo:speed`. `LoadParamDefs()` expands those definitions at startup:

| Definition in `paramdefs.json` | Runtime names used by the UI, API, and saved presets |
| --- | --- |
| `effect.goo` | `effect.1-goo`, `effect.2-goo` |
| `effect.goo:speed` | `effect.1-goo:speed`, `effect.2-goo:speed` |

Write the unnumbered definitions only. The two runtime controls target the
first and second instances in the clip's effect chain.

## Included example: Invert RGB

Resolume's built-in **Invert RGB** effect is registered in `paramdefs.json`
and `resolume.json`. `PaletteDefault.avc` and `PaletteDefaultSP.avc` include
two instances on clip 1 of each A/B/C/D layer, after Infinite Zoom in each
effect group. No third-party plugin installation or binary rebuild is needed.
The legacy `PaletteKora.avc` layout is not updated; see its routing limitation
below.

The native effect class is `InvertRGB` (`A105`); its OSC identifiers are
`invertrgb` and `invertrgb2`. Each instance exposes these Palette controls:

| Definition | Initial value | OSC suffix for instance 1 |
| --- | --- | --- |
| `effect.invertrgb` | `false` (bypassed) | `/invertrgb/bypassed` (0 enables, 1 bypasses) |
| `effect.invertrgb:opacity` | `1.0` | `/invertrgb/opacity` |
| `effect.invertrgb:r` | `1.0` | `/invertrgb/effect/r` |
| `effect.invertrgb:g` | `1.0` | `/invertrgb/effect/g` |
| `effect.invertrgb:b` | `1.0` | `/invertrgb/effect/b` |

The numeric controls are floats in the range 0–1. Opacity belongs to the
outer effect, so its address does **not** include `/effect/`. Alpha inversion
is left at Resolume's default of zero. The toggle and controls have no Rand
fields, so randomization leaves them unchanged. Both numbered instances are
generated automatically, just like Goo above.

After deploying the JSON files and composition and restarting the engine,
test A with `palette patchset A effect.1-invertrgb true`, then restore bypass
with `palette patchset A effect.1-invertrgb false`. Use `effect.2-invertrgb`
for the second instance. Bypass, opacity, and all three RGB channels have been
checked through OSC against Resolume's REST readback on all eight instances.

Test the two instances separately. With full inversion settings and opacity,
two inversions can cancel each other, making working controls appear to have
no effect. Keep the second instance bypassed while checking the first.

## 1. Install the effect and prepare the composition

For a built-in effect, no plugin installation is needed. For a third-party
FFGL effect, install the vendor's build for your Resolume version and CPU
architecture. Resolume supports `.dll` plugins on Windows and `.bundle`
plugins on macOS. Put them in its user **Extra Effects** directory or add their
folder under **Preferences > Video > FFGL Directories**, then restart Resolume.
See Resolume's [effects](https://resolume.com/support/en/effects) and
[plugin directory preferences](https://resolume.com/support/en/preferences).

Open the composition the installation actually uses. The standard setup opens
`PaletteDefault.avc`; the engine's Resolume process launcher does not select a
composition filename for you.

1. Select **clip 1 in layer A** and its clip video effects chain. Add the new
   effect after the Palette generator, in the first alphabetical group of
   effects. Add a second instance in the second alphabetical group. Preserve
   the order of the existing effects: processing order changes the picture.
2. Repeat for **clip 1 in B, C, and D**: eight new instances in total. These are
   clip effects, not effects on the layer or composition. Leave the text and
   attract-video layers alone.
3. Initially bypass both instances. Keep their settings predictable, including
   any controls you are not exposing in Palette.
4. Save the composition in Resolume and copy that saved file back to the
   appropriate repository `data_default/config/*.avc` and runtime config
   location. Explicitly open the saved copy to verify it contains the changes.

The routing is fixed: A/B/C/D are Resolume layers 1/2/3/4, and effect commands
always target clip 1. Renaming a layer does not change those numbers. Update
other supported compositions only if they also need the new effect; JSON
registration alone cannot insert plugins into a composition.

`PaletteKora.avc` is a special case: its Palette clips are in layers 2, 4, 6,
and 8, whereas the current Go routing is fixed at 1, 2, 3, and 4. Do not use it
as a drop-in template for this procedure without first reconciling that layout
and routing. `PaletteDefault.avc` and `PaletteDefaultSP.avc` match the current
routing.

## 2. Record the exact OSC addresses and types

Enable **Preferences > OSC > OSC Input** on port **7000**. Palette currently
sends effect OSC to `127.0.0.1:7000`; the top-level `host` and `port` fields in
`resolume.json` are not used to configure that client.

Use **Shortcuts > Edit OSC**, then click bypass and each control you want to
expose. Record the absolute clip address, type tag, and range, for both
instances. Resolume documents this inspection workflow in its
[OSC guide](https://resolume.com/support/en/osc).

For the existing Goo effect on A, the expected form is:

```text
/composition/layers/1/clips/1/video/effects/goo/bypassed
/composition/layers/1/clips/1/video/effects/goo/effect/speed
/composition/layers/1/clips/1/video/effects/goo2/bypassed
/composition/layers/1/clips/1/video/effects/goo2/effect/speed
```

Use the OSC identifier reported by Resolume, not a guess from its display name.
Palette's effect key must match that identifier, and the second instance must
be addressable by appending `2`. For a new key `neweffect`, the implementation
expects `neweffect` and `neweffect2`. If a plugin uses a different naming
pattern, the current routing code needs an extension; JSON cannot describe two
independent instance identifiers.

For each full address, remove this prefix before putting it in the JSON:

```text
/composition/layers/1/clips/1/video/effects
```

Keep the remaining leading slash. For example, store
`/goo/effect/speed`. Palette adds the layer/clip prefix and changes `goo` to
`goo2` for instance 2. Do not store a full address, a numeric effect index, or a
`selectedlayer`/`selectedclip` address.

## 3. Add the Palette parameter definitions

Use the existing Goo definitions as a working template. The following is a
valid standalone JSON example; insert its entries into the existing top-level
object in `paramdefs.json`, replacing the effect and control names with the
ones you recorded. Goo is already registered, so do not add duplicate keys.

```json
{
  "effect.goo": {
    "valuetype": "bool",
    "min": "false",
    "max": "true",
    "randmax": "0.2",
    "init": "false",
    "comment": "# Enable Goo"
  },
  "effect.goo:resolution": {
    "valuetype": "int",
    "min": "4",
    "max": "128",
    "randmin": "4",
    "randmax": "32",
    "init": "16",
    "comment": "# Goo resolution"
  },
  "effect.goo:speed": {
    "valuetype": "float",
    "min": "0.0",
    "max": "1.0",
    "randmin": "0.0",
    "randmax": "0.25",
    "init": "0.15",
    "comment": "# Goo speed"
  }
}
```

All definition values, including numbers and booleans, are JSON **strings**.
Use a boolean parent toggle without a colon, then `effect.<key>:<control>` for
each exposed control. Start with `init: "false"` for the parent so older
presets do not unexpectedly enable the new effect.

Choose the type from the OSC input, not the units displayed on the knob:

| `valuetype` | Message sent by Palette |
| --- | --- |
| `float` | One OSC `float32`, often normalized to 0–1 |
| `int` | One OSC `int32`, for controls accepting integer values |
| `bool` | One OSC `int32`: 1 for true, 0 for false |
| `string` | One OSC string; named choices come from `paramenums.json` |

For a string parameter, `min` names the enum in `paramenums.json`; follow an
existing string definition and keep the other required fields. Resolume's
normalized range can differ from its displayed units. Palette sends values
as-is; setting `min` and `max` does not add unit conversion. Packed OSC colors,
multi-argument messages, or other special protocols need engine changes unless
the plugin exposes compatible individual controls.

Randomization is opt-in. For numeric controls, add `randmin`/`randmax` to choose
an appropriate range. For the parent boolean, `randmax: "0.2"` gives each
instance a 20% enable probability when randomized. Without random fields the
control is skipped by Rand. Start conservatively and test the two instances
together.

## 4. Add the Resolume mapping

Add one entry under the existing `effects` object in `resolume.json`. This is
the matching Goo subset, again intended as a template rather than an addition
to the already-existing Goo entry:

```json
{
  "goo": {
    "on": { "addr": "/goo/bypassed", "arg": 0 },
    "off": { "addr": "/goo/bypassed", "arg": 1 },
    "params": {
      "resolution": "/goo/effect/resolution",
      "speed": "/goo/effect/speed"
    }
  }
}
```

The effect key must match the key after `effect.` in the definitions. Each
`params` key must match the name after the colon. Address suffixes must match
what you inspected in Resolume, including punctuation and nested paths. For
example, the existing HueRotate control `satscale` maps to
`/huerotate/effect/sat.scale`.

The `on`/`off` arguments here are JSON **numbers**, not strings. Bypass 0 means
the effect is enabled; bypass 1 means disabled. Both messages are sent as OSC
integers. You may expose only bypass initially, omitting `params`, then add
controls after the routing works.

## 5. Deploy and verify

1. Check both edited files for valid JSON and duplicate keys. From the
   repository root, `python3 -m json.tool data_default/config/paramdefs.json`
   and `python3 -m json.tool data_default/config/resolume.json` check syntax,
   but do not catch duplicate keys. Review the additions for those separately.
2. Copy both JSON files and the modified composition to the active runtime
   installation. `scripts/updateconfig.bat` copies only `param*.json`; it does
   **not** copy `resolume.json` or the composition, so it is insufficient by
   itself.
3. Open the modified composition in Resolume and restart the Palette engine.
   Definitions are loaded at initialization, and the Resolume mapping is cached
   in its singleton client; a browser refresh alone does not reload them.
4. In the web UI, select A, open **Effect**, and click the tab again for
   **Parameters**. Both numbered toggles should appear automatically. Enable
   one to reveal its controls; controls of disabled effects are hidden.
5. Test one instance at a time on A, then repeat on B/C/D. In Resolume, confirm
   the expected bypass button and parameter actually change. API success only
   confirms that Palette accepted the value; OSC provides no effect-applied
   acknowledgement here.
6. Save an Effect preset, reload it, and test Init and Rand. Check an older quad
   preset as well: missing new quad values are initialized from the definitions.
   Check `paramoverrides.json` if loaded values differ from expectations.

For a known existing effect, these CLI commands exercise the same route that
new controls will use. Substitute your effect/control names and suitable values:

```sh
palette patchget A effect.1-goo
palette patchset A effect.1-goo:speed 0.15
palette patchset A effect.1-goo true
palette patchset A effect.1-goo false
palette patchset A effect.2-goo true
palette patchset A effect.2-goo false
```

`palette patchset ABCD ...` is not the all-patch syntax; use `palette patchset
"*" ...` when intentionally changing all four. Setting patch values also saves
the current quad, so use a test setup or save your working preset first.

If nothing changes, inspect Resolume's OSC monitor and Palette's `engine.log`.
Use `palette set global.log osc,params` to enable relevant tracing, then restore
the previous logging setting afterward. Common causes are the wrong runtime
config copy, a missing second instance, an effect placed on the layer instead
of clip 1, a wrong address/type, or OSC Input being disabled. A missing
`1-`/`2-` runtime prefix produces `no dash in effect`; missing mappings produce
`no effects value` or `No params value` messages.

## Builds and delivery

No parameter generation or binary rebuild is required for normal `effect.*`
additions. `python/generateparams.py` writes the C++ `LayerParams_*.h` files for
`visual.*`/`sprite.*`; it does not generate effect mappings. Neither `ffgl.json`
nor the Palette generator's C++ parameter declarations register these effects.

If an unusual plugin requires routing/type changes in Go, rebuild from
`cmd/palette_engine` with `go build .`. If you change `kit/webui/`, rebuild the
engine as well because the UI is embedded. Changes to the Palette generator
itself require its separate FFGL build.

For delivery, include the changed JSON files, each supported composition, and
any new saved presets. Install third-party plugins on every target machine
using the vendor's distribution terms. Keep an unmodified composition/config
backup for rollback; removing JSON controls alone does not remove plugin
instances saved in Resolume.
