package samplesplitter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

const (
	audioSampleRate = 44100
	audioChannels   = 1
	audioFadeSec    = 0.008
)

type AudioManager struct {
	mu            sync.Mutex
	ffmpegPath    string
	state         *State
	context       *malgo.AllocatedContext
	device        *malgo.Device
	cache         map[string]*audioBuffer
	voices        map[string]*audioVoice
	channelVoices map[int]string
	channelOrder  map[int][]string
	channelActive map[int]string
	noteVoices    map[[2]int]string
	reverb        *monoReverb
	outputID      *int
	outputName    *string
	err           error
}

type AudioDevice struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default,omitempty"`
}

type audioBuffer struct {
	samples        []int16
	compressed     []int16
	compressedOnce sync.Once
}

type audioVoice struct {
	pcm      []int16
	position int
	loop     bool
	channel  int
	note     int
	active   bool
}

type monoReverb struct {
	wet               float64
	length            float64
	earlyTaps         []reverbTap
	combs             []reverbComb
	allpasses         []reverbAllpass
	tailFrames        int
	maxTailFrames     int
	baseMaxTailFrames int
}

type reverbTap struct {
	buffer []float64
	index  int
	gain   float64
}

type reverbComb struct {
	buffer       []float64
	index        int
	filterStore  float64
	feedback     float64
	baseFeedback float64
	damping      float64
}

type reverbAllpass struct {
	buffer   []float64
	index    int
	feedback float64
}

func newMonoReverb(sampleRate int) *monoReverb {
	scale := float64(sampleRate) / 44100.0
	scaled := func(samples int) int {
		n := int(math.Round(float64(samples) * scale))
		if n < 1 {
			return 1
		}
		return n
	}
	r := &monoReverb{
		earlyTaps: []reverbTap{
			newReverbTap(scaled(419), 0.24),
			newReverbTap(scaled(683), -0.18),
			newReverbTap(scaled(967), 0.15),
			newReverbTap(scaled(1277), -0.12),
			newReverbTap(scaled(1699), 0.09),
		},
		combs: []reverbComb{
			newReverbComb(scaled(1357), 0.38, 0.22),
			newReverbComb(scaled(1559), 0.41, 0.25),
			newReverbComb(scaled(1777), 0.36, 0.23),
			newReverbComb(scaled(2027), 0.40, 0.26),
			newReverbComb(scaled(2251), 0.37, 0.22),
			newReverbComb(scaled(2477), 0.39, 0.24),
		},
		allpasses: []reverbAllpass{
			newReverbAllpass(scaled(113), 0.25),
			newReverbAllpass(scaled(337), 0.25),
		},
		baseMaxTailFrames: sampleRate * 6 / 5,
	}
	r.SetLength(DefaultReverbLength)
	return r
}

func newReverbTap(delay int, gain float64) reverbTap {
	return reverbTap{
		buffer: make([]float64, delay),
		gain:   gain,
	}
}

func newReverbComb(delay int, feedback, damping float64) reverbComb {
	return reverbComb{
		buffer:       make([]float64, delay),
		feedback:     feedback,
		baseFeedback: feedback,
		damping:      damping,
	}
}

func newReverbAllpass(delay int, feedback float64) reverbAllpass {
	return reverbAllpass{
		buffer:   make([]float64, delay),
		feedback: feedback,
	}
}

func (r *monoReverb) SetWet(wet float64) {
	r.wet = clampReverbWet(wet)
	if r.wet == 0 {
		r.Reset()
	}
}

func (r *monoReverb) SetLength(length float64) {
	r.length = clampReverbLength(length)
	if r.baseMaxTailFrames <= 0 {
		r.baseMaxTailFrames = audioSampleRate * 6 / 5
	}
	r.maxTailFrames = int(math.Round(float64(r.baseMaxTailFrames) * r.length))
	if r.maxTailFrames < 1 {
		r.maxTailFrames = 1
	}
	if r.tailFrames > r.maxTailFrames {
		r.tailFrames = r.maxTailFrames
	}
	for i := range r.combs {
		baseFeedback := r.combs[i].baseFeedback
		if baseFeedback <= 0 {
			baseFeedback = r.combs[i].feedback
			r.combs[i].baseFeedback = baseFeedback
		}
		r.combs[i].feedback = math.Pow(baseFeedback, 1/r.length)
	}
}

