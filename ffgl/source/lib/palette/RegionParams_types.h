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
DEFINE_TYPES(midibehaviour);
DEFINE_TYPES(scale);
DEFINE_TYPES(inputport);
DEFINE_TYPES(synth);
DEFINE_TYPES(regionport);

void
RegionParams_InitializeTypes() {

	RegionParams_destinationTypes.push_back(".");
	RegionParams_destinationTypes.push_back("A");
	RegionParams_destinationTypes.push_back("B");
	RegionParams_destinationTypes.push_back("C");
	RegionParams_destinationTypes.push_back("D");

	RegionParams_logic_soundTypes.push_back("default");
	RegionParams_logic_soundTypes.push_back("midigrid");

	RegionParams_logic_visualTypes.push_back("default");
	RegionParams_logic_visualTypes.push_back("maze");
	RegionParams_logic_visualTypes.push_back("maze4");
	RegionParams_logic_visualTypes.push_back("maze33");

	RegionParams_quantTypes.push_back("none");
	RegionParams_quantTypes.push_back("frets");
	RegionParams_quantTypes.push_back("fixed");
	RegionParams_quantTypes.push_back("pressure");

	RegionParams_volTypes.push_back("fixed");
	RegionParams_volTypes.push_back("pressure");

	RegionParams_shapeTypes.push_back("line");
	RegionParams_shapeTypes.push_back("triangle");
	RegionParams_shapeTypes.push_back("square");
	RegionParams_shapeTypes.push_back("circle");

	RegionParams_movedirTypes.push_back("cursor");
	RegionParams_movedirTypes.push_back("left");
	RegionParams_movedirTypes.push_back("right");
	RegionParams_movedirTypes.push_back("up");
	RegionParams_movedirTypes.push_back("down");
	RegionParams_movedirTypes.push_back("random");
	RegionParams_movedirTypes.push_back("random90");
	RegionParams_movedirTypes.push_back("updown");
	RegionParams_movedirTypes.push_back("leftright");

	RegionParams_rotangdirTypes.push_back("right");
	RegionParams_rotangdirTypes.push_back("left");
	RegionParams_rotangdirTypes.push_back("random");

	RegionParams_mirrorTypes.push_back("none");
	RegionParams_mirrorTypes.push_back("vertical");
	RegionParams_mirrorTypes.push_back("horizontal");
	RegionParams_mirrorTypes.push_back("four");

	RegionParams_justificationTypes.push_back("center");
	RegionParams_justificationTypes.push_back("left");
	RegionParams_justificationTypes.push_back("right");
	RegionParams_justificationTypes.push_back("top");
	RegionParams_justificationTypes.push_back("bottom");
	RegionParams_justificationTypes.push_back("topleft");
	RegionParams_justificationTypes.push_back("topright");
	RegionParams_justificationTypes.push_back("bottomleft");
	RegionParams_justificationTypes.push_back("bottomright");

	RegionParams_controllerstyleTypes.push_back("modulationonly");
	RegionParams_controllerstyleTypes.push_back("allcontrollers");
	RegionParams_controllerstyleTypes.push_back("pitchYZ");
	RegionParams_controllerstyleTypes.push_back("nothing");

	RegionParams_placementTypes.push_back("random");
	RegionParams_placementTypes.push_back("linear");
	RegionParams_placementTypes.push_back("cursor");
	RegionParams_placementTypes.push_back("top");
	RegionParams_placementTypes.push_back("bottom");
	RegionParams_placementTypes.push_back("left");
	RegionParams_placementTypes.push_back("right");

	RegionParams_spritesourceTypes.push_back("cursor");
	RegionParams_spritesourceTypes.push_back("midi");
	RegionParams_spritesourceTypes.push_back("none");

	RegionParams_midibehaviourTypes.push_back("scalecapture");
	RegionParams_midibehaviourTypes.push_back("none");
	RegionParams_midibehaviourTypes.push_back("sprite");

	RegionParams_scaleTypes.push_back("external");
	RegionParams_scaleTypes.push_back("newage");
	RegionParams_scaleTypes.push_back("arabian");
	RegionParams_scaleTypes.push_back("ionian");
	RegionParams_scaleTypes.push_back("dorian");
	RegionParams_scaleTypes.push_back("phrygian");
	RegionParams_scaleTypes.push_back("lydian");
	RegionParams_scaleTypes.push_back("mixolydian");
	RegionParams_scaleTypes.push_back("aeolian");
	RegionParams_scaleTypes.push_back("locrian");
	RegionParams_scaleTypes.push_back("octaves");
	RegionParams_scaleTypes.push_back("harminor");
	RegionParams_scaleTypes.push_back("melminor");
	RegionParams_scaleTypes.push_back("chromatic");

	RegionParams_inputportTypes.push_back("");
	RegionParams_inputportTypes.push_back("microKEY2 Air");

	RegionParams_synthTypes.push_back("");
	RegionParams_synthTypes.push_back("DummyWave");

	RegionParams_regionportTypes.push_back("01. Internal MIDI");
	RegionParams_regionportTypes.push_back("02. Internal MIDI");
	RegionParams_regionportTypes.push_back("03. Internal MIDI");
	RegionParams_regionportTypes.push_back("04. Internal MIDI");
	RegionParams_regionportTypes.push_back("05. Internal MIDI");
	RegionParams_regionportTypes.push_back("06. Internal MIDI");
	RegionParams_regionportTypes.push_back("07. Internal MIDI");
	RegionParams_regionportTypes.push_back("08. Internal MIDI");
	RegionParams_regionportTypes.push_back("09. Internal MIDI");
	RegionParams_regionportTypes.push_back("10. Internal MIDI");
	RegionParams_regionportTypes.push_back("11. Internal MIDI");
	RegionParams_regionportTypes.push_back("12. Internal MIDI");
};
