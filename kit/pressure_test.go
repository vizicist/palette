package kit

import "testing"

func TestPressureToVelocityDoesNotWrapAtFullScale(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want uint8
	}{
		{name: "below range clamps low", in: -1.0, want: 1},
		{name: "zero", in: 0.0, want: 1},
		{name: "middle", in: 0.5, want: 64},
		{name: "one", in: 1.0, want: 127},
		{name: "above range clamps high", in: 2.0, want: 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pressureToVelocity(tt.in, 1, 127)
			if got != tt.want {
				t.Fatalf("pressureToVelocity(%v, 1, 127) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
