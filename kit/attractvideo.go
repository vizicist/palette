package kit

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"
)

// Attract-mode videos are an optional per-installation extra. When
// config/attractmode_videos exists and contains video files, attract mode plays
// them on the Resolume output on a layer above everything else, so an idle
// installation shows video rather than only the generated graphics. The
// directory is deliberately left out of the installer (see build_data.bat):
// the files are large and whatever plays at one venue is rarely right for
// another.
//
// Resolume's OSC API can only trigger clips that already exist in the
// composition, so getting a file from disk into a clip goes through Resolume's
// REST API. Everything after loading - connecting a clip, showing and hiding
// the layer - reuses the OSC path the rest of the engine already uses.
const attractVideoDirName = "attractmode_videos"

// attractVideoExtensions keeps stray files (thumbnails, notes, .DS_Store) out
// of the playlist. It is not a claim about what Resolume can decode; Resolume
// decides that when the clip is opened.
var attractVideoExtensions = map[string]bool{
	".mp4":  true,
	".mov":  true,
	".avi":  true,
	".mkv":  true,
	".webm": true,
	".m4v":  true,
	".mpg":  true,
	".mpeg": true,
}

// fallbackVideoSecs is how long a clip stays up when Resolume could not tell us
// the real duration, so a failed /file-info degrades to a slideshow instead of
// freezing on the first video forever.
const fallbackVideoSecs = 30.0

// resolumeRESTTimeout is per request. Opening a clip makes Resolume read the
// file header, which is slower than the OSC messages elsewhere in the engine.
const resolumeRESTTimeout = 10 * time.Second

type AttractVideoPlayer struct {
	mutex      sync.Mutex
	files      []string  // absolute paths, sorted
	durations  []float64 // seconds, parallel to files
	loaded     bool      // clips have been pushed into Resolume
	playing    bool
	current    int // index into files of the clip now showing
	nextSwitch time.Time
	warned     bool // REST failure already logged; don't repeat it every time
}

var theAttractVideoPlayer *AttractVideoPlayer

func TheAttractVideoPlayer() *AttractVideoPlayer {
	if theAttractVideoPlayer == nil {
		theAttractVideoPlayer = &AttractVideoPlayer{}
	}
	return theAttractVideoPlayer
}

// AttractVideoDir is the optional directory of videos played during attract
// mode. Nothing creates it; an installation that wants videos adds it by hand.
func AttractVideoDir() string {
	return filepath.Join(ConfigDir(), attractVideoDirName)
}

var resolumeRESTClient = &http.Client{Timeout: resolumeRESTTimeout}

// attractVideosEnabled gates the whole feature. It defaults to true so an
// installation that has the directory gets the videos without configuring
// anything, and turning it off leaves the directory in place but unused - which
// is easier than moving hundreds of megabytes out of the way for one gig.
func attractVideosEnabled() bool {
	// Attract mode can be driven before the parameters are loaded (and is, in
	// tests), and a half-started engine has no business talking to Resolume.
	if GlobalParams == nil {
		return false
	}
	return IsTrueValue(GetParamWithDefault("global.attractvideos", "true"))
}

func attractVideoLayerNum() int {
	layerNum, err := GetParamInt("global.attractvideolayer")
	if err != nil {
		LogIfError(err)
		layerNum = 6 // last resort, matching the paramdef's init
	}
	return layerNum
}

func resolumeRESTPort() int {
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

// attractVideoFileURL converts an absolute path to the file:/// URL Resolume
// expects: forward slashes and percent-encoded special characters, even on
// Windows, where C:\a b\c.mp4 becomes file:///C:/a%20b/c.mp4.
func attractVideoFileURL(path string) string {
	slashed := strings.TrimPrefix(filepath.ToSlash(path), "/")
	u := url.URL{Scheme: "file", Path: "/" + slashed}
	return u.String()
}

// attractVideoFiles lists the videos to play, sorted by name so the order on
// screen matches the order in the directory listing. A missing directory is the
// normal case and returns nothing without complaint.
func attractVideoFiles() []string {

	dir := AttractVideoDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			LogWarn("unable to read attract video directory", "dir", dir, "err", err)
		}
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !attractVideoExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files
}

