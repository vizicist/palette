package samplesplitter

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
)

type Server struct {
	State     *State
	Analyzer  Analyzer
	StaticDir string
	MIDI      *MIDIManager
	Audio     *AudioManager
}

func ServerFromService(service *Service, staticDir string) Server {
	if service == nil {
		return Server{StaticDir: staticDir}
	}
	if staticDir == "" {
		staticDir = ResolveStaticDir("")
	}
	return Server{
		State:     service.state,
		Analyzer:  service.analyzer,
		StaticDir: staticDir,
		MIDI:      service.midi,
		Audio:     service.audio,
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/media", s.handleMedia)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/analyze", s.handleAnalyze)
	mux.HandleFunc("/api/midi_ports", s.handleMIDIPorts)
	mux.HandleFunc("/api/audio_outputs", s.handleAudioOutputs)
	mux.HandleFunc("/api/set_midi", s.handleSetMIDI)
	mux.HandleFunc("/api/set_audio_output", s.handleSetAudioOutput)
	mux.HandleFunc("/api/set_base_note", s.handleSetBaseNote)
	mux.HandleFunc("/api/set_peak_start", s.handleSetPeakStart)
	mux.HandleFunc("/api/set_compressed", s.handleSetCompressed)
	mux.HandleFunc("/api/reload_sigils", s.handleReloadSigils)
	mux.HandleFunc("/api/set_pitch_bend", s.handleSetPitchBend)
	mux.HandleFunc("/api/stop_all", s.handleStopAll)
	mux.HandleFunc("/api/preview_on", s.handlePreviewOn)
	mux.HandleFunc("/api/preview_off", s.handlePreviewOff)
	return mux
}

func (s Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.StaticDir, "index.html"))
}

func (s Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	files, err := ListMP3Files(s.State.Config.MP3Dir)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.Name
	}
	writeJSON(w, map[string]any{"files": names, "dir": s.State.Config.MP3Dir}, http.StatusOK)
}

func (s Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	path, err := ResolveMP3File(s.State.Config.MP3Dir, r.URL.Query().Get("file"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	http.ServeFile(w, r, path)
}

func (s Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.State.Snapshot(), http.StatusOK)
}

func (s Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path, err := ResolveMP3File(s.State.Config.MP3Dir, q.Get("file"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	opts := DefaultAnalyzeOptions()
	if mode := q.Get("mode"); mode != "" {
		opts.Mode = mode
	}
	opts.Interval = parseFloat(q.Get("interval"), opts.Interval)
	opts.SilenceThreshold = parseFloat(q.Get("silence_thresh"), opts.SilenceThreshold)
	opts.SilenceMinimum = parseFloat(q.Get("silence_min"), opts.SilenceMinimum)
	opts.WordsPerSplit = parseInt(q.Get("words_per_split"), opts.WordsPerSplit)

	cue, waveform, err := s.Analyzer.AnalyzeFile(path, opts)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	s.State.SetCurrent(path, cue, waveform)
	writeJSON(w, map[string]any{"cue_data": cue, "waveform": waveform}, http.StatusOK)
}

func (s Server) handleMIDIPorts(w http.ResponseWriter, r *http.Request) {
	ports, err := s.midiPorts()
	if err != nil {
		s.State.SetMIDIStatus("", err)
		writeJSON(w, map[string]any{
			"ports":   []string{},
			"current": s.State.Snapshot().MIDIPort,
			"error":   err.Error(),
		}, http.StatusOK)
		return
	}
	writeJSON(w, map[string]any{
		"ports":   ports,
		"current": s.State.Snapshot().MIDIPort,
		"error":   nil,
	}, http.StatusOK)
}

func (s Server) handleAudioOutputs(w http.ResponseWriter, r *http.Request) {
	if s.Audio == nil {
		writeJSON(w, map[string]any{
			"devices":      []AudioDevice{},
			"default":      nil,
			"current":      nil,
			"current_name": nil,
			"error":        "audio backend is not initialized",
		}, http.StatusOK)
		return
	}
	devices, defaultID, err := s.Audio.Devices()
	if err != nil {
		writeJSON(w, map[string]any{
			"devices":      []AudioDevice{},
			"default":      nil,
			"current":      nil,
			"current_name": nil,
			"error":        err.Error(),
		}, http.StatusOK)
		return
	}
	snapshot := s.State.Snapshot()
	writeJSON(w, map[string]any{
		"devices":      devices,
		"default":      defaultID,
		"current":      snapshot.AudioOutputID,
		"current_name": snapshot.AudioOutputName,
		"error":        nil,
	}, http.StatusOK)
}

func (s Server) handleSetMIDI(w http.ResponseWriter, r *http.Request) {
	if s.MIDI == nil {
		writeError(w, errors.New("MIDI backend is not initialized"), http.StatusNotImplemented)
		return
	}
	port := r.URL.Query().Get("port")
	if port == "" {
		writeError(w, errors.New("missing port"), http.StatusBadRequest)
		return
	}
	resolved, err := s.MIDI.Start(port)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "midi_port": resolved}, http.StatusOK)
}

func (s Server) handleSetAudioOutput(w http.ResponseWriter, r *http.Request) {
	if s.Audio == nil {
		writeError(w, errors.New("audio backend is not initialized"), http.StatusNotImplemented)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		writeError(w, errors.New("bad audio output id"), http.StatusBadRequest)
		return
	}
	name, err := s.Audio.SetOutput(id)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "id": id, "name": name}, http.StatusOK)
}

