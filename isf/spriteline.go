package isf

import (
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/vizicist/palette/engine"
)

// SpriteLine xxx
type SpriteLine struct {
	Sprite
	State  *SpriteState
	Params *SpriteParams
}

// SpriteState xxx
func (s *SpriteLine) SpriteState() *SpriteState {
	return s.State
}

// SpriteParams xxx
func (s *SpriteLine) SpriteParams() *SpriteParams {
	return s.Params
}

// NewSpriteLine xxx
func NewSpriteLine(x, y, z float32, params *engine.ParamValues) *SpriteLine {
	spriteParams := NewSpriteParams(params)
	spriteParams.sizeinitial *= z
	spriteParams.sizefinal *= z
	s := &SpriteLine{
		State: &SpriteState{
			born:        time.Now(),
			x:           x,
			y:           y,
			size:        spriteParams.sizeinitial,
			hue:         spriteParams.hueinitial,
			alpha:       spriteParams.alphainitial,
			rotangsofar: spriteParams.rotanginit,
			rotdir:      RotdirValue(params.ParamStringValue("rotangdir", "right")),
		},
		Params: spriteParams,
	}
	// s.AdjustForAge(0) // to initialize things - should probably build into initializer above?
	return s
}

// Draw xxx
func (s *SpriteLine) Draw(viz *VizletISF) {
	nlines := 2
	gl.BufferData(gl.ARRAY_BUFFER, len(LineVerticesForLines)*4, gl.Ptr(LineVerticesForLines), gl.STATIC_DRAW)
	gl.DrawArrays(gl.LINES, 0, int32(2*nlines))
}
