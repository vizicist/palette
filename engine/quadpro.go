package engine

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type QuadPro struct {

	// Per-patch things
	patch      map[string]*Patch
	patchLogic map[string]*PatchLogic

	started bool

	transposeAuto   bool
	transposeNext   Clicks
	transposeClicks Clicks // time between auto transpose changes
	transposeIndex  int    // current place in tranposeValues
	transposeValues []int
}

var TheQuadPro *QuadPro

func init() {
	quadpro := &QuadPro{
		patch:      map[string]*Patch{},
		patchLogic: map[string]*PatchLogic{},

		transposeAuto:   true,
		transposeNext:   0,
		transposeClicks: 0,
		transposeIndex:  0,
		transposeValues: []int{0, -2, 3, -5},
	}
	RegisterPlugin("quadpro", quadpro.Api)
	TheQuadPro = quadpro
}

func (quadpro *QuadPro) Api(ctx *PluginContext, api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "start":
		return "", quadpro.start(ctx)

	case "stop":
		quadpro.started = false
		return "", StopRunning("all")

	case "set":
		LogWarn("quadpro.set doesn't do anything")
		return "", nil

	case "get":
		return quadpro.onGet(apiargs)

	case "event":
		eventName, ok := apiargs["event"]
		if !ok {
			return "", fmt.Errorf("QuadPro. Missing event argument")
		}

		switch eventName {
		case "midi":
			return quadpro.onMidiEvent(ctx, apiargs)
		case "cursor":
			return quadpro.onCursorEvent(ctx, apiargs)
		case "clientrestart":
			return quadpro.onClientRestart(ctx, apiargs)

		case "quad_save_current":
			return "", quadpro.saveQuad("_Current")

		/*
			case "patch_set":
				// return "", quadpro.onPatchSet(ctx, apiargs)
				LogWarn("This shouldn't be used!  use patch.set api")
				return "", fmt.Errorf("patch_set shouldn't be used!  use patch.set api")
		*/

		case "patch_refresh_all":
			LogWarn("Take a look at patch_refresh_all")
			return "", quadpro.onPatchRefreshAll(apiargs)

		default:
			return "", fmt.Errorf("QuadPro. Unhandled event type %s", eventName)
		}

	case "ANO":
		for _, patch := range quadpro.patch {
			patch.Synth().SendANO()
		}
		return "", nil

	case "echo":
		value, ok := apiargs["value"]
		if !ok {
			value = "ECHO!"
		}
		return value, nil

	case "load":
		category, oksaved := apiargs["category"]
		if !oksaved {
			return "", fmt.Errorf("missing category parameter")
		}
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		TheAttractManager.setAttractMode(false)
		return "", quadpro.Load(ctx, category, filename)

	case "save":
		category, oksaved := apiargs["category"]
		if !oksaved {
			return "", fmt.Errorf("missing category parameter")
		}
		filename, oksaved := apiargs["filename"]
		if !oksaved {
			return "", fmt.Errorf("missing filename parameter")
		}
		return "", quadpro.save(category, filename)

	case "test":
		var ntimes = 10                 // default
		var dt = 500 * time.Millisecond // default
		s, ok := apiargs["ntimes"]
		if ok {
			tmp, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return "", err
			}
			ntimes = int(tmp)
		}
		s, ok = apiargs["dt"]
		if ok {
			tmp, err := time.ParseDuration(s)
			if err != nil {
				return "", err
			}
			dt = tmp
		}
		go quadpro.doTest(ctx, ntimes, dt)
		return "", nil

	default:
		LogWarn("QuadPro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("QuadPro.Api unrecognized api=%s", api)
	}
}

func (quadpro *QuadPro) onPatchRefreshAll(apiargs map[string]string) error {
	patchName, ok := apiargs["patch"]
	if !ok {
		return fmt.Errorf("QuadPro.onPtchRefreshAll: no patch argument")
	}
	patch := GetPatch(patchName)
	if patch == nil {
		return fmt.Errorf("no such patch: %s", patchName)
	}
	patch.RefreshAllPatchValues()
	return nil
}

