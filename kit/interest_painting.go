package kit

import "math"

// PaintingMetrics measures new, visible activity near the gesture rather than
// rewarding detail or motion anywhere on the output monitor.
type PaintingMetrics struct {
	Response         float64 `json:"response"`
	Localization     float64 `json:"localization"`
	Score            float64 `json:"score"`
	centerX, centerY float64
}

// Probe both axes, both directions, and separated parts of the screen. A fixed
// central animation, a mirror, or a random placement cannot follow all four.
func interestPaintingStrokes() [4][3]CursorPos {
	return [4][3]CursorPos{
		{{X: 0.2, Y: 0.2, Z: 0.6}, {X: 0.5, Y: 0.2, Z: 0.6}, {X: 0.8, Y: 0.2, Z: 0.6}},
		{{X: 0.8, Y: 0.8, Z: 0.6}, {X: 0.5, Y: 0.8, Z: 0.6}, {X: 0.2, Y: 0.8, Z: 0.6}},
		{{X: 0.2, Y: 0.8, Z: 0.6}, {X: 0.2, Y: 0.5, Z: 0.6}, {X: 0.2, Y: 0.2, Z: 0.6}},
		{{X: 0.8, Y: 0.2, Z: 0.6}, {X: 0.8, Y: 0.5, Z: 0.6}, {X: 0.8, Y: 0.8, Z: 0.6}},
	}
}

type interestCursorTransform struct {
	scaleX, scaleY, offsetX, offsetY float64
}

func currentInterestCursorTransform() interestCursorTransform {
	t := interestCursorTransform{scaleX: 1, scaleY: 1}
	for _, axis := range []struct {
		name   string
		scale  *float64
		offset *float64
	}{
		{"x", &t.scaleX, &t.offsetX},
		{"y", &t.scaleY, &t.offsetY},
	} {
		if v, err := GetParamFloat("global.cursorscale" + axis.name); err == nil && v > 0 {
			*axis.scale = v
		}
		if v, err := GetParamFloat("global.cursoroffset" + axis.name); err == nil {
			*axis.offset = v
		}
	}
	return t
}

func (t interestCursorTransform) screenPosition(p CursorPos) CursorPos {
	// The FFGL draws in bottom-up OpenGL coordinates; captures are top-down.
	return CursorPos{X: p.X*t.scaleX + t.offsetX, Y: 1 - (p.Y*t.scaleY + t.offsetY)}
}

func paintingWeight(x, y float64, start, end CursorPos) float64 {
	dx, dy := end.X-start.X, end.Y-start.Y
	fraction := 0.0
	if length2 := dx*dx + dy*dy; length2 > 0 {
		fraction = math.Max(0, math.Min(1, ((x-start.X)*dx+(y-start.Y)*dy)/length2))
	}
	distance := math.Hypot(x-(start.X+fraction*dx), y-(start.Y+fraction*dy))
	// Allow a brush width and a little smoothing/render latency, then penalize
	// activity away from the current stroke segment smoothly.
	distance = math.Max(0, distance-0.08)
	return math.Exp(-distance * distance / (2 * 0.10 * 0.10))
}

// idle is the absolute change observed before moving the cursor, scaled to the
// response interval. Subtracting it discounts independently animated imagery
// and the fading/motion of marks left by earlier strokes. Absolute differences
// allow dark paint on a light background as well as light paint on black.
func computePaintingMetrics(before, after, idle []float64, w, h int, start, end CursorPos) PaintingMetrics {
	m := PaintingMetrics{}
	n := w * h
	if w <= 0 || h <= 0 || len(before) != n || len(after) != n || (idle != nil && len(idle) != n) {
		return m
	}
	energy, aligned, chance := 0.0, 0.0, 0.0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			delta := math.Abs(after[i] - before[i])
			if idle != nil {
				delta -= 1.2 * idle[i]
			}
			// Ignore barely visible changes and capture noise.
			delta = math.Max(0, delta-0.012)
			weight := paintingWeight((float64(x)+0.5)/float64(w), (float64(y)+0.5)/float64(h), start, end)
			energy += delta
			aligned += delta * weight
			m.centerX += delta * (float64(x) + 0.5) / float64(w)
			m.centerY += delta * (float64(y) + 0.5) / float64(h)
			chance += weight
		}
	}
	if energy == 0 {
		return m
	}
	m.Response = math.Min(1, energy/float64(n)/0.0005)
	m.centerX /= energy
	m.centerY /= energy
	chance /= float64(n)
	// Remove the overlap a fullscreen change would get just by chance. A busy
	// screen or noise must not pass merely because some pixels cover the path.
	if chance < 1 {
		m.Localization = math.Max(0, math.Min(1, (aligned/energy-chance)/(1-chance)))
	}
	m.Score = m.Response * m.Localization
	return m
}

func paintingInterestScore(appearance float64, painting PaintingMetrics) float64 {
	// Painting is required; visual richness only supplies a small bonus. A
	// clear, sparse brush can pass without filling the screen with texture.
	return painting.Score * (0.9 + 0.1*appearance)
}

func acceptsPaintingScore(score, level float64) bool {
	// Level zero accepts any visible painting response, never blank output.
	return score > 0 && score >= math.Max(0, math.Min(1, level))
}
