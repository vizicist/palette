package kit

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"

	"github.com/hypebeast/go-osc/osc"
)

var resolumePort = 7000

// resolumeRESTTimeout is per request. Opening a clip makes Resolume read the
// file header, which is slower than the OSC messages elsewhere in the engine.
const resolumeRESTTimeout = 10 * time.Second

var resolumeRESTClient = &http.Client{Timeout: resolumeRESTTimeout}

func resolumeRESTPort() int {
	if GlobalParams == nil {
		return 8080
	}
	port, err := GetParamInt("global.resolumerestport")
	if err != nil {
		LogIfError(err)
		port = 8080 // Resolume's default webserver port
	}
	return port
}

// resolumeREST sends one request to Resolume's REST API and returns the body of
// a 2xx response. Anything else - including Resolume not running, or its
// webserver being switched off - comes back as an error for the caller to
// report once.
//
// The REST API exists alongside OSC because OSC can only address things the
// composition already contains: opening a file into a clip has no OSC
// equivalent.
func resolumeREST(method string, apipath string, contentType string, body string) ([]byte, error) {

	fullURL := fmt.Sprintf("http://%s:%d/api/v1%s", LocalAddress, resolumeRESTPort(), apipath)

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, fullURL, reader)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := resolumeRESTClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		LogIfError(resp.Body.Close())
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("resolume REST %s %s returned %s: %s",
			method, apipath, resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

// resolumeFileURL converts an absolute path to the file:/// URL Resolume
// expects: forward slashes and percent-encoded special characters, even on
// Windows, where C:\a b\c.mp4 becomes file:///C:/a%20b/c.mp4.
func resolumeFileURL(path string) string {
	slashed := strings.TrimPrefix(filepath.ToSlash(path), "/")
	u := url.URL{Scheme: "file", Path: "/" + slashed}
	return u.String()
}

type Resolume struct {
	resolumeClient   *osc.Client
	freeframeClients map[string]*osc.Client
	effectsJSON      map[string]any // unmarshalled resolume.json
}

var (
	theResolume     *Resolume
	theResolumeOnce sync.Once
)

// TheResolume returns the one Resolume client. Construction is behind a
// sync.Once: the attract tick and the API goroutine both reach it now that
// attract videos drive layers from either side, and two clients would each
// carry their own freeframeClients map.
func TheResolume() *Resolume {
	theResolumeOnce.Do(func() {
		theResolume = &Resolume{
			resolumeClient:   osc.NewClient(LocalAddress, resolumePort),
			freeframeClients: map[string]*osc.Client{},
		}

		// _ = theResolume.bypassLayer // to avoid unused error

		err := theResolume.loadResolumeJSON()
		if err != nil {
			LogIfError(err)
		}
	})
	return theResolume
}

func (r *Resolume) loadResolumeJSON() error {
	path := ConfigFilePath("resolume.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to read resolume.json, err=%w", err)
	}
	var j map[string]any
	err = json.Unmarshal(bytes, &j)
	if err != nil {
		return fmt.Errorf("unable to Unmarshal %s", path)
	}
	r.effectsJSON = j
	return nil
}

func (r *Resolume) PortAndLayerNumForPatch(patchName string) (portNum, layerNum int) {
	switch patchName {
	case "A":
		return 3334, 1
	case "B":
		return 3335, 2
	case "C":
		return 3336, 3
	case "D":
		return 3337, 4
	default:
		LogError(fmt.Errorf("no port for layer"), "patchName", patchName)
		return 0, 0
	}
}

func (r *Resolume) freeframeClientFor(patchName string) *osc.Client {
	ff, ok := r.freeframeClients[patchName]
	if !ok {
		portNum, _ := r.PortAndLayerNumForPatch(patchName)
		if portNum == 0 {
			return nil
		}
		ff = osc.NewClient(LocalAddress, portNum)
		r.freeframeClients[patchName] = ff
	}
	return ff
}

func (r *Resolume) ToFreeFramePlugin(patchName string, msg *osc.Message) {
	LogOfType("freeframe", "Resolume.toFreeframe", "patch", patchName, "msg", msg)
	ff := r.freeframeClientFor(patchName)
	if ff == nil {
		LogIfError(fmt.Errorf("no freeframe client for layer"), "patch", patchName)
		return
	}
	LogOfType("ffgl", "toFreeFramePlugin", "patch", patchName, "msg", msg)
	theEngine.SendOsc(ff, msg)
}