func (r *monoReverb) Reset() {
	for i := range r.earlyTaps {
		clear(r.earlyTaps[i].buffer)
		r.earlyTaps[i].index = 0
	}
	for i := range r.combs {
		clear(r.combs[i].buffer)
		r.combs[i].index = 0
		r.combs[i].filterStore = 0
	}
	for i := range r.allpasses {
		clear(r.allpasses[i].buffer)
		r.allpasses[i].index = 0
	}
	r.tailFrames = 0
}

func (r *monoReverb) Active() bool {
	return r.wet > 0 && r.tailFrames > 0
}

func (r *monoReverb) Process(sample int) int {
	if r.wet <= 0 {
		return sample
	}

	input := float64(sample)
	if math.Abs(input) > 1 {
		r.tailFrames = r.maxTailFrames
	} else if r.tailFrames > 0 {
		r.tailFrames--
	}

	earlySignal := 0.0
	for i := range r.earlyTaps {
		earlySignal += r.earlyTaps[i].Process(input)
	}
	tailSignal := 0.0
	for i := range r.combs {
		tailSignal += r.combs[i].Process(input * 0.32)
	}
	diffuseSignal := tailSignal / 3.0
	for i := range r.allpasses {
		diffuseSignal = r.allpasses[i].Process(diffuseSignal)
	}
	wetSignal := earlySignal + diffuseSignal

	dryGain := 1.0 - r.wet*0.12
	wetGain := r.wet * 0.85
	mixed := input*dryGain + wetSignal*wetGain
	if mixed > 32767 {
		return 32767
	}
	if mixed < -32768 {
		return -32768
	}
	return int(math.Round(mixed))
}

func (t *reverbTap) Process(input float64) float64 {
	output := t.buffer[t.index] * t.gain
	t.buffer[t.index] = input
	t.index++
	if t.index >= len(t.buffer) {
		t.index = 0
	}
	return output
}

func (c *reverbComb) Process(input float64) float64 {
	output := c.buffer[c.index]
	c.filterStore = output*(1-c.damping) + c.filterStore*c.damping
	c.buffer[c.index] = input + c.filterStore*c.feedback
	c.index++
	if c.index >= len(c.buffer) {
		c.index = 0
	}
	return output
}

func (a *reverbAllpass) Process(input float64) float64 {
	buffered := a.buffer[a.index]
	output := buffered - input
	a.buffer[a.index] = input + buffered*a.feedback
	a.index++
	if a.index >= len(a.buffer) {
		a.index = 0
	}
	return output
}

func NewAudioManager(ffmpegPath string, state *State) (*AudioManager, error) {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	a := &AudioManager{
		ffmpegPath:    ffmpegPath,
		state:         state,
		context:       ctx,
		cache:         make(map[string]*audioBuffer),
		voices:        make(map[string]*audioVoice),
		channelVoices: make(map[int]string),
		channelOrder:  make(map[int][]string),
		channelActive: make(map[int]string),
		noteVoices:    make(map[[2]int]string),
		reverb:        newMonoReverb(audioSampleRate),
	}
	if state != nil {
		snapshot := state.Snapshot()
		length := snapshot.ReverbLength
		if length <= 0 {
			length = DefaultReverbLength
		}
		a.reverb.SetLength(length)
		a.reverb.SetWet(snapshot.ReverbWet)
	}
	if err := a.openDevice(nil); err != nil {
		ctx.Free()
		return nil, err
	}
	return a, nil
}

func (a *AudioManager) Devices() ([]AudioDevice, int, error) {
	if a == nil || a.context == nil {
		return nil, 0, errors.New("audio backend is not initialized")
	}
	infos, err := a.context.Devices(malgo.Playback)
	if err != nil {
		return nil, 0, err
	}
	devices := make([]AudioDevice, 0, len(infos))
	defaultID := 0
	for i, info := range infos {
		isDefault := info.IsDefault != 0
		if isDefault {
			defaultID = i
		}
		devices = append(devices, AudioDevice{
			ID:      i,
			Name:    info.Name(),
			Default: isDefault,
		})
	}
	return devices, defaultID, nil
}

