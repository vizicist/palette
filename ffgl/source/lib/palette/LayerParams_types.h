DEFINE_TYPES(destination);
DEFINE_TYPES(logic_sound);
DEFINE_TYPES(logic_visual);
DEFINE_TYPES(quant);
DEFINE_TYPES(vol);
DEFINE_TYPES(shape);
DEFINE_TYPES(movedir);
DEFINE_TYPES(rotangdir);
DEFINE_TYPES(mirror);
DEFINE_TYPES(justification);
DEFINE_TYPES(controllerstyle);
DEFINE_TYPES(placement);
DEFINE_TYPES(spritesource);
DEFINE_TYPES(spritestyle);
DEFINE_TYPES(midibehaviour);
DEFINE_TYPES(scale);
DEFINE_TYPES(inputport);
DEFINE_TYPES(synth);
DEFINE_TYPES(midiport);

void
LayerParams_InitializeTypes() {

	LayerParams_destinationTypes.push_back(".");
	LayerParams_destinationTypes.push_back("A");
	LayerParams_destinationTypes.push_back("B");
	LayerParams_destinationTypes.push_back("C");
	LayerParams_destinationTypes.push_back("D");

	LayerParams_logic_soundTypes.push_back("default");
	LayerParams_logic_soundTypes.push_back("midigrid");

	LayerParams_logic_visualTypes.push_back("default");
	LayerParams_logic_visualTypes.push_back("maze");
	LayerParams_logic_visualTypes.push_back("maze4");
	LayerParams_logic_visualTypes.push_back("maze33");

	LayerParams_quantTypes.push_back("none");
	LayerParams_quantTypes.push_back("frets");
	LayerParams_quantTypes.push_back("fixed");
	LayerParams_quantTypes.push_back("pressure");

	LayerParams_volTypes.push_back("fixed");
	LayerParams_volTypes.push_back("pressure");

	LayerParams_shapeTypes.push_back("line");
	LayerParams_shapeTypes.push_back("triangle");
	LayerParams_shapeTypes.push_back("square");
	LayerParams_shapeTypes.push_back("circle");

	LayerParams_movedirTypes.push_back("cursor");
	LayerParams_movedirTypes.push_back("left");
	LayerParams_movedirTypes.push_back("right");
	LayerParams_movedirTypes.push_back("up");
	LayerParams_movedirTypes.push_back("down");
	LayerParams_movedirTypes.push_back("random");
	LayerParams_movedirTypes.push_back("random90");
	LayerParams_movedirTypes.push_back("updown");
	LayerParams_movedirTypes.push_back("leftright");

	LayerParams_rotangdirTypes.push_back("right");
	LayerParams_rotangdirTypes.push_back("left");
	LayerParams_rotangdirTypes.push_back("random");

	LayerParams_mirrorTypes.push_back("none");
	LayerParams_mirrorTypes.push_back("vertical");
	LayerParams_mirrorTypes.push_back("horizontal");
	LayerParams_mirrorTypes.push_back("four");

	LayerParams_justificationTypes.push_back("center");
	LayerParams_justificationTypes.push_back("left");
	LayerParams_justificationTypes.push_back("right");
	LayerParams_justificationTypes.push_back("top");
	LayerParams_justificationTypes.push_back("bottom");
	LayerParams_justificationTypes.push_back("topleft");
	LayerParams_justificationTypes.push_back("topright");
	LayerParams_justificationTypes.push_back("bottomleft");
	LayerParams_justificationTypes.push_back("bottomright");

	LayerParams_controllerstyleTypes.push_back("modulationonly");
	LayerParams_controllerstyleTypes.push_back("allcontrollers");
	LayerParams_controllerstyleTypes.push_back("pitchYZ");
	LayerParams_controllerstyleTypes.push_back("nothing");

	LayerParams_placementTypes.push_back("random");
	LayerParams_placementTypes.push_back("linear");
	LayerParams_placementTypes.push_back("cursor");
	LayerParams_placementTypes.push_back("top");
	LayerParams_placementTypes.push_back("bottom");
	LayerParams_placementTypes.push_back("left");
	LayerParams_placementTypes.push_back("right");

	LayerParams_spritesourceTypes.push_back("cursor");
	LayerParams_spritesourceTypes.push_back("midi");
	LayerParams_spritesourceTypes.push_back("none");

	LayerParams_spritestyleTypes.push_back("hue");
	LayerParams_spritestyleTypes.push_back("texture");

	LayerParams_midibehaviourTypes.push_back("scalecapture");
	LayerParams_midibehaviourTypes.push_back("none");
	LayerParams_midibehaviourTypes.push_back("sprite");

	LayerParams_scaleTypes.push_back("external");
	LayerParams_scaleTypes.push_back("newage");
	LayerParams_scaleTypes.push_back("arabian");
	LayerParams_scaleTypes.push_back("ionian");
	LayerParams_scaleTypes.push_back("dorian");
	LayerParams_scaleTypes.push_back("phrygian");
	LayerParams_scaleTypes.push_back("lydian");
	LayerParams_scaleTypes.push_back("mixolydian");
	LayerParams_scaleTypes.push_back("aeolian");
	LayerParams_scaleTypes.push_back("locrian");
	LayerParams_scaleTypes.push_back("octaves");
	LayerParams_scaleTypes.push_back("harminor");
	LayerParams_scaleTypes.push_back("melminor");
	LayerParams_scaleTypes.push_back("chromatic");

	LayerParams_inputportTypes.push_back("");
	LayerParams_inputportTypes.push_back("microKEY2 Air");

	LayerParams_synthTypes.push_back("");
	LayerParams_synthTypes.push_back("DummyWave");

	LayerParams_midiportTypes.push_back("01. Internal MIDI");
	LayerParams_midiportTypes.push_back("02. Internal MIDI");
	LayerParams_midiportTypes.push_back("03. Internal MIDI");
	LayerParams_midiportTypes.push_back("04. Internal MIDI");
	LayerParams_midiportTypes.push_back("05. Internal MIDI");
	LayerParams_midiportTypes.push_back("06. Internal MIDI");
	LayerParams_midiportTypes.push_back("07. Internal MIDI");
	LayerParams_midiportTypes.push_back("08. Internal MIDI");
	LayerParams_midiportTypes.push_back("09. Internal MIDI");
	LayerParams_midiportTypes.push_back("10. Internal MIDI");
	LayerParams_midiportTypes.push_back("11. Internal MIDI");
	LayerParams_midiportTypes.push_back("12. Internal MIDI");
};