func (r *Resolume) SendEffectParam(patchName string, name string, value string) {
	portNum, layerNum := r.PortAndLayerNumForPatch(patchName)
	if portNum == 0 {
		LogIfError(fmt.Errorf("no such layer"), "name", patchName)
		return
	}
	// Effect parameters that have ":" in their name are plugin parameters
	i := strings.Index(name, ":")
	if i > 0 {
		effectName := name[0:i]
		paramName := name[i+1:]
		r.sendPadOneEffectParam(layerNum, effectName, paramName, value)
	} else {
		onoff, err := strconv.ParseBool(value)
		if err != nil {
			LogIfError(err)
			onoff = false
		}
		r.sendPadOneEffectOnOff(layerNum, name, onoff)
	}
}

func (r *Resolume) sendPadOneEffectParam(layerNum int, effectName string, paramName string, value string) {
	fullName := "effect" + "." + effectName + ":" + paramName
	paramsMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, "params")
	if err != nil {
		LogIfError(err)
		return
	}
	if paramsMap == nil {
		LogWarn("No params value for", "effecdt", effectName)
		return
	}
	oneParam, ok := paramsMap[paramName]
	if !ok {
		LogWarn("No params value for", "param", paramName, "effect", effectName)
		return
	}

	oneDef, ok := ParamDefs[fullName]
	if !ok {
		LogWarn("No paramdef value for", "param", paramName, "effect", effectName)
		return
	}

	addr, ok := oneParam.(string)
	if !ok {
		LogWarn("resolume.json: param addr is not a string", "param", paramName, "effect", effectName)
		return
	}
	resEffectName := resolumeEffectNameOf(realEffectName, realEffectNum)
	addr = strings.Replace(addr, realEffectName, resEffectName, 1)
	addr = addLayerAndClipNums(addr, layerNum, 1)

	msg := osc.NewMessage(addr)

	// Append the value to the message, depending on the type of the parameter

	switch oneDef.TypedParamDef.(type) {

	case ParamDefInt:
		valint, err := strconv.Atoi(value)
		if err != nil {
			LogIfError(err)
			valint = 0
		}
		msg.Append(int32(valint))

	case ParamDefBool:
		valbool, err := strconv.ParseBool(value)
		if err != nil {
			LogIfError(err)
			valbool = false
		}
		onoffValue := 0
		if valbool {
			onoffValue = 1
		}
		msg.Append(int32(onoffValue))

	case ParamDefString:
		valstr := value
		msg.Append(valstr)

	case ParamDefFloat:
		valfloat, err := ParseFloat(value, resEffectName)
		if err != nil {
			LogIfError(err)
			valfloat = 0.0
		}
		msg.Append(float32(valfloat))

	default:
		LogWarn("sendPadOneEffectParam: unknown type of ParamDef for", "name", fullName)
		return
	}

	r.toResolume(msg)
}

func (r *Resolume) toResolume(msg *osc.Message) {
	theEngine.SendOsc(r.resolumeClient, msg)
}

func (r *Resolume) sendPadOneEffectOnOff(layerNum int, effectName string, onoff bool) {
	var mapType string
	if onoff {
		mapType = "on"
	} else {
		mapType = "off"
	}

	onoffMap, realEffectName, realEffectNum, err := r.getEffectMap(effectName, mapType)
	if err != nil {
		LogIfError(err)
		return
	}

	if onoffMap == nil {
		LogWarn("No onoffMap value for", "effect", effectName, "maptype", mapType, effectName)
		return
	}

	onoffAddr, ok := onoffMap["addr"]
	if !ok {
		LogWarn("No addr value in onoff", "effect", effectName)
		return
	}
	onoffArg, ok := onoffMap["arg"]
	if !ok {
		LogWarn("No arg valuei in onoff for", "effect", effectName)
		return
	}
	addr, ok := onoffAddr.(string)
	if !ok {
		LogWarn("resolume.json: onoff addr is not a string", "effect", effectName)
		return
	}
	onoffFloat, ok := onoffArg.(float64)
	if !ok {
		LogWarn("resolume.json: onoff arg is not a number", "effect", effectName)
		return
	}
	addr = r.addEffectNum(addr, realEffectName, realEffectNum)
	addr = addLayerAndClipNums(addr, layerNum, 1)
	onoffValue := int(onoffFloat)

	msg := osc.NewMessage(addr)
	msg.Append(int32(onoffValue))
	r.toResolume(msg)
}

func (r *Resolume) addEffectNum(addr string, effect string, num int) string {
	if num == 1 {
		return addr
	}
	// e.g. "blur" becomes "blur2"
	return strings.Replace(addr, effect, fmt.Sprintf("%s%d", effect, num), 1)
}