func (quadpro *QuadPro) start(ctx *PluginContext) error {

	if quadpro.started {
		return fmt.Errorf("QuadPro. already started")
	}
	quadpro.started = true

	ClearExternalScale()

	quadpro.transposeAuto = TheEngine.ParamBool("engine.transposeauto")
	beats := TheEngine.EngineParamIntWithDefault("engine.transposebeats", 48)

	quadpro.transposeNext = Clicks(beats) * OneBeat
	quadpro.transposeClicks = Clicks(beats) * OneBeat

	ctx.AllowSource("A", "B", "C", "D")

	_ = quadpro.addPatch(ctx, "A")
	_ = quadpro.addPatch(ctx, "B")
	_ = quadpro.addPatch(ctx, "C")
	_ = quadpro.addPatch(ctx, "D")

	LogError(quadpro.Load(ctx, "quad", "_Current"))

	// Don't start checking processes right away, after killing them on a restart,
	// they may still be running for a bit
	return nil
}

func (quadpro *QuadPro) onCursorEvent(ctx *PluginContext, apiargs map[string]string) (string, error) {

	ddu, ok := apiargs["ddu"]
	if !ok {
		return "", fmt.Errorf("QuadPro. Missing ddu argument")
	}

	if ddu == "clear" {
		ClearCursors()
		return "", nil
	}

	x, y, z, err := GetArgsXYZ(apiargs)
	if err != nil {
		return "", err
	}

	cid, ok := apiargs["cid"]
	if !ok {
		cid = ""
	}

	ce := CursorEvent{
		Cid:   cid,
		Click: 0,
		Ddu:   ddu,
		X:     x,
		Y:     y,
		Z:     z,
		Area:  0,
	}

	// Any non-internal cursor will turn attract mode off.
	if !ce.IsInternal() {
		TheAttractManager.setAttractMode(false)
	}

	// For the moment, the cursor to patchLogic mapping is 1-to-1.
	// I.e. ce.Source of "A" maps to patchLogic "A"
	patchLogic, ok := quadpro.patchLogic[ce.Source()]
	if !ok || patchLogic == nil {
		return "", nil
	}
	cursorStyle := patchLogic.patch.Get("misc.cursorstyle")
	gensound := IsTrueValue(patchLogic.patch.Get("misc.generatesound"))
	genvisual := IsTrueValue(patchLogic.patch.Get("misc.generatevisual"))
	if gensound && !TheAttractManager.attractModeIsOn {
		patchLogic.generateSoundFromCursor(ctx, ce, cursorStyle)
	}
	if genvisual {
		patchLogic.generateVisualsFromCursor(ce)
	}
	return "", nil
}

func (quadpro *QuadPro) onClientRestart(ctx *PluginContext, apiargs map[string]string) (string, error) {
	// These are messages from the Palette FFGL plugins in Resolume,
	// telling us that they have restarted, and we should resend all parameters
	portnum, ok := apiargs["portnum"]
	if !ok {
		return "", fmt.Errorf("QuadPro. Missing portnum argument")
	}
	ffglportnum, err := strconv.Atoi(portnum)
	if err != nil {
		return "", err
	}
	LogOfType("ffgl","quadpro got clientrestart", "portnum", portnum)
	// The restart message contains the
	// portnum of the plugin that restarted.
	for _, patch := range quadpro.patch {
		e := patch.RefreshAllIfPortnumMatches(ffglportnum)
		if e != nil {
			LogError(e)
			err = e
		}
	}
	return "", err
}

