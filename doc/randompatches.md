# Evaluating random patches for painting

Automatic acceptance favors visible output that follows the location of cursor
gestures. A detailed or animated image alone is insufficient.

`global.randeffectcount` controls how many distinct effect plugins a random
patch uses (0-10, default 6). Each selected plugin is enabled in one effect bank. Zero
produces a patch with no enabled effects; counts above the available plugin types
are capped. The GUI's Effect Rand respects the existing opt-in randomization
ranges. Batch generators read the same count when selecting plugins.

For a complete random patch, randomize Visual and Sound as well as Effect.
Visual uses Palette's configured Rand ranges and current shape choices. Sound
selects a registered synth and controller style, then randomizes ordered MIDI
pitch and velocity ranges. Both are saved with the effects in each accepted
patch. Synthetic visual evaluation is silent; saved patches have sound enabled.

String parameter definitions can include a `randchoices` array to restrict Rand
to a subset of their valid options. The list must be nonempty and contain valid,
distinct strings. It enables randomization even if `randmax` is omitted. Without
`randchoices`, the existing string Rand rules apply; a nonempty `randmax` still
pins a specific value. Manual choices and preset loading keep the full enum.

`visual.shape` currently uses `"randchoices": ["line", "circle", "square", "triangle"]`.

`global.interest_eval` draws four controlled strokes at separated positions,
covering horizontal and vertical movement in both directions. For each of the
twelve cursor positions, the evaluator captures background motion before moving
the cursor and then captures the response. It measures how much visible change
is concentrated around the new stroke segment, subtracting the overlap that
fullscreen animation would receive by chance. The existing cursor scale and
offset settings are included when predicting the position on the output monitor.
Each stroke starts with cleared patch graphics so persistent marks from a prior
test cannot hide the new response. An evaluation takes about six seconds.

New marks can be light or dark. Fading earlier marks and independent background
motion are discounted. Sparse, controllable brush marks can pass without filling
the screen with texture. Randomly placed output, fixed central animation, and
fullscreen flashes score poorly.

The final score is the painting score with a small bonus for visual richness.
`global.interest_score` reports `score`, `appearance`, and `painting` (including
`response`, `localization`, and `score`).

In Global parameters, `global.interestlevel` sets the minimum score for accepting
a random patch, from 0 (most permissive) to 1 (most demanding). The default is
0.80. Level zero still rejects output with no detectable painting response. The
result includes `level` and `accepted`; the GUI's Rand retries use `accepted`.
Batch generators should compare the final `score`, not `appearance`, against
`global.interestlevel`. Each candidate is drawn from the configured random
choices and ranges; accepted and rejected results do not train future draws.

Use `global.interestmonitor` to select the actual Resolume output monitor. The
comparison assumes that output fills the monitor and uses Palette's normal
coordinate orientation. For batch evaluation, disable attract mode, silence and
clear the other pads, and evaluate each candidate. Confirm initial passes with a
second test, then restore the previous state afterward. Save only candidates
that pass both tests, using the next unused `!!!X001`,
`!!!X002`, and so on, with three digits and leading zeroes. Existing saved patches
are not automatically re-evaluated.
