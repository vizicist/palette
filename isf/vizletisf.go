package isf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vizicist/montage/engine"
	"github.com/vizicist/montage/glhf"
	"github.com/vizicist/montage/spout"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

// VizletISF is one ISF shader within a Pipeline
type VizletISF struct {
	Vizlet
	name            string
	shader          *glhf.Shader
	sliceFullScreen *glhf.VertexSlice
	frame           *glhf.Frame

	inputFromVizlet Vizlet
	inputFromSpout  string

	spoutReceiverOK   bool
	spoutInputTexture *glhf.Texture
	inputTexture      *glhf.Texture
	receiveWidth      int
	receiveHeight     int

	hasSpoutOut bool
	spoutSender *spout.Sender

	jsonDict       map[string]interface{}
	isfHeader      isfHeader
	isfInitialized bool // initialization of uniform variables
}

var defaultVertexShaderISF = `

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	isf_FragNormCoord = vec2((gl_Position.x+1.0)/2.0, (gl_Position.y+1.0)/2.0); 
	_inputImage_texCoord = texture;
}
`

var defaultFragmentShaderISF = `
in vec2 _inputImage_texCoord;

out vec4 color;

uniform sampler2D tex;

void main() {
	color = texture(tex, _inputImage_texCoord);
}
`

var vertexShader0 = `
#version 330 core

in vec2 position;
in vec2 texture;
in vec2 RENDERSIZE;

out vec2 _inputImage_texCoord;

void main() {
	gl_Position = vec4(position, 0.0, 1.0);
	_inputImage_texCoord = texture;
}
`

var fragmentShader0 = `
#version 330 core

in vec2 _inputImage_texCoord;

out vec4 color;

uniform sampler2D tex;

void main() {
	color = texture(tex, _inputImage_texCoord);
}
`