func (r *Resolume) showText(text string) {

	textLayerNum := r.TextLayerNum()

	// make sure the layer is not displayed before changing it
	r.bypassLayer(textLayerNum, true)

	// the first clip is the textgenerator clip
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/video/source/textgenerator/text/params/lines", textLayerNum)
	msg := osc.NewMessage(addr)
	text = strings.Replace(text, "_", "\n", 1)
	msg.Append(text)
	theEngine.SendOsc(r.resolumeClient, msg)

	// give it time to "sink in", otherwise the previous text displays briefly
	time.Sleep(150 * time.Millisecond)

	r.connectClip(textLayerNum, 1)     // activate that clip
	r.bypassLayer(textLayerNum, false) // show the layer
}

// In text layer, clip 1 is the animated text generator for the preset names,
// and clips 2,3,... are images for startup and reboot.
func (r *Resolume) showTextLayerClip(clipNum int) {

	textLayerNum := r.TextLayerNum()
	r.connectClip(textLayerNum, clipNum) // activate that clip
}

func (r *Resolume) TextLayerNum() int {
	layerNum, err := GetParamInt("global.resolumetextlayer")
	if err != nil {
		LogIfError(err)
		layerNum = 5 // last resort
	}
	return layerNum
}

// Resolume serves REST calls while it is still opening a composition: the
// window appears first, and the layers arrive over the next several seconds.
const (
	resolumeCompositionPollInterval = 500 * time.Millisecond
	resolumeCompositionTimeout      = 30 * time.Second
)

// resolumeLayerCount returns how many layers the composition currently has.
func resolumeLayerCount() (int, error) {

	data, err := resolumeREST("GET", "/composition", "", "")
	if err != nil {
		return 0, err
	}
	var composition struct {
		Layers []json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(data, &composition); err != nil {
		return 0, fmt.Errorf("unable to parse Resolume composition: %w", err)
	}
	return len(composition.Layers), nil
}

// waitForResolumeComposition blocks until the composition has at least
// minLayers layers, which is how the engine tells that Resolume has finished
// opening it.
//
// Acting on the first answer instead is wrong in both directions: it misses
// layers that are about to exist - the engine logged "text layer is not in the
// composition, numlayers=3" against a five layer composition doing exactly
// that - and it would let the attract videos add layers to a composition that
// is still growing, leaving too many.
func waitForResolumeComposition(minLayers int, timeout time.Duration) (int, error) {

	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		numLayers, err := resolumeLayerCount()
		switch {
		case err != nil:
			lastErr = err
		case numLayers >= minLayers:
			return numLayers, nil
		default:
			lastErr = fmt.Errorf("composition has %d layers, waiting for %d", numLayers, minLayers)
		}
		if time.Now().After(deadline) {
			return 0, lastErr
		}
		time.Sleep(resolumeCompositionPollInterval)
	}
}

// textLayerSplashImages maps a clip number on the text layer to the parameter
// naming the image that belongs in it.
//
// Clip 1 is deliberately absent. It holds the text generator that showText
// writes preset names into, which is a Resolume source rather than a file, and
// whose font, size and colour are set up in the composition - rebuilding it
// would throw that away to fix a problem it doesn't have.
var textLayerSplashImages = map[int]string{
	2: "global.resolumestartingupimage",
	3: "global.resolumerebootingimage",
	4: "global.resolumerestartingimage",
}

// textLayerSplashClips resolves those parameters to files in the config
// directory. A parameter left empty is skipped silently, and one naming a file
// that isn't there is skipped with a warning, so an installation that has only
// some of the images still gets those.
func textLayerSplashClips() map[int]string {

	clips := map[int]string{}
	if GlobalParams == nil {
		return clips
	}

	for clipNum, paramName := range textLayerSplashImages {
		fileName := GetParamWithDefault(paramName, "")
		if fileName == "" {
			continue
		}
		path := ConfigFilePath(fileName)
		if !FileExists(path) {
			LogWarn("splash image named by parameter is not in the config directory",
				"param", paramName, "file", fileName)
			continue
		}
		clips[clipNum] = path
	}
	return clips
}

