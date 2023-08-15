package kit

/*
func (e *Engine) StartRecording() (string, error) {
	fpath, err := e.NewRecordingPath()
	if err != nil {
		return "", err
	}
	f, err := os.Create(fpath)
	if err != nil {
		return "", err
	}
	e.recordingFile = f
	e.recordingPath = fpath
	e.RecordStartEvent()
	LogInfo("startrecording", "fpath", fpath)
	return fpath, nil
}

func (e *Engine) StopRecording() (string, error) {
	if e.recordingFile == nil {
		return "", fmt.Errorf("executeengineapi: not recording")
	}
	e.RecordStopEvent()
	LogInfo("stoprecording", "recordingPath", e.recordingPath)
	e.recordingFile.Close()
	e.recordingFile = nil
	e.recordingPath = ""
	return "", nil
}

func (e *Engine) StartPlayback(fname string) error {
	fpath := e.RecordingPath(fname)
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	go e.doPlayback(f)
	return nil
}

func (e *Engine) doPlayback(f *os.File) {
	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)
	LogInfo("doPlayback start")
	for fileScanner.Scan() {
		var rec RecordingEvent
		err := json.Unmarshal(fileScanner.Bytes(), &rec)
		if err != nil {
			LogIfError(err)
			continue
		}
		LogInfo("Playback", "rec", rec)
		switch rec.Event {
		case "cursor":
			ce := rec.Value.(CursorEvent)
			LogInfo("Playback", "cursor", ce)
			ScheduleAt(CurrentClick(), ce.Tag, ce)
		}
	}
	err := fileScanner.Err()
	if err != nil {
		LogIfError(err)
	}
	LogInfo("doPlayback ran out of input")
	f.Close()
}

func (e *Engine) RecordingPath(fname string) string {
	return filepath.Join(PaletteDataPath(), "recordings", fname)
}

func (e *Engine) NewRecordingPath() (string, error) {
	recdir := filepath.Join(PaletteDataPath(), "recordings")
	_, err := os.Stat(recdir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			LogInfo("NewRecordingPath: Creating %s", recdir)
			err = os.MkdirAll(recdir, os.FileMode(0777))
		}
		if err != nil {
			return "", err
		}
	}
	for {
		fname := fmt.Sprintf("%03d.json", e.recordingIndex)
		fpath := filepath.Join(recdir, fname)
		if !kit.FileExists(fpath) {
			return fpath, nil
		}
		e.recordingIndex++
	}
}

func (e *Engine) RecordStartEvent() {
	pe := PlaybackEvent{
		Click: CurrentClick(),
	}
	e.RecordPlaybackEvent(pe)
}

func (e *Engine) RecordStopEvent() {
	pe := PlaybackEvent{
		Click:     CurrentClick(),
		IsRunning: false,
	}
	e.RecordPlaybackEvent(pe)
}

// The following routines can make use of generics, I suspect

func (e *Engine) RecordPlaybackEvent(event PlaybackEvent) {
	if e.recordingFile == nil {
		return
	}
	bytes := []byte("{\"PlaybackEvent\":")
	morebytes, err := json.Marshal(event)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, morebytes...)
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordMidiEvent(event *MidiEvent) {
	if e.recordingFile == nil {
		return
	}
	bytes := []byte("{\"MidiEvent\":")
	morebytes, err := json.Marshal(event)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, morebytes...)
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordOscEvent(event *OscEvent) {
	if e.recordingFile == nil {
		return
	}
	re := RecordingEvent{
		Event: "osc",
		Value: event,
	}
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '}', '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}

func (e *Engine) RecordCursorEvent(event CursorEvent) {
	if e.recordingFile == nil {
		return
	}

	re := RecordingEvent{
		Event: "cursor",
		Value: event,
	}
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '\n')

	e.recordingMutex.Lock()

	_, err = e.recordingFile.Write(bytes)

	e.recordingMutex.Unlock()

	LogIfError(err)
}

func (e *Engine) SaveRecordingEvent(re RecordingEvent) {
	bytes, err := json.Marshal(re)
	if err != nil {
		LogIfError(err)
		return
	}
	bytes = append(bytes, '\n')
	_, err = e.recordingFile.Write(bytes)
	LogIfError(err)
}
*/