// DoVizlet xxx
func (viz *VizletISF) DoVizlet() error {

	if viz.inputFromVizlet != nil && viz.inputFromSpout != "" {
		return fmt.Errorf("VizletISF %s: unable to handle Spout input and passed-in inputTexture", viz.inputFromVizlet.Name())
	}

	switch {

	case viz.inputFromSpout != "":

		if viz.receiveSpoutInputTexture() == false {
			return fmt.Errorf("Unable to receive spout input = %s", viz.inputFromSpout)
		}
		viz.inputTexture = viz.spoutInputTexture

	case viz.inputFromVizlet != nil:
		viz.inputTexture = viz.inputFromVizlet.OutputTexture()

	}

	if viz.inputTexture == nil {
		return fmt.Errorf("VizletISF %s: no input texture?", viz.Name())
	}

	viz.frame.Begin()
	viz.shader.Begin()

	// the shader has to be bound before we can set uniforms

	// XXX - the RENDERSIZE should probably only be set once,
	// not on each Vizlet
	for i, uniform := range viz.shader.UniformFormat() {
		if uniform.Name == "RENDERSIZE" {
			val := mgl32.Vec2{800.0, 600.0}
			ok := viz.shader.SetUniformAttr(i, val)
			if !ok {
				// The variable may have been optimized away,
				// so this probably isn't an actual error.
			}
		}
	}

	if !viz.isfInitialized {
		err := viz.initializeUniformDefaults()
		if err != nil {
			return err
		}
		viz.isfInitialized = true
	}

	glhf.Clear(0, 1, 0, 1)

	viz.inputTexture.Begin()

	viz.sliceFullScreen.Begin()
	viz.sliceFullScreen.Draw()
	viz.sliceFullScreen.End()

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

// Name xxx
func (viz *VizletISF) Name() string {
	return viz.name
}

// Shader xxx
func (viz *VizletISF) Shader() *glhf.Shader {
	return viz.shader
}

// SetInputFromVizlet xxx
func (viz *VizletISF) SetInputFromVizlet(from Vizlet) {
	viz.inputFromVizlet = from
}

var tjtval float32 = 0.0
var tjtinc float32 = 0.0005

// SetSpoutIn xxx
func (viz *VizletISF) SetSpoutIn(spoutin string) {
	viz.inputFromSpout = spoutin
}

// receiveSpoutInputTexture xxx
func (viz *VizletISF) receiveSpoutInputTexture() bool {

	// One-time initialization of the spout receiver
	if !viz.spoutReceiverOK {
		viz.spoutReceiverOK = spout.CreateReceiver(viz.inputFromSpout, &viz.receiveWidth, &viz.receiveHeight, false)
		// The Texture shouldn't be created until after we have successfully
		// created the receiver.
		if !viz.spoutReceiverOK {
			viz.spoutInputTexture = nil
			return false
		}
		// New blank texture
		w := viz.receiveWidth
		h := viz.receiveHeight
		viz.spoutInputTexture = glhf.NewTexture(w, h, true, make([]uint8, w*h*4))
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

func (viz *VizletISF) initializeUniformDefaults() error {
	for i, input := range viz.isfHeader.INPUTS {
		var univalue interface{}
		switch input.TYPE {
		case "event":
			// not handled yet
		case "bool":
			// XXX - should be using input values which are
			// initialized to defaults elsewhere
			if input.DEFAULT == nil {
				univalue = false // if "DEFAULT" is not specified
			} else {
				// Even if the IFS header has a number for a bool's DEFAULT,
				// it will have been converted to a bool before this
				univalue = input.DEFAULT.(bool)
			}
		case "long":
			// not handled yet
			if input.DEFAULT == nil {
				univalue = float64(0.0) // if "DEFAULT" is not specified
			} else {
				fv := input.DEFAULT.(float64)
				univalue = int32(fv)
			}
		case "float":
			// XXX - should be using input values which are
			// initialized to defaults elsewhere
			if input.DEFAULT == nil {
				univalue = float64(0.0) // if "DEFAULT" is not specified
			} else {
				univalue = input.DEFAULT.(float64)
			}
		case "point2D":
			if input.DEFAULT == nil {
				univalue = mgl32.Vec2{0.3, 0.3}
			} else {
				defpair := input.DEFAULT.([]interface{})
				if len(defpair) != 2 {
					return fmt.Errorf("DoVizletISF: DEFAULT point2D value - unexpected format")
				}
				x := defpair[0].(float64)
				y := defpair[1].(float64)
				univalue = mgl32.Vec2{float32(x), float32(y)}
			}
			log.Printf("INITIALIZING point2D input %s to %+v\n", input.NAME, univalue)
		case "color":
			if input.DEFAULT == nil {
				univalue = mgl32.Vec4{1, 1, 1, 1} // white
			} else {
				defpair := input.DEFAULT.([]interface{})
				if len(defpair) != 4 {
					return fmt.Errorf("DoVizletISF: DEFAULT color value - unexpected format")
				}
				r := defpair[0].(float64)
				g := defpair[1].(float64)
				b := defpair[2].(float64)
				a := defpair[3].(float64)
				univalue = mgl32.Vec4{float32(r), float32(g), float32(b), float32(a)}
			}
			log.Printf("INITIALIZING color input %s to %+v\n", input.NAME, univalue)
		case "image":
			univalue = int32(gl.TEXTURE0)
		case "audio":
			// not handled yet
		case "audioFFT":
			// not handled yet
		}

		if univalue == nil {
			return fmt.Errorf("DoVizletISF: unable to handle input.TYPE=%s", input.TYPE)
		}

		// Note: i is the index within the INPUTS array, NOT a Uniform id/location
		ok := viz.shader.SetUniformAttr(i, univalue)
		if !ok {
			// The variable may have been optimized away,
			// so this usually isn't an actual error.
		}
	}
	return nil
}

// SetSpoutOut xxx
func (viz *VizletISF) SetSpoutOut(spoutout string) {
	viz.spoutSender = spout.CreateSender(spoutout, viz.Width(), viz.Height())
	viz.hasSpoutOut = true
}

// OutputTexture xxx
func (viz *VizletISF) OutputTexture() *glhf.Texture {
	return viz.frame.Texture()
}

// Width xxx
func (viz *VizletISF) Width() int {
	return viz.frame.Texture().Width()
}

// Height xxx
func (viz *VizletISF) Height() int {
	return viz.frame.Texture().Height()
}

func isFileExist(name string) bool {
	_, err := os.Stat(name)
	if err != nil {
		return false
	}
	return true
}

func separatePreambleOfISF(isf string) (preamble string, restof string, err error) {
	i1 := strings.Index(isf, "/*")
	i2 := strings.Index(isf, "*/")
	if i1 == -1 || i2 == -1 {
		return "", "", fmt.Errorf("unable to find ISF comment in shader file")
	}
	leng := len(isf)
	i1 += 2
	if i1 >= leng {
		i1 = leng - 1
	}
	preamble = isf[i1:i2]
	i2 += 2
	if i2 >= leng {
		i2 = leng - 1
	}
	restof = isf[i2:]
	return preamble, restof, nil
}

type isfInput struct {
	NAME    string
	TYPE    string
	DEFAULT interface{} // depends on TYPE
	MIN     interface{} // depends on TYPE
	MAX     interface{} // depends on TYPE
}

type isfHeader struct {
	CREDIT     string
	ISFVSN     string
	CATEGORIES []string
	INPUTS     []isfInput
}

// NewVizletISF xxx
func NewVizletISF(name string, isfname string, frame *glhf.Frame) (Vizlet, error) {

	viz := &VizletISF{
		name: name,
	}

	fspath, vspath := FilePaths(isfname)

	var fsbytes []byte
	var vsbytes []byte
	var err error

	if isFileExist(fspath) {
		fsbytes, err = ioutil.ReadFile(fspath)
		if err != nil {
			return nil, err
		}
	} else {
		if DebugISF {
			log.Printf("NewVizletISF: no .fs file for %s, using default fragmentShaderISF\n", isfname)
		}
		fsbytes = []byte(defaultFragmentShaderISF)
	}

	if isFileExist(vspath) {
		vsbytes, err = ioutil.ReadFile(vspath)
		if err != nil {
			return nil, err
		}
	} else {
		if DebugISF {
			log.Printf("NewVizletISF: no .vs file for %s, using default vertexShaderISF\n", isfname)
		}
		vsbytes = []byte(defaultVertexShaderISF)
	}

	originalVsShader := string(vsbytes)

	preamble, originalFsShader, err := separatePreambleOfISF(string(fsbytes))
	if err != nil {
		return nil, err
	}

	viz.parsePreamble(preamble)

	vertexFormat := glhf.AttrFormat{
		{Name: "position", Type: glhf.Vec2},
		{Name: "texture", Type: glhf.Vec2},
	}

	// create the declarations for ISF variables
	fsDeclarations := viz.fragmentShaderDeclarations()
	vsDeclarations := viz.vertexShaderDeclarations()

	isfInputs, uniformFormat, err := viz.isfUniformDeclarations()
	if err != nil {
		return nil, err
	}

	// Add the automatically-declared variables for ISF.
	// Note: isf_FragNormCoord gets set using the "position", in isf_vertShaderInit().
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "PASSINDEX", Type: glhf.Int})
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "RENDERSIZE", Type: glhf.Vec2})
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "TIME", Type: glhf.Float})
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "TIMEDELTA", Type: glhf.Float})
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "DATE", Type: glhf.Vec4})
	uniformFormat = append(uniformFormat, glhf.Attr{Name: "FRAMEINDEX", Type: glhf.Int})

	version := "\n#version 330 core\n"
	finalFsShader := version + fsDeclarations + isfInputs + originalFsShader
	finalVsShader := version + vsDeclarations + isfInputs + originalVsShader

	finalVsShader = strings.ReplaceAll(finalVsShader, "vv_FragNormCoord", "isf_FragNormCoord")
	finalFsShader = strings.ReplaceAll(finalFsShader, "vv_FragNormCoord", "isf_FragNormCoord")

	if DebugShader {
		log.Printf("==================== FINAL FRAGMENT SHADER\n" +
			finalFsShader + "\n====================\n")
		log.Printf("==================== FINAL VERTEX SHADER\n" +
			finalVsShader + "\n====================\n")
	}

	// This is where the shader gets compiled
	shader, err := glhf.NewShader(vertexFormat, uniformFormat, finalVsShader, finalFsShader)
	if err != nil {
		return nil, err
	}

	viz.shader = shader
	viz.frame = frame

	// Note that this slice does NOT have the texture V value
	// flipped the way it is in the Pipeline
	viz.sliceFullScreen = newVertexSlice(shader, 6, 6,
		[]float32{
			-1, -1, 0, 1,
			+1, -1, 1, 1,
			+1, +1, 1, 0,

			-1, -1, 0, 1,
			+1, +1, 1, 0,
			-1, +1, 0, 0,
		})

	return viz, nil
}

