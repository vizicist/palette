package isf

import (
	"fmt"
	"log"
	"time"

	"github.com/vizicist/palette/glhf"
	"github.com/vizicist/palette/spout"

	"github.com/go-gl/gl/v3.3-core/gl"
)

type bits uint8

func setBit(b, flag bits) bits    { return b | flag }
func clearBit(b, flag bits) bits  { return b &^ flag }
func toggleBit(b, flag bits) bits { return b ^ flag }
func hasBit(b, flag bits) bool    { return b&flag != 0 }

// VizletDrawer is one Drawer shader within a Pipeline
type VizletDrawer struct {
	Vizlet
	name   string
	shader *glhf.Shader
	frame  *glhf.Frame

	inputFromVizlet Vizlet
	inputFromSpout  string

	spoutReceiverOK   bool
	spoutInputTexture *glhf.Texture
	inputTexture      *glhf.Texture
	blankInputTexture *glhf.Texture // only if spoutInputTexture isn't received
	receiveWidth      int
	receiveHeight     int

	hasSpoutOut bool
	spoutSender *spout.Sender

	sprites []Sprite
}

var defaultVertexShaderDrawer = `
#version 330 core
in vec2 position;
in vec2 texture;
out vec2 _inputImage_texCoord;
void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	_inputImage_texCoord = texture;
}
`

var defaultFragmentShaderDrawer = `
#version 330 core
in vec2 _inputImage_texCoord;
out vec4 color;
uniform sampler2D tex;
void main() {
	color = texture(tex, _inputImage_texCoord);
	color = vec4(0.0, 0.0, 1.0, 1.0);
}
`

var hasComplainedAboutSpoutInput bool

// DoVizlet xxx
func (viz *VizletDrawer) DoVizlet() error {

	if viz.inputFromVizlet != nil && viz.inputFromSpout != "" {
		return fmt.Errorf("VizletDrawer %s: unable to handle Spout input and passed-in inputTexture", viz.inputFromVizlet.Name())
	}

	switch {

	case viz.inputFromSpout != "":

		if viz.receiveSpoutInputTexture() == true {
			viz.inputTexture = viz.spoutInputTexture
			hasComplainedAboutSpoutInput = false // so we complain everytime it disappears
			// log.Printf("receivedSpoutInputTexture true\n")
		} else {
			if hasComplainedAboutSpoutInput == false {
				// The first time we complain, create/use a blank texture (should it be red?)
				log.Printf("DoVizlet: unable to receive spout input = %s, using blank texture", viz.inputFromSpout)
				w := viz.Width()
				h := viz.Height()
				viz.blankInputTexture = glhf.NewTexture(w, h, true, make([]uint8, w*h*4))
				hasComplainedAboutSpoutInput = true
				// return fmt.Errorf("Unable to receive spout input = %s", viz.inputFromSpout)
			}
			viz.inputTexture = viz.blankInputTexture
		}

	case viz.inputFromVizlet != nil:
		viz.inputTexture = viz.inputFromVizlet.OutputTexture()

	}

	if viz.inputTexture == nil {
		return fmt.Errorf("VizletDrawer %s: no input texture?", viz.Name())
	}

	viz.frame.Begin()
	viz.shader.Begin()

	// This clears the screen (and for debugging, may be a color other than black)
	glhf.Clear(0, 1, 0, 1)

	viz.inputTexture.Begin()

	viz.DrawSprites()

	viz.inputTexture.End()

	viz.shader.End()

	if viz.hasSpoutOut {
		gl.BindTexture(gl.TEXTURE_2D, viz.frame.Texture().ID())
		w := viz.Width()
		h := viz.Height()
		gl.CopyTexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 0, 0, int32(w), int32(h))
		gl.BindTexture(gl.TEXTURE_2D, 0)
		spout.SendTexture(*viz.spoutSender, viz.frame.Texture().ID(), w, h)
	}

	viz.frame.End()

	return nil
}

// AddSprite xxx
func (viz *VizletDrawer) AddSprite(s Sprite) {
	viz.sprites = append(viz.sprites, s)
}

var drawSprites = false

// DrawSprites xxx
func (viz *VizletDrawer) DrawSprites() {
	viz.trySquare(viz.Shader())
	for _, s := range viz.sprites {
		if s != nil {
			s.Draw(viz.Shader())
		} else {
			log.Printf("HEY! something in Reactor.sprites is nil?\n")
		}
	}
}