func (a *AudioManager) SetOutput(id int) (string, error) {
	if a == nil || a.context == nil {
		return "", errors.New("audio backend is not initialized")
	}
	infos, err := a.context.Devices(malgo.Playback)
	if err != nil {
		return "", err
	}
	if id < 0 || id >= len(infos) {
		return "", fmt.Errorf("audio output device %d not found", id)
	}
	info := infos[id]
	if err := a.openDevice(&info.ID); err != nil {
		a.setErr(err)
		return "", err
	}
	name := info.Name()
	a.mu.Lock()
	a.outputID = &id
	a.outputName = &name
	a.mu.Unlock()
	if a.state != nil {
		a.state.SetAudioOutput(&id, &name)
	}
	return name, nil
}

func (a *AudioManager) Play(req *PlaybackRequest) error {
	if a == nil {
		return errors.New("audio backend is not initialized")
	}
	if req == nil {
		return errors.New("missing playback request")
	}
	if req.FilePath == "" {
		return errors.New("playback request has no file path")
	}

	buf, err := a.decode(req.FilePath)
	if err != nil {
		a.setErr(err)
		return err
	}
	pcm, err := renderSegment(buf, req.StartSec, req.EndSec, req.PitchRatio, req.Velocity, req.Compressed)
	if err != nil {
		a.setErr(err)
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	voiceKey := req.VoiceKey
	if voiceKey == "" {
		voiceKey = "preview"
	}
	a.ensureVoiceMapsLocked()
	a.stopVoiceLocked(voiceKey)
	voice := &audioVoice{pcm: pcm, loop: req.Loop, channel: req.Channel, note: req.Note, active: true}
	if req.Channel >= 0 {
		if activeKey, ok := a.channelActive[req.Channel]; ok {
			if activeVoice := a.voices[activeKey]; activeVoice != nil {
				activeVoice.active = false
				activeVoice.position = 0
			}
		}
		a.channelVoices[req.Channel] = voiceKey
		a.channelActive[req.Channel] = voiceKey
		a.noteVoices[[2]int{req.Channel, req.Note}] = voiceKey
		a.channelOrder[req.Channel] = appendVoiceKey(a.channelOrder[req.Channel], voiceKey)
	}

	a.voices[voiceKey] = voice
	a.syncActiveVoicesLocked()
	a.err = nil
	return nil
}

func (a *AudioManager) Preload(paths []string) error {
	if a == nil {
		return errors.New("audio backend is not initialized")
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, err := a.decode(path); err != nil {
			a.setErr(err)
			return err
		}
	}
	a.setErr(nil)
	return nil
}

func (a *AudioManager) PreloadCompressed(paths []string) error {
	if a == nil {
		return errors.New("audio backend is not initialized")
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		buf, err := a.decode(path)
		if err != nil {
			a.setErr(err)
			return err
		}
		_ = buf.compressedSamples()
	}
	a.setErr(nil)
	return nil
}

func (a *AudioManager) SetReverbWet(wet float64) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensureReverbLocked()
	a.reverb.SetWet(wet)
}

func (a *AudioManager) SetReverbLength(length float64) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensureReverbLocked()
	a.reverb.SetLength(length)
}

func (a *AudioManager) ClearCache() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[string]*audioBuffer)
}

func (a *AudioManager) StopVoice(voiceKey string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopVoiceLocked(voiceKey)
}

func (a *AudioManager) StopNote(channel, note int) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	before := len(a.voices)
	if note < 0 {
		a.stopChannelLocked(channel)
		if before != len(a.voices) && a.state != nil {
			a.state.SetAudioStatus(true, nil)
		}
		return
	}
	key := [2]int{channel, note}
	if voiceKey, ok := a.noteVoices[key]; ok {
		a.stopVoiceLocked(voiceKey)
		delete(a.noteVoices, key)
		if a.state != nil {
			a.state.SetAudioStatus(true, nil)
		}
		return
	}
}

func (a *AudioManager) StopAll() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for voiceKey := range a.voices {
		a.stopVoiceLocked(voiceKey)
	}
	a.channelVoices = make(map[int]string)
	a.channelOrder = make(map[int][]string)
	a.channelActive = make(map[int]string)
	a.noteVoices = make(map[[2]int]string)
	a.syncActiveVoicesLocked()
}