// buildTextLayer constructs the text layer's splash clips - starting up,
// rebooting, restarting - from this machine's config directory, the same way
// the attract videos construct their own layer.
//
// The compositions ship with absolute paths baked in, and neither survives
// being installed anywhere else: PaletteDefault.avc has a literal
// "%LOCALAPPDATA%\Palette\..." that Resolume never expands, and
// PaletteDefaultSP.avc points into a developer's home directory. Resolume shows
// the media as offline and waits for someone to relocate each file by hand.
//
// Building the clips outright, rather than repairing whatever the composition
// happens to contain, means it doesn't matter what state the .avc on disk is
// in: an unsaved composition, or one saved on a different machine, comes up
// correct anyway because the clips are opened afresh on every activation.
func (r *Resolume) buildTextLayer() {

	layerNum := r.TextLayerNum()

	numLayers, err := waitForResolumeComposition(layerNum, resolumeCompositionTimeout)
	if err != nil {
		LogWarn("text layer not built, Resolume composition unavailable",
			"layer", layerNum, "err", err)
		return
	}

	clips := textLayerSplashClips()
	if len(clips) == 0 {
		return
	}

	clipNums := make([]int, 0, len(clips))
	for clipNum := range clips {
		clipNums = append(clipNums, clipNum)
	}
	sort.Ints(clipNums)

	built := 0
	for _, clipNum := range clipNums {
		path := clips[clipNum]
		openPath := fmt.Sprintf("/composition/layers/%d/clips/%d/open", layerNum, clipNum)
		if _, err := resolumeREST("POST", openPath, "text/plain", resolumeFileURL(path)); err != nil {
			LogWarn("unable to build splash clip",
				"clip", clipNum, "file", filepath.Base(path), "err", err)
			continue
		}
		built++
		LogOfType("resolume", "built splash clip", "clip", clipNum, "file", filepath.Base(path))
	}
	LogInfo("text layer built", "layer", layerNum, "clips", built, "numlayers", numLayers)
}

func (r *Resolume) ProcessInfo() *ProcessInfo {
	fullpath, err := GetParam("global.resolumepath")
	LogIfError(err)
	if fullpath == "" || !FileExists(fullpath) {
		// try other hardcoded paths
		hardcoded := []string{
			"C:/Program Files/Resolume Avenue/Avenue.exe",
			"C:/Program Files/Resolume Arena/Arena.exe",
		}
		if runtime.GOOS == "darwin" {
			hardcoded = []string{
				"/Applications/Resolume Avenue/Avenue.app/Contents/MacOS/Avenue",
				"/Applications/Resolume Arena/Arena.app/Contents/MacOS/Arena",
			}
		}
		fullpath = ""
		for _, path := range hardcoded {
			if FileExists(path) {
				fullpath = path
				break
			}
		}
		if fullpath == "" {
			LogWarn("No Resolume found, set global.resolumepath to the full path to Resolume")
			return EmptyProcessInfo()
		}
	}
	exe := filepath.Base(fullpath)
	return NewProcessInfo(exe, fullpath, "", r.Activate)
}

func (r *Resolume) Activate() {

	LogInfo("Activating Resolume")

	// Get max wait time from parameter (reuse resolumeactivate as max attempts)
	maxAttempts, err := GetParamInt("global.resolumeactivate")
	if err != nil {
		LogIfError(err)
		maxAttempts = 24 // fallback to default
	}

	// Wait for Resolume window to appear before sending activation OSC
	windowFound := false
	for i := 0; !windowFound && i < maxAttempts; i++ {
		time.Sleep(5 * time.Second)
		LogInfo("Checking for Resolume window", "attempt", i+1)
		if FindWindowByTitleContains("resolume") {
			LogInfo("Resolume window detected", "attempt", i+1)
			windowFound = true
			break
		}
		LogOfType("resolume", "Waiting for Resolume window", "attempt", i+1, "of", maxAttempts)
	}

	if !windowFound {
		LogInfo("Resolume window not detected after max attempts, activating anyway")
	}

	time.Sleep(5 * time.Second)

	// Build the splash clips from this machine's config directory before showing
	// one, otherwise the "starting up" clip is offline media and shows nothing.
	// This waits for Resolume to finish opening the composition.
	r.buildTextLayer()

	LogInfo("Sending showClip 2 OSC to Resolume")
	r.showTextLayerClip(2) // show the "starting up" splash clip while waiting

	// Clear a solo left behind by an engine that was killed while attract
	// videos were playing. Without this, Resolume comes up showing only that
	// layer, with nothing on screen to explain why.
	r.soloLayer(attractVideoLayerNum(), false)

	// Activate all layers a few times to make sure it takes
	for i := 0; i < 12; i++ {
		for _, patch := range PatchNames() {
			_, layerNum := r.PortAndLayerNumForPatch(string(patch))
			LogOfType("resolume", "Activating Resolume", "patch", layerNum, "attempt", i+1)
			r.connectClip(layerNum, 1)
		}
		time.Sleep(10 * time.Second)
		r.showTextLayerClip(1) // gets rid of the "starting up" clip
	}

	// Show the animated text generator for preset names
	r.showTextLayerClip(1)

	// This Resolume is freshly started, so its composition has none of the
	// attract-video clips or the layer they were pushed into - those are only
	// ever put into the running composition, never saved. Tell the player to
	// forget them, so the next attract entry rebuilds them rather than
	// addressing clips that no longer exist. Done last, once the composition
	// has settled, and it restores the videos itself if attract mode is already
	// running - which is likely, since Resolume restarting takes long enough
	// for the idle timer to elapse.
	TheAttractVideoPlayer().Reload()
}

