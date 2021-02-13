package isf

import (
	"time"

	"github.com/vizicist/palette/engine"
	"github.com/vizicist/palette/glhf"
)

// SpriteSquare xxx
type SpriteSquare struct {
	Sprite
	State  *SpriteState
	Params *SpriteParams
	// Slice  *glhf.VertexSlice
}

// SpriteState xxx
func (s *SpriteSquare) SpriteState() *SpriteState {
	return s.State
}

// SpriteParams xxx
func (s *SpriteSquare) SpriteParams() *SpriteParams {
	return s.Params
}

// NewSpriteSquare xxx
func NewSpriteSquare(x, y, z float32, params *engine.ParamValues) *SpriteSquare {
	spriteParams := NewSpriteParams(params)
	spriteParams.sizeinitial *= z
	spriteParams.sizefinal *= z
	s := &SpriteSquare{
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
	// log.Printf("newsquaresprite rotangsofar=%f\n", s.State.rotangsofar)
	// s.AdjustForAge(0) // to initialize things - should probably build into initializer above?
	return s
}

var squareSlice *glhf.VertexSlice

// Draw xxx
func (s *SpriteSquare) Draw(shader *glhf.Shader) {

	// Comes in 0,0 to 1,1
	x := s.State.x*2.0 - 1.0
	y := s.State.y*2.0 - 1.0
	sz := s.State.size
	// log.Printf("Draw SpriteSquare x,y = %f, %f  sz = %f\n", x, y, sz)
	data := []float32{
		x - sz, y - sz, 0, 1,
		x + sz, y - sz, 1, 1,
		x + sz, y + sz, 1, 0,

		x - sz, y - sz, 0, 1,
		x + sz, y + sz, 1, 0,
		x - sz, y + sz, 0, 0,
	}
	if squareSlice == nil {
		squareSlice = newVertexSlice(shader, 6, 6, nil)
	}

	squareSlice.Begin()
	squareSlice.SetVertexData(data)
	squareSlice.Draw()
	squareSlice.End()
}