// ensureLayer grows the composition until layerNum exists. Resolume adds new
// layers on top, which is where the attract videos belong: they cover the four
// patch layers and the text layer while they play.
func (p *AttractVideoPlayer) ensureLayer(layerNum int) error {

	data, err := resolumeREST("GET", "/composition", "", "")
	if err != nil {
		return err
	}
	var composition struct {
		Layers []json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(data, &composition); err != nil {
		return fmt.Errorf("unable to parse Resolume composition: %w", err)
	}

	for numLayers := len(composition.Layers); numLayers < layerNum; numLayers++ {
		if _, err := resolumeREST("POST", "/composition/layers/add", "text/plain", ""); err != nil {
			return fmt.Errorf("unable to add Resolume layer: %w", err)
		}
		LogInfo("added Resolume layer for attract videos", "layer", numLayers+1)
	}
	return nil
}

// loadClips pushes each video into a clip on the video layer, one clip per
// file. The layer is cleared first so videos removed from the directory don't
// linger as clips from an earlier run.
func (p *AttractVideoPlayer) loadClips(layerNum int) error {

	if err := p.ensureLayer(layerNum); err != nil {
		return err
	}

	clearPath := fmt.Sprintf("/composition/layers/%d/clearclips", layerNum)
	if _, err := resolumeREST("POST", clearPath, "text/plain", ""); err != nil {
		return err
	}

	for i, file := range p.files {
		clipNum := i + 1
		openPath := fmt.Sprintf("/composition/layers/%d/clips/%d/open", layerNum, clipNum)
		if _, err := resolumeREST("POST", openPath, "text/plain", attractVideoFileURL(file)); err != nil {
			return fmt.Errorf("unable to load %s into clip %d: %w", filepath.Base(file), clipNum, err)
		}
		LogOfType("resolume", "loaded attract video", "clip", clipNum, "file", filepath.Base(file))
	}
	return nil
}

// loadDurations asks Resolume how long each video is so each one plays through
// before the next starts. Any file it can't measure keeps the fallback, so one
// odd file doesn't stall the rotation.
func (p *AttractVideoPlayer) loadDurations() []float64 {

	durations := make([]float64, len(p.files))
	for i := range durations {
		durations[i] = fallbackVideoSecs
	}

	fileURLs := make([]string, len(p.files))
	for i, file := range p.files {
		fileURLs[i] = attractVideoFileURL(file)
	}
	body, err := json.Marshal(fileURLs)
	if err != nil {
		LogIfError(err)
		return durations
	}

	data, err := resolumeREST("POST", "/file-info", "application/json", string(body))
	if err != nil {
		LogWarn("unable to get attract video durations, using fallback",
			"secs", fallbackVideoSecs, "err", err)
		return durations
	}

	var infos []struct {
		Video *struct {
			DurationMS float64 `json:"duration_ms"`
		} `json:"video"`
	}
	if err := json.Unmarshal(data, &infos); err != nil {
		LogWarn("unable to parse Resolume file-info", "err", err)
		return durations
	}

	for i, info := range infos {
		if i >= len(durations) {
			break
		}
		if info.Video != nil && info.Video.DurationMS > 0 {
			durations[i] = info.Video.DurationMS / 1000.0
		}
	}
	return durations
}

// Start puts the first video up. It is called when attract mode turns on, from
// a goroutine, because loading clips talks to Resolume over HTTP and must not
// block the attract-mode tick.
func (p *AttractVideoPlayer) Start() {

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !attractVideosEnabled() {
		return
	}

	files := attractVideoFiles()
	if len(files) == 0 {
		// No directory, or nothing playable in it: attract mode behaves exactly
		// as it did before this existed.
		return
	}

	layerNum := attractVideoLayerNum()

	// Reload only when the directory contents changed, so turning attract mode
	// on and off doesn't re-push every file to Resolume.
	if !p.loaded || !slices.Equal(files, p.files) {
		p.files = files
		if err := p.loadClips(layerNum); err != nil {
			if !p.warned {
				LogWarn("attract videos unavailable - is Resolume running with its webserver enabled? (Preferences > Webserver)",
					"dir", AttractVideoDir(), "err", err)
				p.warned = true
			}
			p.loaded = false
			return
		}
		p.durations = p.loadDurations()
		p.loaded = true
		p.warned = false
	}

	// Loading can take a moment. If attract mode was turned off while we were
	// talking to Resolume - someone put a hand on the pads - don't put the
	// video up after the fact.
	if theAttractManager != nil && !theAttractManager.AttractModeIsOn() {
		return
	}

	p.current = 0
	p.playing = true
	TheResolume().connectClip(layerNum, 1)
	TheResolume().bypassLayer(layerNum, false)
	p.nextSwitch = time.Now().Add(p.durationOf(0))

	LogInfo("attract videos playing", "count", len(p.files), "layer", layerNum)
}

// Stop hides the video layer. Bypassing the layer is what takes the video off
// screen; as connectClip notes, sending a clip 0 doesn't turn a layer off.
func (p *AttractVideoPlayer) Stop() {

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.playing {
		return
	}
	p.playing = false
	TheResolume().bypassLayer(attractVideoLayerNum(), true)
	LogInfo("attract videos stopped")
}

// Advance moves to the next video once the current one has played through,
// wrapping at the end. It runs from the attract-mode tick rather than a timer
// of its own, so it stops as soon as attract mode does, and it only sends OSC -
// the REST work all happened in Start.
func (p *AttractVideoPlayer) Advance() {

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.playing || len(p.files) == 0 {
		return
	}
	if time.Now().Before(p.nextSwitch) {
		return
	}

	p.current = (p.current + 1) % len(p.files)
	TheResolume().connectClip(attractVideoLayerNum(), p.current+1)
	p.nextSwitch = time.Now().Add(p.durationOf(p.current))

	LogOfType("attract", "attract video advance",
		"clip", p.current+1, "file", filepath.Base(p.files[p.current]))
}

func (p *AttractVideoPlayer) durationOf(i int) time.Duration {
	secs := fallbackVideoSecs
	if i < len(p.durations) && p.durations[i] > 0 {
		secs = p.durations[i]
	}
	return time.Duration(secs * float64(time.Second))
}
