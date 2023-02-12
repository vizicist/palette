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

var TheQuadPro *QuadPro

type QuadPro struct {

	// Per-patch things
	patch      map[string]*Patch
	patchLogic map[string]*PatchLogic

	started bool
}

func NewQuadPro() *QuadPro {
	quadpro := &QuadPro{
		patch:      map[string]*Patch{},
		patchLogic: map[string]*PatchLogic{},
	}
	return quadpro
}

func (quadpro *QuadPro) Stop() {
	quadpro.started = false
}

func (quadpro *QuadPro) Api(api string, apiargs map[string]string) (result string, err error) {

	switch api {

	case "get":
		return quadpro.onGet(apiargs)

	case "event":
		return "", fmt.Errorf("event no longer handled in Quadpro.Api")

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
		return "", quadpro.Load(category, filename)

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
		go quadpro.doTest(ntimes, dt)
		return "", nil

	default:
		LogWarn("QuadPro.ExecuteAPI api is not recognized\n", "api", api)
		return "", fmt.Errorf("QuadPro.Api unrecognized api=%s", api)
	}
}

func (quadpro *QuadPro) Start() {

	if quadpro.started {
		LogInfo("QuadPro.Start: already started")
		return
	}
	quadpro.started = true

	TheCursorManager.AddCursorHandler("QuadPro", TheQuadPro, "A", "B", "C", "D")

	_ = quadpro.addPatch("A")
	_ = quadpro.addPatch("B")
	_ = quadpro.addPatch("C")
	_ = quadpro.addPatch("D")

	err := quadpro.Load("quad", "_Current")
	LogIfError(err)
}

func (quadpro *QuadPro) PatchForCursorEvent(ce CursorEvent) *Patch {
	patchLogic, ok := quadpro.patchLogic[ce.Source()]
	if !ok {
		LogWarn("QuadPro.PatchForCursorEvent: no patchLogic for source", "source", ce.Source())
		return nil
	}
	return patchLogic.patch
}

func (quadpro *QuadPro) onCursorEvent(state ActiveCursor) error {

	if state.Current.Ddu == "clear" {
		TheCursorManager.clearCursors()
		return nil
	}

	// Any non-internal cursor will turn attract mode off.
	if !state.Current.IsInternal() {
		TheAttractManager.setAttractMode(false)
	}

	// For the moment, the cursor to patchLogic mapping is 1-to-1.
	// I.e. ce.Source of "A" maps to patchLogic "A"
	patchLogic, ok := quadpro.patchLogic[state.Current.Source()]
	if !ok || patchLogic == nil {
		LogWarn("Source doesn't exist in patchLogic", "source", state.Current.Source())
		return nil
	}
	cursorStyle := patchLogic.patch.Get("misc.cursorstyle")
	gensound := IsTrueValue(patchLogic.patch.Get("misc.generatesound"))
	genvisual := IsTrueValue(patchLogic.patch.Get("misc.generatevisual"))
	if gensound && !TheAttractManager.attractModeIsOn {
		patchLogic.generateSoundFromCursor(state.Current, cursorStyle)
	}
	if genvisual {
		patchLogic.generateVisualsFromCursor(state.Current)
	}
	return nil
}

func (quadpro *QuadPro) onClientRestart(portnum int) {
	LogOfType("resolume", "quadpro got clientrestart", "portnum", portnum)
	// Refresh the patch that has that portnum
	for _, patch := range quadpro.patch {
		patch.RefreshAllIfPortnumMatches(portnum)
	}
}

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

func (quadpro *QuadPro) onMidiEvent(me MidiEvent) error {
	LogOfType("midi", "QuadPro.onMidiEvent", "me", me)
	return nil
}

func (quadpro *QuadPro) doTest(ntimes int, dt time.Duration) {
	LogInfo("doTest start", "ntimes", ntimes, "dt", dt)
	var state ActiveCursor
	for n := 0; n < ntimes; n++ {
		if n > 0 {
			time.Sleep(dt)
		}
		source := string("ABCD"[rand.Int()%4])
		cid := source + "#0"
		state.Previous = state.Current
		state.Current = CursorEvent{
			Cid:   cid,
			Click: CurrentClick(),
			Ddu:   "down",
			X:     rand.Float32(),
			Y:     rand.Float32(),
			Z:     rand.Float32(),
		}
		err := quadpro.onCursorEvent(state)
		if err != nil {
			LogError(err)
			return
		}
		time.Sleep(dt)
		state.Previous = state.Current
		state.Current = CursorEvent{
			Cid:   cid,
			Click: CurrentClick(),
			Ddu:   "up",
			X:     rand.Float32(),
			Y:     rand.Float32(),
			Z:     rand.Float32(),
		}
		err = quadpro.onCursorEvent(state)
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

func (quadpro *QuadPro) Load(category string, filename string) error {

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

func (quadpro *QuadPro) addPatch(name string) *Patch {
	patch := NewPatch(name)
	quadpro.patch[name] = patch
	quadpro.patchLogic[name] = NewPatchLogic(patch)
	return patch
}

/*
func (quadpro *QuadPro) scheduleNoteNow(dest string, pitch, velocity uint8, duration Clicks) {
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
