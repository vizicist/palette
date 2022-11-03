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
PlayerParams_InitializeTypes() {

	PlayerParams_destinationTypes.push_back(".");
	PlayerParams_destinationTypes.push_back("A");
	PlayerParams_destinationTypes.push_back("B");
	PlayerParams_destinationTypes.push_back("C");
	PlayerParams_destinationTypes.push_back("D");

	PlayerParams_logic_soundTypes.push_back("default");
	PlayerParams_logic_soundTypes.push_back("midigrid");

	PlayerParams_logic_visualTypes.push_back("default");
	PlayerParams_logic_visualTypes.push_back("maze");
	PlayerParams_logic_visualTypes.push_back("maze4");
	PlayerParams_logic_visualTypes.push_back("maze33");

	PlayerParams_quantTypes.push_back("none");
	PlayerParams_quantTypes.push_back("frets");
	PlayerParams_quantTypes.push_back("fixed");
	PlayerParams_quantTypes.push_back("pressure");

	PlayerParams_volTypes.push_back("fixed");
	PlayerParams_volTypes.push_back("pressure");

	PlayerParams_shapeTypes.push_back("line");
	PlayerParams_shapeTypes.push_back("triangle");
	PlayerParams_shapeTypes.push_back("square");
	PlayerParams_shapeTypes.push_back("circle");

	PlayerParams_movedirTypes.push_back("cursor");
	PlayerParams_movedirTypes.push_back("left");
	PlayerParams_movedirTypes.push_back("right");
	PlayerParams_movedirTypes.push_back("up");
	PlayerParams_movedirTypes.push_back("down");
	PlayerParams_movedirTypes.push_back("random");
	PlayerParams_movedirTypes.push_back("random90");
	PlayerParams_movedirTypes.push_back("updown");
	PlayerParams_movedirTypes.push_back("leftright");

	PlayerParams_rotangdirTypes.push_back("right");
	PlayerParams_rotangdirTypes.push_back("left");
	PlayerParams_rotangdirTypes.push_back("random");

	PlayerParams_mirrorTypes.push_back("none");
	PlayerParams_mirrorTypes.push_back("vertical");
	PlayerParams_mirrorTypes.push_back("horizontal");
	PlayerParams_mirrorTypes.push_back("four");

	PlayerParams_justificationTypes.push_back("center");
	PlayerParams_justificationTypes.push_back("left");
	PlayerParams_justificationTypes.push_back("right");
	PlayerParams_justificationTypes.push_back("top");
	PlayerParams_justificationTypes.push_back("bottom");
	PlayerParams_justificationTypes.push_back("topleft");
	PlayerParams_justificationTypes.push_back("topright");
	PlayerParams_justificationTypes.push_back("bottomleft");
	PlayerParams_justificationTypes.push_back("bottomright");

	PlayerParams_controllerstyleTypes.push_back("modulationonly");
	PlayerParams_controllerstyleTypes.push_back("allcontrollers");
	PlayerParams_controllerstyleTypes.push_back("pitchYZ");
	PlayerParams_controllerstyleTypes.push_back("nothing");

	PlayerParams_placementTypes.push_back("random");
	PlayerParams_placementTypes.push_back("linear");
	PlayerParams_placementTypes.push_back("cursor");
	PlayerParams_placementTypes.push_back("top");
	PlayerParams_placementTypes.push_back("bottom");
	PlayerParams_placementTypes.push_back("left");
	PlayerParams_placementTypes.push_back("right");

	PlayerParams_spritesourceTypes.push_back("cursor");
	PlayerParams_spritesourceTypes.push_back("midi");
	PlayerParams_spritesourceTypes.push_back("none");

	PlayerParams_spritestyleTypes.push_back("hue");
	PlayerParams_spritestyleTypes.push_back("texture");

	PlayerParams_midibehaviourTypes.push_back("scalecapture");
	PlayerParams_midibehaviourTypes.push_back("none");
	PlayerParams_midibehaviourTypes.push_back("sprite");

	PlayerParams_scaleTypes.push_back("external");
	PlayerParams_scaleTypes.push_back("newage");
	PlayerParams_scaleTypes.push_back("arabian");
	PlayerParams_scaleTypes.push_back("ionian");
	PlayerParams_scaleTypes.push_back("dorian");
	PlayerParams_scaleTypes.push_back("phrygian");
	PlayerParams_scaleTypes.push_back("lydian");
	PlayerParams_scaleTypes.push_back("mixolydian");
	PlayerParams_scaleTypes.push_back("aeolian");
	PlayerParams_scaleTypes.push_back("locrian");
	PlayerParams_scaleTypes.push_back("octaves");
	PlayerParams_scaleTypes.push_back("harminor");
	PlayerParams_scaleTypes.push_back("melminor");
	PlayerParams_scaleTypes.push_back("chromatic");

	PlayerParams_inputportTypes.push_back("");
	PlayerParams_inputportTypes.push_back("microKEY2 Air");

	PlayerParams_synthTypes.push_back("");
	PlayerParams_synthTypes.push_back("DummyWave");

	PlayerParams_midiportTypes.push_back("01. Internal MIDI");
	PlayerParams_midiportTypes.push_back("02. Internal MIDI");
	PlayerParams_midiportTypes.push_back("03. Internal MIDI");
	PlayerParams_midiportTypes.push_back("04. Internal MIDI");
	PlayerParams_midiportTypes.push_back("05. Internal MIDI");
	PlayerParams_midiportTypes.push_back("06. Internal MIDI");
	PlayerParams_midiportTypes.push_back("07. Internal MIDI");
	PlayerParams_midiportTypes.push_back("08. Internal MIDI");
	PlayerParams_midiportTypes.push_back("09. Internal MIDI");
	PlayerParams_midiportTypes.push_back("10. Internal MIDI");
	PlayerParams_midiportTypes.push_back("11. Internal MIDI");
	PlayerParams_midiportTypes.push_back("12. Internal MIDI");
};