// FilePath xxx
func FilePath(nm string) string {
	dir := engine.LocalMontageDir()
	ps := os.Getenv("MONTAGE_SOURCE")
	if ps != "" {
		dir = filepath.Join(ps, "default")
	}
	path := filepath.Join(dir, "isf", nm)
	return path
}

// FilePaths returns the 2 ISF file paths for a given name
func FilePaths(nm string) (string, string) {
	return FilePath(nm + ".fs"), FilePath(nm + ".vs")
}

func (viz *VizletISF) isfUniformDeclarations() (string, glhf.AttrFormat, error) {
	s := "// INPUT variables of ISF shader\n"

	attrs := glhf.AttrFormat{}

	for _, i := range viz.isfHeader.INPUTS {
		line := ""
		// XXX - eventually make this a map lookup?
		switch i.TYPE {
		case "bool":
			line = "uniform bool " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Bool})
		case "long":
			// This value defines some UI VALUES and LABELS
			log.Printf("inputDeclarations: long attribute needs work, name=%s\n", i.NAME)
			line = "uniform int " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Int})
		case "float":
			line = "uniform float " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Float})
		case "point2D":
			line = "uniform vec2 " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Vec2})
		case "color":
			line = "uniform vec4 " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Vec4})
		case "image":
			line = "uniform sampler2D " + i.NAME + ";\n"
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Int})
		case "event":
			log.Printf("inputDeclarations: event type is not handled yet")
		case "audio":
			log.Printf("inputDeclarations: audio attribute may need work, name=%s\n", i.NAME)
			line = "uniform sampler2D " + i.NAME + ";\n" // not sure here
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Int})
		case "audioFFT":
			log.Printf("inputDeclarations: audioFFT attribute may need work, name=%s\n", i.NAME)
			line = "uniform sampler2D " + i.NAME + ";\n" // not sure here
			attrs = append(attrs, glhf.Attr{Name: i.NAME, Type: glhf.Int})
		}
		if line == "" {
			return "", nil, fmt.Errorf("unhandled ISF TYPE=%s for NAME=%s", i.TYPE, i.NAME)
		}
		s += line
	}
	return s, attrs, nil
}

