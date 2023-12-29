DEFINE_TYPES(resolumepath);
DEFINE_TYPES(plugins);
DEFINE_TYPES(log);
DEFINE_TYPES(pitchset);
DEFINE_TYPES(guisize);
DEFINE_TYPES(mmtt);
DEFINE_TYPES(destination);
DEFINE_TYPES(logic_sound);
DEFINE_TYPES(logic_visual);
DEFINE_TYPES(quantstyle);
DEFINE_TYPES(volstyle);
DEFINE_TYPES(shape);
DEFINE_TYPES(morphtype);
DEFINE_TYPES(midiinput);
DEFINE_TYPES(movedir);
DEFINE_TYPES(rotangdir);
DEFINE_TYPES(mirror);
DEFINE_TYPES(justification);
DEFINE_TYPES(controllerstyle);
DEFINE_TYPES(cursorstyle);
DEFINE_TYPES(placement);
DEFINE_TYPES(spritesource);
DEFINE_TYPES(spritestyle);
DEFINE_TYPES(midibehaviour);
DEFINE_TYPES(patch);
DEFINE_TYPES(scale);
DEFINE_TYPES(enginescale);
DEFINE_TYPES(inputport);
DEFINE_TYPES(synth);

void
LayerParams_InitializeTypes() {

	LayerParams_resolumepathTypes.push_back("C:/Program Files/Resolume Avenue/Avenue.exe");
	LayerParams_resolumepathTypes.push_back("C:/Program Files/Resolume Arena/Arena.exe");

	LayerParams_pluginsTypes.push_back("");
	LayerParams_pluginsTypes.push_back("quad");

	LayerParams_logTypes.push_back("");
	LayerParams_logTypes.push_back("cursor");
	LayerParams_logTypes.push_back("midi");
	LayerParams_logTypes.push_back("cursor,midi");
	LayerParams_logTypes.push_back("cursor,midi,osc");
	LayerParams_logTypes.push_back("osc");
	LayerParams_logTypes.push_back("api");
	LayerParams_logTypes.push_back("loop");
	LayerParams_logTypes.push_back("morph");
	LayerParams_logTypes.push_back("mmtt");
	LayerParams_logTypes.push_back("quant");
	LayerParams_logTypes.push_back("transpose");
	LayerParams_logTypes.push_back("ffgl");
	LayerParams_logTypes.push_back("gesture");

	LayerParams_pitchsetTypes.push_back("");
	LayerParams_pitchsetTypes.push_back("stylusrmx");

	LayerParams_guisizeTypes.push_back("small");
	LayerParams_guisizeTypes.push_back("medium");
	LayerParams_guisizeTypes.push_back("palette");

	LayerParams_mmttTypes.push_back("kinect");
	LayerParams_mmttTypes.push_back("depthai");
	LayerParams_mmttTypes.push_back("");

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

	LayerParams_quantstyleTypes.push_back("none");
	LayerParams_quantstyleTypes.push_back("frets");
	LayerParams_quantstyleTypes.push_back("fixed");
	LayerParams_quantstyleTypes.push_back("pressure");

	LayerParams_volstyleTypes.push_back("fixed");
	LayerParams_volstyleTypes.push_back("pressure");

	LayerParams_shapeTypes.push_back("line");
	LayerParams_shapeTypes.push_back("triangle");
	LayerParams_shapeTypes.push_back("square");
	LayerParams_shapeTypes.push_back("circle");

	LayerParams_morphtypeTypes.push_back("quadrants");
	LayerParams_morphtypeTypes.push_back("corners");
	LayerParams_morphtypeTypes.push_back("A");
	LayerParams_morphtypeTypes.push_back("B");
	LayerParams_morphtypeTypes.push_back("C");
	LayerParams_morphtypeTypes.push_back("D");

	LayerParams_midiinputTypes.push_back("microKEY2 Air");
	LayerParams_midiinputTypes.push_back("");

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

	LayerParams_cursorstyleTypes.push_back("downonly");
	LayerParams_cursorstyleTypes.push_back("retrigger");

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

	LayerParams_patchTypes.push_back("A");
	LayerParams_patchTypes.push_back("B");
	LayerParams_patchTypes.push_back("C");
	LayerParams_patchTypes.push_back("D");

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

	LayerParams_enginescaleTypes.push_back("");
	LayerParams_enginescaleTypes.push_back("external");
	LayerParams_enginescaleTypes.push_back("newage");
	LayerParams_enginescaleTypes.push_back("arabian");
	LayerParams_enginescaleTypes.push_back("ionian");
	LayerParams_enginescaleTypes.push_back("dorian");
	LayerParams_enginescaleTypes.push_back("phrygian");
	LayerParams_enginescaleTypes.push_back("lydian");
	LayerParams_enginescaleTypes.push_back("mixolydian");
	LayerParams_enginescaleTypes.push_back("aeolian");
	LayerParams_enginescaleTypes.push_back("locrian");
	LayerParams_enginescaleTypes.push_back("octaves");
	LayerParams_enginescaleTypes.push_back("harminor");
	LayerParams_enginescaleTypes.push_back("melminor");
	LayerParams_enginescaleTypes.push_back("chromatic");

	LayerParams_inputportTypes.push_back("");

	LayerParams_synthTypes.push_back("");
};