func (viz *VizletDrawer) trySquare(shader *glhf.Shader) {

	// Comes in 0,0 to 1,1
	x := float32(0.5)
	y := float32(0.5)
	sz := float32(0.1)
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

// AgeSprites xxx
func (viz *VizletDrawer) AgeSprites() {
	// This method of deleting things from an array without
	// allocating a new array is found on this page:
	// https://vbauerster.github.io/2017/04/removing-items-from-a-slice-while-iterating-in-go/
	sprites := viz.sprites[:0]
	now := time.Now()
	for _, s := range viz.sprites {
		addme := false
		if s.SpriteState().NoAging {
			addme = true
		} else {
			// age this sprite
			dt := now.Sub(s.SpriteState().born)
			age := float32(dt.Seconds())
			if age <= s.SpriteParams().lifetime {
				SpriteAdjustForAge(s, age)
				// s.AdjustForAge(age)
				addme = true
			} else {
				// log.Printf("AgeSprites is killing sprite\n")
			}
		}
		if addme {
			sprites = append(sprites, s)
		}
	}
	viz.sprites = sprites
}

// Name xxx
func (viz *VizletDrawer) Name() string {
	return viz.name
}

// Shader xxx
func (viz *VizletDrawer) Shader() *glhf.Shader {
	return viz.shader
}

// SetInputFromVizlet xxx
func (viz *VizletDrawer) SetInputFromVizlet(from Vizlet) {
	viz.inputFromVizlet = from
}

// SetSpoutIn xxx
func (viz *VizletDrawer) SetSpoutIn(spoutin string) {
	viz.inputFromSpout = spoutin
}

var hasComplained = false

// receiveSpoutInputTexture xxx
func (viz *VizletDrawer) receiveSpoutInputTexture() bool {

	// One-time initialization of the spout receiver
	if !viz.spoutReceiverOK {
		viz.spoutReceiverOK = spout.CreateReceiver(viz.inputFromSpout, &viz.receiveWidth, &viz.receiveHeight, false)
		// The Texture shouldn't be created until after we have successfully
		// created the receiver.
		if !viz.spoutReceiverOK {
			if !hasComplained {
				hasComplained = true
				log.Printf("spout.CreateReceiver fails for input=%s\n", viz.inputFromSpout)
			}
			viz.spoutInputTexture = nil
			return false
		}
		if viz.spoutInputTexture == nil {
			// New blank texture
			w := viz.receiveWidth
			h := viz.receiveHeight
			viz.spoutInputTexture = glhf.NewTexture(w, h, true, make([]uint8, w*h*4))
		}
	}

	var w int = viz.Width()
	var h int = viz.Height()
	var textureID int = int(viz.spoutInputTexture.ID())
	var textureTarget = gl.TEXTURE_2D
	var bInvert = false
	var hostFBO = 0

	b := spout.ReceiveTexture(viz.inputFromSpout, &w, &h, textureID, textureTarget, bInvert, hostFBO)
	if !b {
		viz.spoutReceiverOK = false
		return false
	}

	gl.BufferData(gl.ARRAY_BUFFER, len(SquareVerticiesForTriangles)*4, gl.Ptr(SquareVerticiesForTriangles), gl.STATIC_DRAW)
	return true
}

// SetSpoutOut xxx
func (viz *VizletDrawer) SetSpoutOut(spoutout string) {
	viz.spoutSender = spout.CreateSender(spoutout, viz.Width(), viz.Height())
	viz.hasSpoutOut = true
}

// OutputTexture xxx
func (viz *VizletDrawer) OutputTexture() *glhf.Texture {
	return viz.frame.Texture()
}

// Width xxx
func (viz *VizletDrawer) Width() int {
	return viz.frame.Texture().Width()
}

// Height xxx
func (viz *VizletDrawer) Height() int {
	return viz.frame.Texture().Height()
}

// NewVizletDrawer xxx
func NewVizletDrawer(name string, isfname string, frame *glhf.Frame) (Vizlet, error) {

	viz := &VizletDrawer{
		name:    name,
		sprites: make([]Sprite, 0),
	}

	vertexFormat := glhf.AttrFormat{
		{Name: "position", Type: glhf.Vec2},
		{Name: "texture", Type: glhf.Vec2},
	}

	// This is where the shader gets compiled
	shader, err := glhf.NewShader(vertexFormat, glhf.AttrFormat{}, defaultVertexShaderDrawer, defaultFragmentShaderDrawer)
	if err != nil {
		return nil, err
	}

	viz.shader = shader
	viz.frame = frame

	return viz, nil
}

// Sprites xxx
func (viz *VizletDrawer) Sprites() []Sprite {
	return viz.sprites
}
