# Palette - a system for visual music instruments

# WARNING - main branch is under active development, do not use.

Palette is used in the <a href=https://youtu.be/HDtxEyCI_zc>Space Palette Pro</a>,
an instrument that lets you fingerpaint sound and visuals
using your fingers as 3D cursors on Sensel Morph pads.  The Palette software consists of:

* A realtime engine written in Go that accepts 3D cursors and MIDI as input,
      and generates MIDI (for sounds) and OSC (for visuals in Resolume).  Most of the functionality in the system is implemented here.  The primary source of 3D cursors is
      the Sensel Morph pads, 
      and the support for this is built into the engine.
      It is possible to drive the system via OSC or JSON/HTTP;
      for example the OSC interface is used by the Kinect in the original Space Palette.

* A Freeframe plugin written in C++ that runs inside Resolume, generating the graphical sprites that are controlled by OSC from the realtime engine.

* A GUI written in Python that lets you control the presets and parameters of
      the musical and graphical output.  The third iteration of this GUI, it has both a "casual" single-screen interface and a more advanced "pro" interface which allows:

1. Looping, Scales, MIDI input, and other controls
1. Independent control of the four pads
1. Creation and modification of the presets
1. Independent preset control of the Sound, Visual, and Effect parameters.  Sound refers to the MIDI output,
      Visual refers to the sprite generation of the Palette FFGL plugin, and Effect refers to all other FFGL plugins.

# Documentation

- <a href=doc/using_resolume.md>Using Palette with the Sensel Morph as a visual instrument in Resolume

- <a href=doc/starting_and_using.md>Using it after it's been installed</a>

- <a href=doc/implementation.md>Implementation details</a>

- <a href=doc/building.md>Configuring a build/development environment on Windows</a>

- <a href=doc/addingeffect.md>How to add a new FFGL effect to the Palette.</a>


