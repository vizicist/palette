package samplesplitter

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"runtime"
	"sort"
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
	noteVoices    map[[2]int]string
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
	samples []int16
}

type audioVoice struct {
	pcm      []int16
	position int
	loop     bool
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
		noteVoices:    make(map[[2]int]string),
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
	pcm, err := renderSegment(buf, req.StartSec, req.EndSec, req.PitchRatio, req.Velocity)
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
	if req.Channel >= 0 {
		a.stopChannelLocked(req.Channel)
		a.channelVoices[req.Channel] = voiceKey
		a.noteVoices[[2]int{req.Channel, req.Note}] = voiceKey
	}
	a.stopVoiceLocked(voiceKey)

	a.voices[voiceKey] = &audioVoice{pcm: pcm, loop: req.Loop}
	a.syncActiveVoicesLocked()
	a.err = nil
	return nil
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
	key := [2]int{channel, note}
	if voiceKey, ok := a.noteVoices[key]; ok {
		a.stopVoiceLocked(voiceKey)
		delete(a.noteVoices, key)
		return
	}
	a.stopChannelLocked(channel)
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
	if len(a.voices) == 0 {
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
	for frame := 0; frame < frameCount; frame++ {
		mixed := 0
		for voiceKey, voice := range a.voices {
			if len(voice.pcm) == 0 {
				completed = append(completed, voiceKey)
				continue
			}
			if voice.position >= len(voice.pcm) {
				if voice.loop {
					voice.position = 0
				} else {
					completed = append(completed, voiceKey)
					continue
				}
			}
			mixed += int(voice.pcm[voice.position])
			voice.position++
		}
		if mixed > 32767 {
			mixed = 32767
		} else if mixed < -32768 {
			mixed = -32768
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

func (a *AudioManager) stopChannelLocked(channel int) {
	if voiceKey, ok := a.channelVoices[channel]; ok {
		a.stopVoiceLocked(voiceKey)
		delete(a.channelVoices, channel)
	}
	for key, voiceKey := range a.noteVoices {
		if key[0] == channel {
			a.stopVoiceLocked(voiceKey)
			delete(a.noteVoices, key)
		}
	}
}

func (a *AudioManager) stopVoiceLocked(voiceKey string) {
	delete(a.voices, voiceKey)
	for channel, activeVoice := range a.channelVoices {
		if activeVoice == voiceKey {
			delete(a.channelVoices, channel)
		}
	}
	for key, activeVoice := range a.noteVoices {
		if activeVoice == voiceKey {
			delete(a.noteVoices, key)
		}
	}
	a.syncActiveVoicesLocked()
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

func renderSegment(buf *audioBuffer, startSec, endSec, pitchRatio float64, velocity int) ([]int16, error) {
	if buf == nil || len(buf.samples) == 0 {
		return nil, errors.New("empty audio buffer")
	}
	if pitchRatio <= 0 {
		pitchRatio = 1
	}
	if velocity <= 0 {
		velocity = 110
	}
	startIdx := max(0, min(len(buf.samples), int(startSec*audioSampleRate)))
	endIdx := max(startIdx+1, min(len(buf.samples), int(endSec*audioSampleRate)))
	if startIdx >= len(buf.samples) || startIdx >= endIdx {
		return nil, fmt.Errorf("invalid segment %.4f..%.4f", startSec, endSec)
	}

	outSamples := max(1, int(float64(endIdx-startIdx)/pitchRatio))
	volume := math.Min(1, math.Max(0, float64(velocity)/127.0))
	fadeSec := audioFadeSec
	fadeSamples := min(outSamples/2, int(fadeSec*float64(audioSampleRate)))
	pcm := make([]int16, outSamples)
	for i := 0; i < outSamples; i++ {
		sourceIdx := startIdx + min(endIdx-startIdx-1, int(float64(i)*pitchRatio))
		sample := float64(buf.samples[sourceIdx]) * volume
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
	return pcm, nil
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