/*
func (quadpro *QuadPro) onPatchSet(ctx *PluginContext, apiargs map[string]string) error {
	paramName, paramValue, err := GetNameValue(apiargs)
	if err != nil {
		return fmt.Errorf("QuadPro.onPatchSet: %s", err)
	}
	patchName, ok := apiargs["patch"]
	if !ok {
		return fmt.Errorf("QuadPro.onPatchSet: no patch argument")
	}
	patch := GetPatch(patchName)
	if patch == nil {
		return fmt.Errorf("no such patch: %s", patchName)
	}
	return patch.Set(paramName, paramValue)
}
*/

func (quadpro *QuadPro) onGet(apiargs map[string]string) (result string, err error) {
	paramName, ok := apiargs["name"]
	if !ok {
		return "", fmt.Errorf("QuadPro.onPatchGet: Missing name argument")
	}
	if strings.HasPrefix(paramName, "engine") {
		return TheEngine.Get(paramName), nil
	} else {
		return "", fmt.Errorf("QuadPro.onGet: can't handle parameter %s", paramName)
	}
}

func (quadpro *QuadPro) onMidiEvent(ctx *PluginContext, apiargs map[string]string) (string, error) {

	me, err := MidiEventFromMap(apiargs)
	if err != nil {
		return "", err
	}

	LogInfo("QuadPro.onMidiEvent", "me", me)

	/*
		if EraeEnabled {
			HandleEraeMIDI(me)
		}
	*/

	return "", nil
}

func (quadpro *QuadPro) doTest(ctx *PluginContext, ntimes int, dt time.Duration) {
	LogInfo("doTest start", "ntimes", ntimes, "dt", dt)
	for n := 0; n < ntimes; n++ {
		if n > 0 {
			time.Sleep(dt)
		}
		source := string("ABCD"[rand.Int()%4])
		cid := source + "#0"
		thismap := map[string]string{
			"cid":   cid,
			"event": "cursor",
			"ddu":   "down",
			"x":     fmt.Sprintf("%f", rand.Float32()),
			"y":     fmt.Sprintf("%f", rand.Float32()),
			"z":     fmt.Sprintf("%f", rand.Float32()),
		}
		_, err := quadpro.Api(ctx, "event", thismap)
		if err != nil {
			LogError(err)
			return
		}
		time.Sleep(dt)
		thismap = map[string]string{
			"cid":   cid,
			"event": "cursor",
			"ddu":   "up",
			"x":     fmt.Sprintf("%f", rand.Float32()),
			"y":     fmt.Sprintf("%f", rand.Float32()),
			"z":     fmt.Sprintf("%f", rand.Float32()),
		}
		_, err = quadpro.Api(ctx, "event", thismap)
		if err != nil {
			LogError(err)
			return
		}
	}
	LogInfo("doTest end")
}

func (quadpro *QuadPro) loadQuadRand() error {

	arr, err := SavedFileList("quad")
	if err != nil {
		return err
	}
	rn := rand.Uint64() % uint64(len(arr))
	LogInfo("loadQuadRand", "quad", arr[rn])

	for _, patch := range quadpro.patch {
		patch.RefreshAllPatchValues()
	}
	return nil
}

func (quadpro *QuadPro) Load(ctx *PluginContext, category string, filename string) error {

	path := SavedFilePath(category, filename)
	paramsMap, err := LoadParamsMap(path)
	if err != nil {
		LogError(err)
		return err
	}

	LogInfo("QuadPro.Load", "category", category, "filename", filename)

	var lasterr error

	switch category {

	case "quad":
		for _, patch := range quadpro.patch {
			err := patch.ApplyPatchValuesFromQuadMap(paramsMap)
			if err != nil {
				LogError(err)
				lasterr = err
			}
			patch.RefreshAllPatchValues()
		}

	case "engine":
		TheEngine.params.ApplyValuesFromMap("engine", paramsMap, TheEngine.Set)
		LogInfo("HEY! need work here")

	default:
		LogWarn("QuadPro.Load: unhandled", "category", category, "filename", filename)
	}

	// Decide what _Current things we should save
	// when we load something.  E.g. if we're
	// loading a patch with something, we want to
	// then save the entire quad
	err = nil
	switch category {
	case "engine":
		// No need to save _Current if we're loading it.
		if filename != "_Current" {
			err = TheEngine.SaveCurrent()
		}
	case "quad":
		if filename != "_Current" {
			err = quadpro.saveQuad("_Current")
		}
	case "patch", "sound", "visual", "effect", "misc":
		// If we're loading a patch (or something inside a patch, like sound, visual, etc),
		// we save the entire quad, since that's our real persistent state
		if filename != "_Current" {
			err = quadpro.saveQuad("_Current")
		}
	}
	if err != nil {
		LogError(err)
		lasterr = err
	}
	return lasterr
}

