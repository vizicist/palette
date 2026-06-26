package kit

import "testing"

func TestPitchSetIndexForXScalesAcrossPitchCount(t *testing.T) {
	tests := []struct {
		name       string
		x          float64
		pitchCount int
		want       int
	}{
		{name: "left edge", x: 0.0, pitchCount: 42, want: 0},
		{name: "first eighth of forty two", x: 1.0 / 8.0, pitchCount: 42, want: 5},
		{name: "middle of forty two", x: 0.5, pitchCount: 42, want: 21},
		{name: "right edge", x: 1.0, pitchCount: 42, want: 41},
		{name: "stylusrmx still spans eight slots", x: 7.0 / 8.0, pitchCount: 8, want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pitchSetIndexForX(tt.x, tt.pitchCount)
			if got != tt.want {
				t.Fatalf("pitchSetIndexForX(%v, %d) = %d, want %d", tt.x, tt.pitchCount, got, tt.want)
			}
		})
	}
}