func (a *AudioManager) Err() error {
	if a == nil {
		return errors.New("audio backend is not initialized")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.err
}

func (a *AudioManager) Close() {
	if a == nil {
		return
	}
	a.StopAll()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.device != nil {
		a.device.Uninit()
		a.device = nil
	}
	if a.context != nil {
		_ = a.context.Uninit()
		a.context.Free()
		a.context = nil
	}
}

func (a *AudioManager) openDevice(deviceID *malgo.DeviceID) error {
	a.mu.Lock()
	if a.device != nil {
		a.device.Uninit()
		a.device = nil
	}
	a.mu.Unlock()

	config := malgo.DefaultDeviceConfig(malgo.Playback)
	config.SampleRate = audioSampleRate
	config.Playback.Format = malgo.FormatS16
	config.Playback.Channels = audioChannels
	config.PerformanceProfile = malgo.LowLatency
	var pinner runtime.Pinner
	if deviceID != nil {
		pinner.Pin(deviceID)
		defer pinner.Unpin()
		config.Playback.DeviceID = unsafe.Pointer(deviceID)
	}
	device, err := malgo.InitDevice(a.context.Context, config, malgo.DeviceCallbacks{
		Data: a.dataCallback,
	})
	if err != nil {
		return err
	}
	if err := device.Start(); err != nil {
		device.Uninit()
		return err
	}

	a.mu.Lock()
	a.device = device
	a.mu.Unlock()
	return nil
}

func (a *AudioManager) dataCallback(output, _ []byte, _ uint32) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range output {
		output[i] = 0
	}
	if len(a.voices) == 0 && !a.reverbActiveLocked() {
		return
	}
	completed := a.mixIntoLocked(output)
	for _, voiceKey := range completed {
		a.stopVoiceLocked(voiceKey)
	}
}

func (a *AudioManager) mixIntoLocked(output []byte) []string {
	frameCount := len(output) / 2
	completed := make([]string, 0)
	// Reuse this map for the whole device callback. Allocating it inside the
	// frame loop created one map per audio sample (44,100 allocations/second),
	// causing enough GC pressure to interfere with the graphics process while
	// sample playback was active.
	mixedVoiceKeys := make(map[string]bool, len(a.voices))
	for frame := 0; frame < frameCount; frame++ {
		mixed := 0
		clear(mixedVoiceKeys)
		for voiceKey, voice := range a.voices {
			if mixedVoiceKeys[voiceKey] || (voice.loop && voice.channel >= 0 && !voice.active) {
				continue
			}
			if len(voice.pcm) == 0 {
				completed = append(completed, voiceKey)
				continue
			}
			if voice.position >= len(voice.pcm) {
				if voice.loop {
					voiceKey, voice = a.advanceChannelVoiceLocked(voice.channel, voiceKey)
					if voice == nil {
						completed = append(completed, voiceKey)
						continue
					}
					if mixedVoiceKeys[voiceKey] {
						continue
					}
				} else {
					completed = append(completed, voiceKey)
					continue
				}
			}
			mixed += int(voice.pcm[voice.position])
			voice.position++
			mixedVoiceKeys[voiceKey] = true
		}
		if mixed > 32767 {
			mixed = 32767
		} else if mixed < -32768 {
			mixed = -32768
		}
		if a.reverb != nil {
			mixed = a.reverb.Process(mixed)
		}
		binary.LittleEndian.PutUint16(output[frame*2:frame*2+2], uint16(int16(mixed)))
	}
	return uniqueStrings(completed)
}

func (a *AudioManager) setErr(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.err = err
}

func (a *AudioManager) ensureVoiceMapsLocked() {
	if a.voices == nil {
		a.voices = make(map[string]*audioVoice)
	}
	if a.channelVoices == nil {
		a.channelVoices = make(map[int]string)
	}
	if a.channelOrder == nil {
		a.channelOrder = make(map[int][]string)
	}
	if a.channelActive == nil {
		a.channelActive = make(map[int]string)
	}
	if a.noteVoices == nil {
		a.noteVoices = make(map[[2]int]string)
	}
}

func (a *AudioManager) ensureReverbLocked() {
	if a.reverb == nil {
		a.reverb = newMonoReverb(audioSampleRate)
	}
}

