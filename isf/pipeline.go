package isf

import (
	"fmt"
	"log"

	"github.com/vizicist/palette/glhf"
)

// Pipeline xxx
type Pipeline struct {
	Name   string
	Width  int
	Height int

	shader0         *glhf.Shader
	fullscreenSlice *glhf.VertexSlice

	vizlets []Vizlet
}

// GetVizlet xxx
func (pl *Pipeline) GetVizlet(name string) Vizlet {
	for _, viz := range pl.vizlets {
		if name == viz.Name() {
			return viz
		}
	}
	return nil
}

// FirstVizlet xxx
func (pl *Pipeline) FirstVizlet() Vizlet {
	return pl.vizlets[0]
}

// LastVizlet xxx
func (pl *Pipeline) LastVizlet() Vizlet {
	return pl.vizlets[len(pl.vizlets)-1]
}

// NewPipeline xxx
func NewPipeline(name string, windowWidth int, windowHeight int, visible bool) (*Pipeline, error) {

	// names and types of the attributes referenced by a shader.
	vertexFormat := glhf.AttrFormat{
		{Name: "position", Type: glhf.Vec2},
		{Name: "texture", Type: glhf.Vec2},
	}

	// Here we create a shader. The second argument is the format of the uniform
	// attributes. Since our shader has no uniform attributes, the format is empty.

	// XXX - try to remove need for custom shader (vertexShader0) on the screen output
	// XXX - it should be the same as defaultVertexShaderISF
	shader0, err := glhf.NewShader(vertexFormat, glhf.AttrFormat{}, vertexShader0, fragmentShader0)
	if err != nil {
		return nil, fmt.Errorf("unable to create shader0")
	}

	pl := &Pipeline{
		Name:    name,
		Width:   windowWidth,
		Height:  windowHeight,
		shader0: shader0,
		vizlets: make([]Vizlet, 0),

		// I reversed (0->1,1->0) the texture V values
		// (the 4th number in each row)
		// in this slice in order to flip the image.
		// Not totally sure if that's important
		fullscreenSlice: newVertexSlice(shader0, 6, 6,
			[]float32{
				-1, -1, 0, 0,
				+1, -1, 1, 0,
				+1, +1, 1, 1,

				-1, -1, 0, 0,
				+1, +1, 1, 1,
				-1, +1, 0, 1,
			}),
	}

	// frame4 := glhf.NewFrame(windowWidth, windowHeight, true)

	// ISF shaders that have been tested:
	// Rotate
	// Vibrance
	// Twirl
	// Mirror
	// Triangle
	// Thermal Camera
	// Cubic Warp
	// Emboss
	// Edges
	// Diagonalize
	// Triangles
	// Median
	// Poly Glitch

	// vizlist := []string{"Empty", "Mirror", "Emboss", "Edges", "Cubic Warp", "Poly Glitch", "Thermal Camera", "Emboss"}
	// vizlist := []string{"Mirror", "Poly Glitch"}
	// vizlist := []string{"Mirror"}
	vizlist := []string{}

	var previousViz Vizlet

	// Add a VizletDrawer at the very beginning
	frame := glhf.NewFrame(windowWidth, windowHeight, true)
	viz, err := NewVizletDrawer("VizletEmpty", "Empty", frame)
	if err != nil {
		return nil, fmt.Errorf("unable to create NewVizletDrawer: err=%s", err)
	}
	viz.SetSpoutOut(viz.Name())
	pl.AddVizlet(viz)
	previousViz = viz

	for i, nm := range vizlist {
		frame := glhf.NewFrame(windowWidth, windowHeight, true)
		vizname := fmt.Sprintf("Vizlet#%d", i)
		viz, err := NewVizletISF(vizname, nm, frame)
		if err != nil {
			return nil, fmt.Errorf("unable to create NewVizletISF: err=%s", err)
		}
		viz.SetSpoutOut(viz.Name())
		viz.SetInputFromVizlet(previousViz)
		pl.AddVizlet(viz)
		previousViz = viz
	}

	return pl, nil
}

func newVertexSlice(shader *glhf.Shader, len int, cap int, data []float32) *glhf.VertexSlice {
	// log.Printf("newVertexSlice len/cap=%d,%d\n", len, cap)
	// The slice inherits the vertex format of the supplied shader. Also, it should
	// only be used with that shader.
	slice := glhf.MakeVertexSlice(shader, len, cap)
	if data != nil {
		slice.Begin()
		// We assign data to the vertex slice. The values are in the order as in the vertex
		// format of the slice (shader). Each two floats correspond to an attribute of type
		// glhf.Vec2.
		slice.SetVertexData(data)
		slice.End()
	}
	return slice
}

// AddVizlet xxx
func (pl *Pipeline) AddVizlet(viz Vizlet) {
	pl.vizlets = append(pl.vizlets, viz)
}

// Do runs the pipeline once
func (pl *Pipeline) Do() {

	// Clear the window.
	// 	glhf.Clear(1, 1, 1, 1)
	// gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	if len(pl.vizlets) == 0 {
		log.Printf("Pipeline.Do: no vizlets?\n")
		return
	}

	for _, viz := range pl.vizlets {
		err := viz.DoVizlet()
		if err != nil {
			log.Printf("Do: vizlet=%s, err=%s\n", viz.Name(), err)
		}
	}

	// Now draw the entire screen to the real screen window
	pl.shader0.Begin()
	pl.LastVizlet().OutputTexture().Begin()
	pl.fullscreenSlice.Begin()
	pl.fullscreenSlice.Draw()
	pl.fullscreenSlice.End()
	pl.LastVizlet().OutputTexture().End()
	pl.shader0.End()
}
