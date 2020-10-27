package engine

import "math"

// VizColor xxx
type VizColor struct {
	red        float32 // 0 to 255
	green      float32 // 0 to 255
	blue       float32 // 0 to 255
	hue        float32 // 0.0 to 360.0
	luminance  float32 // 0.0 to 1.0
	saturation float32 // 0.0 to 1.0
}

func min(a float32, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max(a float32, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// ColorFromRGB creates a VizColor from red, green, blue
func ColorFromRGB(red, green, blue float32) VizColor {
	c := VizColor{red: red, green: green, blue: blue}
	minval := min(c.red, min(c.green, c.blue))
	maxval := max(c.red, max(c.green, c.blue))
	mdiff := maxval - minval
	msum := maxval + minval
	c.luminance = float32(msum) / 510.0
	if maxval == minval {
		c.saturation = 0
		c.hue = 0
	} else {
		var rnorm float32 = (maxval - c.red) / mdiff
		var gnorm float32 = (maxval - c.green) / mdiff
		var bnorm float32 = (maxval - c.blue) / mdiff
		if c.luminance <= .5 {
			c.saturation = mdiff / msum
		} else {
			c.saturation = mdiff / (510.0 - msum)
		}
		if c.red == maxval {
			c.hue = 60 * (6 + bnorm - gnorm)
		} else if c.green == maxval {
			c.hue = 60 * (2 + rnorm - bnorm)
		} else if c.blue == maxval {
			c.hue = 60 * (4 + gnorm - rnorm)
		}
		c.hue = float32(math.Mod(float64(c.hue), 360.0))
	}
	return c
}

// ColorFromHLS creates a color from hue, luminance, and saturation
func ColorFromHLS(hue, luminance, saturation float32) VizColor {
	c := VizColor{hue: hue, luminance: luminance, saturation: saturation}
	if c.saturation == 0 {
		t := float32(c.luminance * 255)
		c.red = t
		c.green = t
		c.blue = t
	} else {
		var rm2 float32
		if c.luminance <= 0.5 {
			rm2 = c.luminance + c.luminance*c.saturation
		} else {
			rm2 = c.luminance + c.saturation - c.luminance*c.saturation
		}
		rm1 := 2*c.luminance - rm2
		c.red = toRGB1(rm1, rm2, c.hue+120)
		c.green = toRGB1(rm1, rm2, c.hue)
		c.blue = toRGB1(rm1, rm2, c.hue-120)
	}
	return c
}

func toRGB1(rm1 float32, rm2 float32, rh float32) float32 {
	if rh > 360 {
		rh -= 360
	} else if rh < 0 {
		rh += 360
	}

	if rh < 60 {
		rm1 = rm1 + (rm2-rm1)*rh/60
	} else if rh < 180 {
		rm1 = rm2
	} else if rh < 240 {
		rm1 = rm1 + (rm2-rm1)*(240-rh)/60
	}
	return rm1 * 255
}