func (viz *VizletISF) fragmentShaderDeclarations() string {
	// Note: "uniform sampler2D inputImage;" is already included if
	// the ISF inputs declare inputImage to be an INPUT.
	return "// Automatically-generated ISF variables\n" +
		"\n" +
		"#define gl_FragColor isf_FragColor\n" +
		"\n" +

		// Eventually, in GLSL 4, this should be
		// layout(location = 0) out vec4 gl_FragColor;
		"out vec4 gl_FragColor;\n" +

		"in vec2 isf_FragNormCoord;\n" +
		"in vec2 _inputImage_texCoord;\n" +
		"\n" +
		"uniform int PASSINDEX;\n" +
		"uniform vec2 RENDERSIZE;\n" +
		"uniform float TIME;\n" +
		"uniform float TIMEDELTA;\n" +
		"uniform vec4 DATE;\n" + // year, month, day, seconds
		"uniform int FRAMEINDEX;\n" +
		"\n" +
		"vec4 IMG_PIXEL(sampler2D img, vec2 coord) {\n" +
		"	return texture(img, vec2(coord.x/RENDERSIZE.x,coord.y/RENDERSIZE.y) );\n" +
		"}\n" +
		"vec4 IMG_THIS_PIXEL(sampler2D img) {\n" +
		"	return texture(img, vec2(isf_FragNormCoord.x,isf_FragNormCoord.y) );\n" +
		"}\n" +
		"vec4 IMG_NORM_PIXEL(sampler2D img, vec2 coord) {\n" +
		"	return texture(img, vec2(coord.x,coord.y) );\n" +
		"}\n" +
		"vec4 IMG_THIS_NORM_PIXEL(sampler2D img) {\n" +
		"	return texture(img, vec2(isf_FragNormCoord.x,isf_FragNormCoord.y) );\n" +
		"}\n" +
		"\n" +
		"// END OF ISF declarations and functions\n"
}

