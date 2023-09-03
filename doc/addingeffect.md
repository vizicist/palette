## How to add a new FFGL effect to the Palette

- Add the effect to the PaletteABCD.avc composition in Resolume.  It should be added eight times in total.  Within each of the four layers - A,B,C,D - the new effect should be added twice within the long pipeline of effects you'll find there.  Use the existing effects as a guide for where to place the two instances of a new effect, and place them in alphabetical order.  After you change the PaletteABCD.avc composition and write it out, copy it from the standard location (in
  %USERPROFILE%\Documents\Resolume Avenue\Compositions) to your %CommonProgramFiles%\Palette\data\config directory, so that you don't lose it.

- Add the effect and its parameters to the %CommonProgramFiles%\Palette\data\config\paramdefs.json file.  Here's a sample for the recently-added "goo" effect.
<pre>
"effect.goo": {"valuetype": "bool", "min": "false", "max": "true", "randmax": "0.2", "init": "false", "comment": "#" },

"effect.goo:resolution": {"valuetype": "int", "min": "4", "max": "128", "randmin":"4", "randmax":"32", "init": "16", "comment": "#" },

"effect.goo:speed": {"valuetype": "float", "min": "0.0", "max": "1.0", "randmin":"0.0", "randmax":"0.25", "init": "0.15", "comment": "#" },

"effect.goo:maxdistortionx": {"valuetype": "float", "min": "0.0", "max": "1.0", "randmin":"0.0", "randmax":"1.0", "init": "1.0", "comment": "#" },

"effect.goo:maxdistortiony": {"valuetype": "float", "min": "0.0", "max": "1.0", "randmin":"0.0", "randmax":"1.0", "init": "1.0", "comment": "#" },

"effect.goo:shade": {"valuetype": "float", "min": "0.0", "max": "1.0", "randmin":"0.0", "randmax":"1.0", "init": "1.0", "comment": "#" },
</pre>

- The example above shows three types of parameters: bool, int, and float.
All the values should be quoted strings, even if they look like (and whose valuetypes are) floats or ints.

- The "randmin" and "randmax" values are not required, but should be put on any parameters that should be randomized when the Rand button in the GUI is pressed.
For bool values, the "randmax" value is interpreted as the liklihood that the value will be true.  E.g. the value of "0.2" above says that the "effect:goo" parameter will be true 20% of the time.

- Be very careful when editing the file, to make sure it's valid JSON.
In particular, the last entry in any JSON list (like the one in paramdefs.json) should not be followed by a comma.  E.g. if the "

- Next, edit the resolume.json file to add an entry for the new effect.  Here's the example for the goo effect.
<pre>
    "goo": {
      "on": {
        "addr": "/goo/bypassed",
        "arg": 0
      },
      "off": {
        "addr": "/goo/bypassed",
        "arg": 1
      },
      "params": {
        "resolution": "/goo/effect/resolution",
        "speed": "/goo/effect/speed",
        "maxdistortionx": "/goo/effect/maxdistortionx",
        "maxdistortiony": "/goo/effect/maxdistortiony",
        "maxdistortionz": "/goo/effect/maxdistortionz",
        "shade": "/goo/effect/shade"
      }
    },
</pre>
- The values in the resolume.json file provide the OSC addresses for their respective parameters.  They sometimes match the parameter name displayed in Resolume, but sometimes they are different.  You can use "Shortcuts->Edit OSC" in Resolume to display the OSC address of the parameters, to check.

- Again, carefully note whether there's a comma after the last entry in a JSON list. The two examples above are intended to be inserted into the middle of their respective files, hence they show a final comma.

- That's it!  If you restart the Palette, the new effect should show up in the GUI and be usable.