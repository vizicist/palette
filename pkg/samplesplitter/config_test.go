package samplesplitter

import (
	"path/filepath"
	"testing"
)

func TestConfigNormalizeDefaultsReverbLength(t *testing.T) {
	config := Config{}
	if err := config.Normalize(); err != nil {
		t.Fatalf("Normalize err = %v", err)
	}
	if config.ReverbLength != DefaultReverbLength {
		t.Fatalf("ReverbLength = %v, want %v", config.ReverbLength, DefaultReverbLength)
	}
}

func TestConfigNormalizeClampsReverbLength(t *testing.T) {
	config := Config{ReverbLength: MaxReverbLength * 2}
	if err := config.Normalize(); err != nil {
		t.Fatalf("Normalize err = %v", err)
	}
	if config.ReverbLength != MaxReverbLength {
		t.Fatalf("ReverbLength = %v, want %v", config.ReverbLength, MaxReverbLength)
	}
}

func TestConfigNormalizePreservesExplicitMP3Dir(t *testing.T) {
	dir := t.TempDir()
	config := Config{MP3Dir: dir}
	if err := config.Normalize(); err != nil {
		t.Fatalf("Normalize err = %v", err)
	}
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if config.MP3Dir != want {
		t.Fatalf("MP3Dir = %q, want %q", config.MP3Dir, want)
	}
}
