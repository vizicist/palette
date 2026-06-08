package samplesplitter

import "testing"

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