func (r *Resolume) connectClip(layerNum int, clip int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/%d/connect", layerNum, clip)
	msg := osc.NewMessage(addr)
	// Note: sending 0 doesn't seem to disable a clip; you need to
	// bypass the layer to turn it off
	msg.Append(int32(1))
	theEngine.SendOsc(r.resolumeClient, msg)
}

func (r *Resolume) bypassLayer(layerNum int, onoff bool) {
	addr := fmt.Sprintf("/composition/layers/%d/bypassed", layerNum)
	msg := osc.NewMessage(addr)
	v := 0
	if onoff {
		v = 1
	}
	msg.Append(int32(v))
	theEngine.SendOsc(r.resolumeClient, msg)
}

// soloLayer makes a layer the only one that reaches the output, and is how
// attract videos cover the patch layers completely. Turning solo off restores
// whatever the other layers were doing, without having to remember their
// individual bypass states.
func (r *Resolume) soloLayer(layerNum int, onoff bool) {
	addr := fmt.Sprintf("/composition/layers/%d/solo", layerNum)
	msg := osc.NewMessage(addr)
	v := 0
	if onoff {
		v = 1
	}
	msg.Append(int32(v))
	theEngine.SendOsc(r.resolumeClient, msg)
}

// setLayerOpacity sets a layer's video opacity, 0.0 to 1.0. Resolume creates
// new layers at half opacity, so a layer the engine added has to be told to be
// opaque before anything on it will cover what is underneath.
func (r *Resolume) setLayerOpacity(layerNum int, opacity float32) {
	addr := fmt.Sprintf("/composition/layers/%d/video/opacity", layerNum)
	msg := osc.NewMessage(addr)
	msg.Append(opacity)
	theEngine.SendOsc(r.resolumeClient, msg)
}

// getEffectMap returns the resolume.json map for a given effect
// and map type ("on", "off", or "params")
func (r *Resolume) getEffectMap(effectName string, mapType string) (map[string]any, string, int, error) {
	if len(effectName) < 2 || effectName[1] != '-' {
		err := fmt.Errorf("no dash in effect, name=%s", effectName)
		return nil, "", 0, err
	}
	effects, ok := r.effectsJSON["effects"]
	if !ok {
		err := fmt.Errorf("no effects value in resolume.json?")
		return nil, "", 0, err
	}
	realEffectName := effectName[2:]

	n, err := strconv.Atoi(effectName[0:1])
	if err != nil {
		return nil, "", 0, fmt.Errorf("bad format of effectName=%s", effectName)
	}
	effnum := int(n)

	effectsmap, err := jsonMap(effects, "resolume.json effects")
	if err != nil {
		return nil, "", 0, err
	}
	oneEffect, ok := effectsmap[realEffectName]
	if !ok {
		err := fmt.Errorf("no effects value for effect=%s", effectName)
		return nil, "", 0, err
	}
	oneEffectMap, err := jsonMap(oneEffect, "resolume.json effect "+realEffectName)
	if err != nil {
		return nil, "", 0, err
	}
	mapValue, ok := oneEffectMap[mapType]
	if !ok {
		err := fmt.Errorf("no params value for effect=%s", effectName)
		return nil, "", 0, err
	}
	resultMap, err := jsonMap(mapValue, "resolume.json "+realEffectName+"."+mapType)
	if err != nil {
		return nil, "", 0, err
	}
	return resultMap, realEffectName, effnum, nil
}

func addLayerAndClipNums(addr string, layerNum int, clipNum int) string {
	if addr[0] != '/' {
		LogWarn("addr in resolume.json doesn't start with /", "addr", addr)
		addr = "/" + addr
	}
	addr = fmt.Sprintf("/composition/layers/%d/clips/%d/video/effects%s", layerNum, clipNum, addr)
	return addr
}

func resolumeEffectNameOf(name string, num int) string {
	if num == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, num)
}
