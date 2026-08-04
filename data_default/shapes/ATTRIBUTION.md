# Shape attributions

Any `.svg` file placed in this directory becomes available as a
`visual.shape` value (the engine scans it at startup; see
`loadShapeEnums` in `kit/params_defs.go`). The renderer's SVG support is
minimal — see the comment at the top of
`ffgl/source/lib/palette/SvgSprite.cpp` for what a file may contain
(path elements only, commands M L H V C S Q T A Z, at most one outer
`<g transform>` with translate/scale).

## game-icons.net icons

The icons below are from [game-icons.net](https://game-icons.net), used
under the [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/)
license, from the repository
[github.com/game-icons/icons](https://github.com/game-icons/icons).
Two modifications, neither touching the path geometry: the full-frame
background path (`<path d="M0 0h512v512H0z"/>`) was removed from each
file so the icon renders as a standalone shape, and the remaining
path's `fill` was changed from white to black so the files preview
visibly in file managers. The engine ignores `fill` entirely (sprite
color comes from the `visual.hue*` parameters), so it is presentation
metadata only.

Authors: **Lorc** (https://lorcblog.blogspot.com) and **Delapouite**
(https://delapouite.com).

| File | Original icon | Author |
|---|---|---|
| butterfly.svg | lorc/butterfly.svg | Lorc |
| clef.svg | delapouite/g-clef.svg | Delapouite |
| dove.svg | lorc/dove.svg | Lorc |
| dragonfly.svg | lorc/dragonfly.svg | Lorc |
| elephant.svg | delapouite/elephant.svg | Delapouite |
| feather.svg | lorc/feather.svg | Lorc |
| frog.svg | lorc/frog.svg | Lorc |
| hummingbird.svg | delapouite/hummingbird.svg | Delapouite |
| jellyfish.svg | lorc/jellyfish.svg | Lorc |
| lotus.svg | lorc/lotus.svg | Lorc |
| mapleleaf.svg | lorc/maple-leaf.svg | Lorc |
| moon.svg | lorc/moon.svg | Lorc |
| mushroom.svg | lorc/mushroom.svg | Lorc |
| octopus.svg | lorc/octopus.svg | Lorc |
| owl.svg | lorc/owl.svg | Lorc |
| palmtree.svg | delapouite/palm-tree.svg | Delapouite |
| snail.svg | lorc/snail.svg | Lorc |
| snowflake.svg | lorc/snowflake-1.svg | Lorc |
| sun.svg | lorc/sun.svg | Lorc |
| sunflower.svg | delapouite/sunflower.svg | Delapouite |
| turtle.svg | delapouite/sea-turtle.svg | Delapouite |
| vortex.svg | lorc/vortex.svg | Lorc |
| wolf.svg | lorc/wolf-howl.svg | Lorc |
| yinyang.svg | delapouite/yin-yang.svg | Delapouite |

## Third-party brand artwork

`dirtygoat.svg` is traced from the Dirty Goat Roadhouse logo
(`DGR_BW_highres_onwhite.jpg` in the venue's supplied logo package). It
is the venue's trademark, included here for Palette performances at that
venue — not covered by this repository's license, and not for reuse
elsewhere without their permission.

## Other shapes

chaos, directive, oracle, sacred, and the goat* shapes are
project-original artwork (traced with potrace).
