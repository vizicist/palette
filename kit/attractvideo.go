package kit

import (
	"fmt"
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

// The videos play in one of two places, chosen by
// global.attractvideodestination:
//
//	main - the Resolume output, the projection the audience sees (the default,
//	       and everything above about clips and layers describes this one)
//	gui  - the browser on the GUI screen, which plays the files itself
//
// These are two different code paths rather than two ways of driving Resolume,
// because Resolume renders to its own output and nothing here can point that at
// the GUI screen. The GUI screen is a browser, though, and a browser already
// knows how to play video: "gui" serves this same directory over the engine's
// HTTP server (/attractvideos/) and the page plays it in a <video> element. So
// the GUI destination needs no Resolume at all - not even its webserver, which
// is the part of the "main" path most likely to be switched off.
//
// The engine does almost nothing for the GUI destination. It reports where the
// videos belong in the UI status snapshot and lists the files for
// global.attractvideolist; the browser does the playing and the advancing.
const (
	attractVideoDestMain = "main"
	attractVideoDestGUI  = "gui"
)

// normalizeAttractVideoDestination folds anything unrecognized onto "main", so
// a typo in the parameter leaves the videos where they have always been rather
// than making them disappear from both places.
func normalizeAttractVideoDestination(dest string) string {
	if strings.EqualFold(strings.TrimSpace(dest), attractVideoDestGUI) {
		return attractVideoDestGUI
	}
	return attractVideoDestMain
}

// AttractVideoDestination is where attract videos play right now.
func AttractVideoDestination() string {
	if GlobalParams == nil {
		return attractVideoDestMain
	}
	return normalizeAttractVideoDestination(
		GetParamWithDefault("global.attractvideodestination", attractVideoDestMain))
}

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

type AttractVideoPlayer struct {
	mutex      sync.Mutex
	files      []string  // absolute paths, sorted
	durations  []float64 // seconds, parallel to files
	loaded     bool      // clips have been pushed into Resolume
	playing    bool
	current    int    // index into files of the clip now showing
	layer      int    // the layer Start put the videos on
	dest       string // the destination Start actually used
	nextSwitch time.Time
	warned     bool // REST failure already logged; don't repeat it every time
}

var (
	theAttractVideoPlayer     *AttractVideoPlayer
	theAttractVideoPlayerOnce sync.Once
)

// TheAttractVideoPlayer returns the one player. The construction is behind a
// sync.Once because the callers genuinely race: the attract tick calls Advance
// while the API goroutine turns attract mode (or global.attractvideos) on and
// off. Note that "go TheAttractVideoPlayer().Start()" evaluates the receiver in
// the calling goroutine, so those call sites do not serialize it. Two players
// would mean Start filling in one while Advance and Stop read the other, and
// videos that come up and then never advance or stop.
func TheAttractVideoPlayer() *AttractVideoPlayer {
	theAttractVideoPlayerOnce.Do(func() {
		theAttractVideoPlayer = &AttractVideoPlayer{}
	})
	return theAttractVideoPlayer
}

// AttractVideoDir is the optional directory of videos played during attract
// mode. Nothing creates it; an installation that wants videos adds it by hand.
func AttractVideoDir() string {
	return filepath.Join(ConfigDir(), attractVideoDirName)
}

// attractVideosEnabled gates the whole feature. It defaults to true so an
// installation that has the directory gets the videos without configuring
// anything, and turning it off leaves the directory in place but unused - which
// is easier than moving hundreds of megabytes out of the way for one gig.
func attractVideosEnabled() bool {
	// Playing videos needs a running engine and loaded parameters: every OSC
	// message here goes out through theEngine, and the layer and port come from
	// parameters. Attract mode can be driven without either - it is, in tests -
	// and a half-started engine has no business opening clips, let alone
	// reaching the network to do it.
	if theEngine == nil || GlobalParams == nil {
		return false
	}
	return IsTrueValue(GetParamWithDefault("global.attractvideos", "true"))
}

func attractVideoLayerNum() int {
	// Resolume startup can ask for this before the parameters are loaded.
	if GlobalParams == nil {
		return 6
	}
	layerNum, err := GetParamInt("global.attractvideolayer")
	if err != nil {
		LogIfError(err)
		layerNum = 6 // last resort, matching the paramdef's init
	}
	return layerNum
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

	// Wait for Resolume to finish opening the composition before counting what
	// is in it. It answers REST calls while still loading, and layers added to a
	// composition that is still growing leave more layers than the composition
	// is meant to have. The text layer existing is the signal that the shipped
	// composition has arrived.
	numLayers, err := waitForResolumeComposition(TheResolume().TextLayerNum(), resolumeCompositionTimeout)
	if err != nil {
		return err
	}

	for ; numLayers < layerNum; numLayers++ {
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
		if _, err := resolumeREST("POST", openPath, "text/plain", resolumeFileURL(file)); err != nil {
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
		fileURLs[i] = resolumeFileURL(file)
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

	if AttractVideoDestination() == attractVideoDestGUI {
		p.startGUI(files)
		return
	}
	p.startMain(files)
}

// startGUI records that the videos are playing on the GUI screen. There is no
// work to do: the browser learns from the status snapshot that attract mode is
// on and that the videos belong here, and plays them itself.
//
// What it deliberately does not touch is the files/loaded/durations bookkeeping,
// which is only ever about what has been pushed into Resolume. Recording a GUI
// run's file list there would make a later switch back to "main" believe those
// files were already loaded as clips, and skip the load.
func (p *AttractVideoPlayer) startGUI(files []string) {
	p.current = 0
	p.playing = true
	p.dest = attractVideoDestGUI
	LogInfo("attract videos playing on the GUI screen", "count", len(files))
}

// startMain puts the videos on the Resolume output. The caller holds the mutex.
func (p *AttractVideoPlayer) startMain(files []string) {

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
	// Remember the layer rather than re-reading the parameter later. Stop has to
	// take down the layer it actually put up: changing global.attractvideolayer
	// mid-show would otherwise leave the old layer soloed, which is the stuck
	// state Resolume.Activate already has to clear on startup. The destination
	// is remembered for the same reason - Stop must not send un-solo and bypass
	// for a run that never touched Resolume, nor skip them for one that did.
	p.layer = layerNum
	p.dest = attractVideoDestMain

	// Resolume creates layers at half opacity, which let the patch layers show
	// through the video. Set it every time rather than once at creation: the
	// layer may have been made on another machine, or by hand.
	TheResolume().setLayerOpacity(layerNum, 1.0)
	TheResolume().connectClip(layerNum, 1)
	TheResolume().bypassLayer(layerNum, false)
	// Solo so nothing else reaches the output at all. Opacity alone isn't
	// enough: a video that doesn't fill the frame, or one with an alpha
	// channel, would still let the patch layers show around or through it.
	// Soloing last, once the layer is already up, avoids a frame of black.
	TheResolume().soloLayer(layerNum, true)

	p.nextSwitch = time.Now().Add(p.durationOf(0))

	LogInfo("attract videos playing", "count", len(p.files), "layer", layerNum)
}

// Reload forgets that the clips were pushed into Resolume, and puts them back
// up if attract mode is running right now.
//
// Everything this player sets up - the clips, and the extra layer ensureLayer
// adds - lives only in Resolume's running composition. None of it is saved to
// the .avc, so a Resolume that has restarted has neither. Start skips its
// reload when it believes the clips are already loaded, so without this the
// player would go on sending connect and solo OSC to a layer and clip numbers
// that no longer exist, and the attract videos would silently never appear
// again for the rest of the engine's uptime.
func (p *AttractVideoPlayer) Reload() {

	p.mutex.Lock()
	p.loaded = false
	p.playing = false
	p.mutex.Unlock()

	// If attract mode is on right now, Start puts the videos back. If it isn't,
	// the next entry into attract mode does, and reloads because loaded is now
	// false.
	if theAttractManager != nil && theAttractManager.AttractModeIsOn() {
		p.Start()
	}
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
	if p.dest == attractVideoDestGUI {
		// Nothing was ever sent to Resolume, so there is nothing to take down.
		// The browser stops its own playback when the status snapshot says
		// attract mode is over.
		LogInfo("attract videos stopped on the GUI screen")
		return
	}
	layerNum := p.layer
	// Un-solo first. The video is still opaque and on top, so nothing changes
	// on screen until the layer is bypassed - one clean transition back to
	// normal, rather than a frame of black or of blended patch graphics.
	TheResolume().soloLayer(layerNum, false)
	TheResolume().bypassLayer(layerNum, true)
	LogInfo("attract videos stopped")
}

// Restart takes the videos down from wherever they are and puts them back up at
// whichever destination the parameter now names. It is what a change to
// global.attractvideodestination does while attract mode is already running.
//
// Stopping first is the point, not tidiness: Stop undoes the destination that
// Start actually used, so switching away from "main" un-solos the Resolume
// layer. Without that the layer would stay soloed with a paused video on it and
// nothing but a Resolume restart would clear it, while the GUI played the same
// files underneath.
func (p *AttractVideoPlayer) Restart() {
	p.Stop()
	p.Start()
}

// Advance moves to the next video once the current one has played through,
// wrapping at the end. It runs from the attract-mode tick rather than a timer
// of its own, so it stops as soon as attract mode does, and it only sends OSC -
// the REST work all happened in Start.
func (p *AttractVideoPlayer) Advance() {

	// Start can hold this mutex while Resolume finishes loading a composition
	// and answers REST requests. Advance runs synchronously on the scheduler
	// tick, so waiting here would stall every scheduled event. Missing one
	// advance check is harmless; the next tick will retry.
	if !p.mutex.TryLock() {
		return
	}
	defer p.mutex.Unlock()

	if !p.playing {
		return
	}
	if p.dest == attractVideoDestGUI {
		// The browser moves to the next file when the current one ends, which
		// is both more accurate than a duration we asked Resolume for and the
		// only option here - there is nothing to send.
		return
	}
	if len(p.files) == 0 {
		return
	}
	if time.Now().Before(p.nextSwitch) {
		return
	}

	p.current = (p.current + 1) % len(p.files)
	TheResolume().connectClip(p.layer, p.current+1)
	p.nextSwitch = time.Now().Add(p.durationOf(p.current))

	LogOfType("attract", "attract video advance",
		"clip", p.current+1, "file", filepath.Base(p.files[p.current]))
}

// AttractVideoPlaylist is everything the GUI screen needs to play the videos
// itself: where they are meant to play, and the names to ask for under
// /attractvideos/. Names rather than paths - the browser has no business
// knowing where the directory is, and the file server resolves them relative to
// it.
//
// Files is empty whenever the videos are switched off or the directory holds
// nothing playable, so the browser has one thing to test rather than having to
// re-derive the rules that attractVideoFiles already applies.
type AttractVideoPlaylist struct {
	Destination string   `json:"destination"`
	Files       []string `json:"files"`
}

// AttractVideoListJSON answers global.attractvideolist.
func AttractVideoListJSON() (string, error) {

	// An empty slice rather than nil, so this marshals as [] and the browser
	// never has to distinguish "no videos" from "null".
	playlist := AttractVideoPlaylist{
		Destination: AttractVideoDestination(),
		Files:       []string{},
	}
	if attractVideosEnabled() {
		for _, file := range attractVideoFiles() {
			playlist.Files = append(playlist.Files, filepath.Base(file))
		}
	}

	b, err := json.Marshal(playlist)
	if err != nil {
		return "", fmt.Errorf("AttractVideoListJSON: %w", err)
	}
	return string(b), nil
}

func (p *AttractVideoPlayer) durationOf(i int) time.Duration {
	secs := fallbackVideoSecs
	if i < len(p.durations) && p.durations[i] > 0 {
		secs = p.durations[i]
	}
	return time.Duration(secs * float64(time.Second))
}