func (s Server) handleSetBaseNote(w http.ResponseWriter, r *http.Request) {
	note, err := strconv.Atoi(r.URL.Query().Get("note"))
	if err != nil {
		writeError(w, errors.New("bad note"), http.StatusBadRequest)
		return
	}
	s.State.SetBaseNote(note)
	writeJSON(w, map[string]any{"ok": true, "base_note": note}, http.StatusOK)
}

func (s Server) handleSetPeakStart(w http.ResponseWriter, r *http.Request) {
	enabled := parseBool(r.URL.Query().Get("enabled"))
	s.State.SetPeakStart(enabled)
	writeJSON(w, map[string]any{"ok": true, "peak_start_enabled": enabled}, http.StatusOK)
}

func (s Server) handleSetCompressed(w http.ResponseWriter, r *http.Request) {
	enabled := parseBool(r.URL.Query().Get("enabled"))
	s.State.SetCompressed(enabled)
	writeJSON(w, map[string]any{"ok": true, "compressed": enabled}, http.StatusOK)
}

func (s Server) handleReloadSigils(w http.ResponseWriter, r *http.Request) {
	if s.State == nil {
		writeError(w, errors.New("samplesplitter state is not initialized"), http.StatusNotImplemented)
		return
	}
	s.State.SetBusy(true, "Loading transmissions")
	defer s.State.SetBusy(false, "")
	if s.Audio != nil {
		s.Audio.StopAll()
		s.Audio.ClearCache()
	}
	s.State.LoadSigilDefaults(s.Analyzer, nil)
	s.State.LoadFirstIfEmpty(s.Analyzer)
	if s.Audio != nil {
		var err error
		if s.State.Snapshot().Compressed {
			err = s.Audio.PreloadCompressed(s.State.StartupSamplePaths())
		} else {
			err = s.Audio.Preload(s.State.StartupSamplePaths())
		}
		if err != nil {
			s.State.SetAudioStatus(false, err)
			writeError(w, err, http.StatusInternalServerError)
			return
		}
		s.State.SetAudioStatus(true, nil)
	}
	writeJSON(w, s.State.Snapshot(), http.StatusOK)
}

func (s Server) handleSetPitchBend(w http.ResponseWriter, r *http.Request) {
	semitones := parseFloat(r.URL.Query().Get("semitones"), 0)
	if semitones < -12 {
		semitones = -12
	} else if semitones > 12 {
		semitones = 12
	}
	s.State.SetPitchBendSemitones(-1, semitones)
	writeJSON(w, map[string]any{"ok": true, "pitch_bend_semitones": semitones}, http.StatusOK)
}

func (s Server) handleStopAll(w http.ResponseWriter, r *http.Request) {
	if s.Audio != nil {
		s.Audio.StopAll()
	}
	writeJSON(w, map[string]any{"ok": true}, http.StatusOK)
}

func (s Server) handlePreviewOn(w http.ResponseWriter, r *http.Request) {
	if s.Audio == nil {
		writeError(w, errors.New("audio backend is not initialized"), http.StatusNotImplemented)
		return
	}
	index, err := strconv.Atoi(r.URL.Query().Get("index"))
	if err != nil {
		writeError(w, errors.New("bad index"), http.StatusBadRequest)
		return
	}
	velocity := parseInt(r.URL.Query().Get("velocity"), 110)
	req, err := s.State.PlanPreview(index, r.URL.Query().Get("voice"), velocity)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if err := s.Audio.Play(req); err != nil {
		s.State.SetAudioStatus(false, err)
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	s.State.SetAudioStatus(true, nil)
	writeJSON(w, map[string]any{"ok": true, "index": index, "voice": req.VoiceKey}, http.StatusOK)
}

func (s Server) handlePreviewOff(w http.ResponseWriter, r *http.Request) {
	if s.Audio != nil {
		voice := r.URL.Query().Get("voice")
		if voice == "" {
			voice = "preview"
		}
		s.Audio.StopVoice(voice)
	}
	writeJSON(w, map[string]any{"ok": true}, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, data any, status int) {
	body, err := json.Marshal(data)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeError(w http.ResponseWriter, err error, status int) {
	writeJSON(w, map[string]string{"error": err.Error()}, status)
}

func parseFloat(value string, fallback float64) float64 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseBool(value string) bool {
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (s Server) midiPorts() ([]string, error) {
	if s.MIDI == nil {
		return nil, errors.New("MIDI backend is not initialized")
	}
	return s.MIDI.Ports()
}