func (a *AudioManager) reverbActiveLocked() bool {
	return a.reverb != nil && a.reverb.Active()
}

func (a *AudioManager) stopChannelLocked(channel int) {
	a.ensureVoiceMapsLocked()
	for key, voiceKey := range a.noteVoices {
		if key[0] == channel {
			delete(a.voices, voiceKey)
			delete(a.noteVoices, key)
		}
	}
	delete(a.channelVoices, channel)
	delete(a.channelOrder, channel)
	delete(a.channelActive, channel)
	prefix := fmt.Sprintf("midi-%d-", channel)
	for voiceKey := range a.voices {
		if strings.HasPrefix(voiceKey, prefix) {
			delete(a.voices, voiceKey)
		}
	}
	a.syncActiveVoicesLocked()
}

func (a *AudioManager) stopVoiceLocked(voiceKey string) {
	a.ensureVoiceMapsLocked()
	voice, hadVoice := a.voices[voiceKey]
	delete(a.voices, voiceKey)
	for key, activeVoice := range a.noteVoices {
		if activeVoice == voiceKey {
			delete(a.noteVoices, key)
		}
	}
	if hadVoice && voice.channel >= 0 {
		a.removeChannelVoiceLocked(voice.channel, voiceKey)
	} else {
		for channel, activeVoice := range a.channelActive {
			if activeVoice == voiceKey {
				a.removeChannelVoiceLocked(channel, voiceKey)
			}
		}
	}
	a.syncActiveVoicesLocked()
}

func (a *AudioManager) removeChannelVoiceLocked(channel int, voiceKey string) {
	order := removeVoiceKey(a.channelOrder[channel], voiceKey)
	if len(order) == 0 {
		delete(a.channelOrder, channel)
		delete(a.channelActive, channel)
		delete(a.channelVoices, channel)
		return
	}
	a.channelOrder[channel] = order
	if a.channelActive[channel] != voiceKey {
		return
	}
	nextKey := order[0]
	a.channelActive[channel] = nextKey
	a.channelVoices[channel] = nextKey
	if nextVoice := a.voices[nextKey]; nextVoice != nil {
		nextVoice.active = true
		nextVoice.position = 0
	}
}

func (a *AudioManager) advanceChannelVoiceLocked(channel int, currentKey string) (string, *audioVoice) {
	if channel < 0 {
		if voice := a.voices[currentKey]; voice != nil {
			voice.position = 0
			return currentKey, voice
		}
		return currentKey, nil
	}
	order := a.channelOrder[channel]
	if len(order) == 0 {
		if voice := a.voices[currentKey]; voice != nil {
			voice.position = 0
			return currentKey, voice
		}
		return currentKey, nil
	}
	currentIdx := -1
	for i, key := range order {
		if key == currentKey {
			currentIdx = i
			break
		}
	}
	if currentIdx < 0 {
		currentIdx = 0
	}
	for offset := 1; offset <= len(order); offset++ {
		nextKey := order[(currentIdx+offset)%len(order)]
		nextVoice := a.voices[nextKey]
		if nextVoice == nil || len(nextVoice.pcm) == 0 {
			continue
		}
		if currentVoice := a.voices[currentKey]; currentVoice != nil {
			currentVoice.active = false
			currentVoice.position = 0
		}
		nextVoice.active = true
		nextVoice.position = 0
		a.channelActive[channel] = nextKey
		a.channelVoices[channel] = nextKey
		return nextKey, nextVoice
	}
	return currentKey, nil
}

func (a *AudioManager) syncActiveVoicesLocked() {
	if a.state == nil {
		return
	}
	voices := make([]string, 0, len(a.voices))
	for voiceKey := range a.voices {
		voices = append(voices, voiceKey)
	}
	sort.Strings(voices)
	a.state.SetActiveVoices(voices)
}

func (a *AudioManager) decode(path string) (*audioBuffer, error) {
	a.mu.Lock()
	if cached, ok := a.cache[path]; ok {
		a.mu.Unlock()
		return cached, nil
	}
	a.mu.Unlock()

	cmd := exec.Command(a.ffmpegPath, "-v", "error", "-i", path, "-f", "s16le", "-acodec", "pcm_s16le", "-ac", "1", "-ar", fmt.Sprint(audioSampleRate), "pipe:1")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("ffmpeg decode failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffmpeg decode failed: %w", err)
	}
	if len(output)%2 != 0 {
		output = output[:len(output)-1]
	}
	samples := make([]int16, len(output)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(output[i*2 : i*2+2]))
	}
	buf := &audioBuffer{samples: samples}

	a.mu.Lock()
	a.cache[path] = buf
	a.mu.Unlock()
	return buf, nil
}

