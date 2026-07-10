package samplesplitter

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	DefaultPort                      = 9876
	DefaultBaseNote                  = 48
	DefaultMIDIPortName              = "16. Internal MIDI"
	DefaultSplitMode                 = "words"
	DefaultIntervalSeconds           = 1.0
	DefaultSilenceThreshold          = 0.01
	DefaultSilenceMinimum            = 0.5
	DefaultWordsPerSplit             = 2
	DefaultWordThreshold             = 0.01
	DefaultReverbLength              = 4.0
	DefaultMinimumMP3DurationSeconds = 10.0
	MinReverbLength                  = 0.25
	MaxReverbLength                  = 8.0
	WaveformPoints                   = 1200
)

var Sigils = []string{"chaos", "oracle", "sacred", "directive"}

var SigilByMIDIChannel = map[int]string{
	0: "chaos",
	1: "oracle",
	2: "sacred",
	3: "directive",
}

type Config struct {
	MP3Dir                    string  `json:"mp3_dir"`
	Port                      int     `json:"port"`
	BaseNote                  int     `json:"base_note"`
	MIDIPortName              string  `json:"midi_port_name,omitempty"`
	PeakStartEnabled          bool    `json:"peak_start_enabled"`
	Compressed                bool    `json:"compressed"`
	ReverbWet                 float64 `json:"reverb_wet"`
	ReverbLength              float64 `json:"reverb_length"`
	DefaultMode               string  `json:"default_mode"`
	DefaultInterval           float64 `json:"default_interval"`
	DefaultWords              int     `json:"default_words_per_split"`
	SilenceThreshold          float64 `json:"silence_threshold"`
	SilenceMinimum            float64 `json:"silence_minimum"`
	MinimumMP3DurationSeconds float64 `json:"minimum_mp3_duration_seconds"`
	WordThreshold             float64 `json:"word_threshold"`
	FFmpegPath                string  `json:"ffmpeg_path"`
}

func DefaultConfig() Config {
	return Config{
		Port:                      DefaultPort,
		BaseNote:                  DefaultBaseNote,
		MIDIPortName:              DefaultMIDIPortName,
		PeakStartEnabled:          true,
		DefaultMode:               DefaultSplitMode,
		DefaultInterval:           DefaultIntervalSeconds,
		DefaultWords:              DefaultWordsPerSplit,
		ReverbLength:              DefaultReverbLength,
		MinimumMP3DurationSeconds: DefaultMinimumMP3DurationSeconds,
		WordThreshold:             DefaultWordThreshold,
		SilenceThreshold:          DefaultSilenceThreshold,
		SilenceMinimum:            DefaultSilenceMinimum,
	}
}

func (c *Config) Normalize() error {
	if c.MP3Dir == "" {
		c.MP3Dir = DefaultMP3Dir()
	}
	abs, err := filepath.Abs(c.MP3Dir)
	if err != nil {
		return err
	}
	c.MP3Dir = abs

	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.BaseNote == 0 {
		c.BaseNote = DefaultBaseNote
	}
	if c.DefaultMode == "" {
		c.DefaultMode = DefaultSplitMode
	}
	if c.DefaultInterval <= 0 {
		c.DefaultInterval = DefaultIntervalSeconds
	}
	if c.DefaultWords <= 0 {
		c.DefaultWords = DefaultWordsPerSplit
	}
	if c.SilenceThreshold <= 0 {
		c.SilenceThreshold = DefaultSilenceThreshold
	}
	if c.SilenceMinimum <= 0 {
		c.SilenceMinimum = DefaultSilenceMinimum
	}
	if c.MinimumMP3DurationSeconds < 0 {
		c.MinimumMP3DurationSeconds = DefaultMinimumMP3DurationSeconds
	}
	c.WordThreshold = clampWordThreshold(c.WordThreshold)
	c.ReverbWet = clampReverbWet(c.ReverbWet)
	if c.ReverbLength <= 0 {
		c.ReverbLength = DefaultReverbLength
	}
	c.ReverbLength = clampReverbLength(c.ReverbLength)
	if c.MP3Dir == "" {
		return errors.New("mp3 directory is required")
	}
	return nil
}

func DefaultMP3Dir() string {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return filepath.Join(string(filepath.Separator), "mp3s")
	}
	return filepath.Join(userProfile, "mp3s")
}

func clampReverbWet(wet float64) float64 {
	if wet < 0 {
		return 0
	}
	if wet > 1 {
		return 1
	}
	return wet
}

func clampWordThreshold(threshold float64) float64 {
	if threshold < 0 {
		return 0
	}
	if threshold > 1 {
		return 1
	}
	return threshold
}

func clampReverbLength(length float64) float64 {
	if length < MinReverbLength {
		return MinReverbLength
	}
	if length > MaxReverbLength {
		return MaxReverbLength
	}
	return length
}