func (quadpro *QuadPro) save(category string, filename string) (err error) {

	LogOfType("saved", "QuadPro.save", "category", category, "filename", filename)

	if category == "engine" {
		// err = quadpro.engineClicks.Save("engine", filename)
		LogWarn("QuadPro.save: shouldn't be saving global?")
		err = fmt.Errorf("QuadPro.save: global shouldn't be handled here")
	} else if category == "quad" {
		err = quadpro.saveQuad(filename)
	} else {
		err = fmt.Errorf("QuadPro.Api: unhandled save category %s", category)
	}
	if err != nil {
		LogError(err)
	}
	return err
}

func (quadpro *QuadPro) saveQuad(quadName string) error {

	category := "quad"
	path := WritableFilePath(category, quadName)

	LogOfType("saved", "QuadPro.saveQuad", "quad", quadName)

	sortedPatchNames := []string{}
	for _, patch := range quadpro.patch {
		sortedPatchNames = append(sortedPatchNames, patch.Name())
	}
	sort.Strings(sortedPatchNames)

	s := "{\n    \"params\": {\n"
	sep := ""
	for _, patchName := range sortedPatchNames {
		patch := quadpro.patch[patchName]
		sortedNames := patch.ParamNames()
		for _, fullName := range sortedNames {
			valstring := patch.Get(fullName)
			s += fmt.Sprintf("%s        \"%s-%s\":\"%s\"", sep, patch.Name(), fullName, valstring)
			sep = ",\n"
		}
	}
	s += "\n    }\n}"
	data := []byte(s)
	return os.WriteFile(path, data, 0644)
}

func (quadpro *QuadPro) addPatch(ctx *PluginContext, name string) *Patch {
	patch := NewPatch(name)
	patch.AddListener(ctx)
	quadpro.patch[name] = patch
	quadpro.patchLogic[name] = NewPatchLogic(patch)
	return patch
}

/*
func (quadpro *QuadPro) scheduleNoteNow(ctx *PluginContext, dest string, pitch, velocity uint8, duration Clicks) {
	LogInfo("QuadPro.scheculeNoteNow", "dest", dest, "pitch", pitch)
	pe := &PhraseElement{Value: NewNoteFull(0, pitch, velocity, duration)}
	phr := NewPhrase().InsertElement(pe)
	phr.Destination = dest
	SchedulePhrase(phr, CurrentClick(), dest)
}
*/

func MmttInfo() *ProcessInfo {

	// NOTE: it's inside a sub-directory of bin, so all the necessary .dll's are contained

	// The value of mmtt is either "kinect" or "oak" or ""
	mmtt := TheEngine.Get("engine.mmtt")
	if mmtt == "" {
		return nil
	}
	fullpath := filepath.Join(PaletteDir(), "bin", "mmtt_"+mmtt, "mmtt_"+mmtt+".exe")
	if !FileExists(fullpath) {
		LogWarn("no mmtt executable found, looking for", "path", fullpath)
		fullpath = ""
	}
	return NewProcessInfo("mmtt_"+mmtt+".exe", fullpath, "", nil)
}