func renderSegment(buf *audioBuffer, startSec, endSec, pitchRatio float64, velocity int, compressed bool) ([]int16, error) {
	if buf == nil || len(buf.samples) == 0 {
		return nil, errors.New("empty audio buffer")
	}
	sourceSamples := buf.samples
	if compressed {
		sourceSamples = buf.compressedSamples()
	}
	if pitchRatio <= 0 {
		pitchRatio = 1
	}
	if velocity <= 0 {
		velocity = 110
	}
	startIdx := max(0, min(len(sourceSamples), int(startSec*audioSampleRate)))
	endIdx := max(startIdx+1, min(len(sourceSamples), int(endSec*audioSampleRate)))
	if startIdx >= len(sourceSamples) || startIdx >= endIdx {
		return nil, fmt.Errorf("invalid segment %.4f..%.4f", startSec, endSec)
	}

	outSamples := max(1, int(float64(endIdx-startIdx)/pitchRatio))
	volume := math.Min(1, math.Max(0, float64(velocity)/127.0))
	fadeSec := audioFadeSec
	fadeSamples := min(outSamples/2, int(fadeSec*float64(audioSampleRate)))
	pcm := make([]int16, outSamples)
	for i := 0; i < outSamples; i++ {
		sourceIdx := startIdx + min(endIdx-startIdx-1, int(float64(i)*pitchRatio))
		sample := float64(sourceSamples[sourceIdx])
		if fadeSamples > 0 {
			if i < fadeSamples {
				sample *= float64(i) / float64(fadeSamples)
			}
			if remaining := outSamples - 1 - i; remaining < fadeSamples {
				sample *= float64(remaining) / float64(fadeSamples)
			}
		}
		pcm[i] = int16(math.Max(-32768, math.Min(32767, sample)))
	}
	if volume < 1 {
		for i, sample := range pcm {
			scaled := float64(sample) * volume
			pcm[i] = int16(math.Max(-32768, math.Min(32767, scaled)))
		}
	}
	return pcm, nil
}

func (buf *audioBuffer) compressedSamples() []int16 {
	if buf == nil {
		return nil
	}
	buf.compressedOnce.Do(func() {
		buf.compressed = compressAndNormalizeCopy(buf.samples)
	})
	if len(buf.compressed) == 0 {
		return buf.samples
	}
	return buf.compressed
}

func compressAndNormalizeCopy(pcm []int16) []int16 {
	if len(pcm) == 0 {
		return nil
	}
	const (
		threshold = 0.35
		ratio     = 4.0
		target    = 0.92
	)
	peak := 0.0
	values := make([]float64, len(pcm))
	for i, sample := range pcm {
		value := float64(sample) / 32768.0
		sign := 1.0
		if value < 0 {
			sign = -1
			value = -value
		}
		if value > threshold {
			value = threshold + (value-threshold)/ratio
		}
		value *= sign
		values[i] = value
		if abs := math.Abs(value); abs > peak {
			peak = abs
		}
	}
	if peak <= 0 {
		return append([]int16(nil), pcm...)
	}
	out := make([]int16, len(pcm))
	gain := target / peak
	for i, value := range values {
		scaled := value * gain * 32767.0
		out[i] = int16(math.Max(-32768, math.Min(32767, scaled)))
	}
	return out
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	sort.Strings(values)
	out := values[:0]
	var last string
	for i, value := range values {
		if i == 0 || value != last {
			out = append(out, value)
			last = value
		}
	}
	return out
}

func appendVoiceKey(values []string, voiceKey string) []string {
	for _, value := range values {
		if value == voiceKey {
			return values
		}
	}
	return append(values, voiceKey)
}

func removeVoiceKey(values []string, voiceKey string) []string {
	out := values[:0]
	for _, value := range values {
		if value != voiceKey {
			out = append(out, value)
		}
	}
	return out
}