func (viz *VizletISF) vertexShaderDeclarations() string {
	return "// Automatically-generated ISF variables\n" +

		"in vec2 position;\n" +
		"in vec2 texture;\n" +

		"out vec2 isf_FragNormCoord;\n" +
		"out vec2 _inputImage_texCoord;\n" +

		"uniform vec2 RENDERSIZE;\n" +
		"uniform int PASSINDEX;\n" +
		"uniform float TIME;\n" +
		"uniform float TIMEDELTA;\n" +
		"uniform int FRAMEINDEX;\n" +
		"uniform vec4 DATE;\n" + // year, month, day, seconds
		"\n" +
		"// ISF functions\n" +
		"void isf_vertShaderInit() {\n" +
		"  gl_Position = vec4(position, 0.0, 1.0);\n" +
		"  isf_FragNormCoord = vec2((gl_Position.x+1.0)/2.0, (gl_Position.y+1.0)/2.0); \n" +
		"  _inputImage_texCoord = texture;\n" +
		"}\n" +
		"\n" +
		"// END OF ISF declarations and functions\n"
}

func (viz *VizletISF) parsePreamble(preamble string) error {
	var jsonFs interface{}
	err := json.Unmarshal([]byte(preamble), &jsonFs)
	if err != nil {
		return err
	}

	viz.jsonDict = jsonFs.(map[string]interface{})

	credit := viz.jsonDict["CREDIT"]
	if credit != nil {
		viz.isfHeader.CREDIT = credit.(string)
	}

	isfvsn := viz.jsonDict["ISFVSN"]
	if isfvsn != nil {
		viz.isfHeader.ISFVSN = isfvsn.(string)
	}

	categories := viz.jsonDict["CATEGORIES"].([]interface{})
	for _, s := range categories {
		ss := s.(string)
		viz.isfHeader.CATEGORIES = append(viz.isfHeader.CATEGORIES, ss)

	}
	viz.isfHeader.INPUTS = make([]isfInput, 0)

	inputs := viz.jsonDict["INPUTS"].([]interface{})
	for _, i := range inputs {
		inmap := i.(map[string]interface{})
		nm, ok := inmap["NAME"].(string)
		if !ok {
			return fmt.Errorf("ISF: missing NAME value in INPUT")
		}
		t, ok := inmap["TYPE"].(string)
		if !ok {
			return fmt.Errorf("ISF: missing TYPE value in INPUT")
		}
		// The DEFAULT value is optional, but if it's there,
		// we want to handle the two different formats we see in ISF files.
		// Sometimes a bool DEFAULT is a number (0.0,1.0), and
		// sometimes it's a string (or JSON bool, i.e. true or false)
		defaultValue, ok := inmap["DEFAULT"]
		if t == "bool" && ok {
			switch dv := defaultValue.(type) {
			case bool:
				// leave defaultValue alone
			case float64:
				// force it to be bool
				if dv > 0.5 {
					defaultValue = true
				} else {
					defaultValue = false
				}
			}
		}
		input := isfInput{
			NAME:    nm,
			TYPE:    t,
			DEFAULT: defaultValue,
			MIN:     inmap["MIN"],
			MAX:     inmap["MAX"],
		}
		viz.isfHeader.INPUTS = append(viz.isfHeader.INPUTS, input)
	}
	return nil
}
