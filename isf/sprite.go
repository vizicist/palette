package isf

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/vizicist/palette/kit"
	"github.com/vizicist/palette/glhf"
)

// Sprite xxx
type Sprite interface {
	Draw(*glhf.Shader)
	ID() string
	SpriteState() *SpriteState
	SpriteParams() *SpriteParams
	AdjustForAge(age float32)
}

// SpriteVector xxx
type SpriteVector struct {
	x float32
	y float32
}

// SpriteParams xxx
type SpriteParams struct {
	alphafinal     float32
	alphainitial   float32
	alphatime      float32
	aspect         float32
	bounce         bool
	filled         bool
	huefillfinal   float32
	huefillinitial float32
	huefilltime    float32
	huefinal       float32
	hueinitial     float32
	huetime        float32
	lifetime       float32
	luminance      float32
	mirrortype     string
	movedir        string
	noisevertex    float32
	rotangdir      string
	rotanginit     float32
	rotangspeed    float32
	rotauto        bool
	saturation     float32
	sizefinal      float32
	sizeinitial    float32
	sizetime       float32
	speed          float32
	thickness      float32
	cursorsprites  bool
	nsprites       int
	smoothxyz      int
	shape          string
	zmin           float32
}

// SpriteState xxx
type SpriteState struct {
	x           float32
	y           float32
	size        float32
	visible     bool
	direction   float32
	hue         float32
	huefill     float32
	depth       float32
	alpha       float32
	born        time.Time
	NoAging     bool // if true, the Sprite doesn't age
	lasttm      int
	rotangsofar float32
	stationary  bool
	seq         int // sprite sequence # (mostly for debugging)
	rotdir      int // -1, 0, 1
	rotanginit  float32
}

// RadToDeg and DegToRad are conversion factors
const (
	RadToDeg = 180 / math.Pi
	DegToRad = math.Pi / 180
)

func envelopeValue(initial float32, final float32, duration float32, dt float32) float32 {
	if dt >= duration {
		return final
	}
	if dt <= 0 {
		return initial
	}
	return initial + (final-initial)*(dt/duration)
}

// RotdirValue xxx
func RotdirValue(s string) int {
	dir := 1
	switch s {
	case "right":
		dir = 1
	case "left":
		dir = -1
	case "random":
		if rand.Intn(2) == 0 {
			dir = 1
		} else {
			dir = -1
		}
	default:
		log.Printf("rotdirValue, bad value (%s) for rotangdir, assuming right\n", s)
		dir = 1
	}
	return dir
}

// NewSpriteParams xxx
func NewSpriteParams(params *kit.ParamValues) *SpriteParams {
	return &SpriteParams{
		alphainitial:   params.GetFloatValue("alphainitial"),
		alphafinal:     params.GetFloatValue("alphafinal"),
		alphatime:      params.GetFloatValue("alphatime"),
		aspect:         params.GetFloatValue("aspect"),
		filled:         params.GetBoolValue("filled"),
		sizeinitial:    params.GetFloatValue("sizeinitial"),
		sizefinal:      params.GetFloatValue("sizefinal"),
		sizetime:       params.GetFloatValue("sizetime"),
		bounce:         params.GetBoolValue("bounce"),
		huefillfinal:   params.GetFloatValue("huefillfinal"),
		huefillinitial: params.GetFloatValue("huefillinitial"),
		huefilltime:    params.GetFloatValue("huefilltime"),
		huefinal:       params.GetFloatValue("huefinal"),
		hueinitial:     params.GetFloatValue("hueinitial"),
		huetime:        params.GetFloatValue("huetime"),
		lifetime:       params.GetFloatValue("lifetime"),
		luminance:      params.GetFloatValue("luminance"),
		mirrortype:     params.GetStringValue("mirrortype", "non"),
		movedir:        params.GetStringValue("movedir", "0.0"),
		noisevertex:    params.GetFloatValue("noisevertex"),
		rotangdir:      params.GetStringValue("rotangdir", "right"),
		rotanginit:     params.GetFloatValue("rotanginit"),
		rotangspeed:    params.GetFloatValue("rotangspeed"),
		rotauto:        params.GetBoolValue("rotauto"),
		saturation:     params.GetFloatValue("saturation"),
		speed:          params.GetFloatValue("speed"),
		thickness:      params.GetFloatValue("thickness"),
		cursorsprites:  params.GetBoolValue("cursorsprites"),
		nsprites:       params.GetIntValue("nsprites"),
		smoothxyz:      params.GetIntValue("smoothxyz"),
		shape:          params.GetStringValue("shape", "square"),
		zmin:           params.GetFloatValue("zmin"),
	}
}

// SpriteAdjustForAge updates a SpriteState's values to reflect a given age
func SpriteAdjustForAge(s Sprite, age float32) {
	state := s.SpriteState()
	params := s.SpriteParams()
	state.size = envelopeValue(params.sizeinitial, params.sizefinal, params.sizetime, age)
	// log.Printf("size init=%f final=%f time=%f age=%f lifetime=%f new size=%f\n", params.sizeinitial, params.sizefinal, params.sizetime, age, params.lifetime, state.size)
	state.hue = envelopeValue(params.hueinitial, params.huefinal, params.huetime, age)
	state.alpha = envelopeValue(params.alphainitial, params.alphafinal, params.alphatime, age)
	if params.rotangspeed != 0.0 {
		state.rotangsofar = float32(math.Mod(float64(float32(state.rotdir)*age*params.rotangspeed), 360.0))
	}
}
