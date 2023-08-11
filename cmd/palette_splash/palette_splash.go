package main

import (
	"flag"
	// "image"
	"image/color"
	"log"
	"math"
	"os/exec"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var screenWidth = 640
var screenHeight = 480
var screenImage *ebiten.Image
var normalLineHeight = 20
var bigLineHeight = 50
var screenText = "Hello, World!"
var normalFontSize float64 = 24
var bigFontSize float64 = 48

var (
	whiteImage = ebiten.NewImage(3, 3)
)

func init() {
	whiteImage.Fill(color.White)
}

func genVertices(num int) []ebiten.Vertex {

	centerX := float32(screenWidth / 2)
	centerY := float32(screenHeight / 2)
	r := float64(160.0)

	vs := []ebiten.Vertex{}
	for i := 0; i < num; i++ {
		rate := float64(i) / float64(num)
		cr := 0.0
		cg := 0.0
		cb := 0.0
		if rate < 1.0/3.0 {
			cb = 2 - 2*(rate*3)
			cr = 2 * (rate * 3)
		}
		if 1.0/3.0 <= rate && rate < 2.0/3.0 {
			cr = 2 - 2*(rate-1.0/3.0)*3
			cg = 2 * (rate - 1.0/3.0) * 3
		}
		if 2.0/3.0 <= rate {
			cg = 2 - 2*(rate-2.0/3.0)*3
			cb = 2 * (rate - 2.0/3.0) * 3
		}
		vs = append(vs, ebiten.Vertex{
			DstX:   float32(r*math.Cos(2*math.Pi*rate)) + centerX,
			DstY:   float32(r*math.Sin(2*math.Pi*rate)) + centerY,
			SrcX:   0,
			SrcY:   0,
			ColorR: float32(cr),
			ColorG: float32(cg),
			ColorB: float32(cb),
			ColorA: 1,
		})
	}

	vs = append(vs, ebiten.Vertex{
		DstX:   centerX,
		DstY:   centerY,
		SrcX:   0,
		SrcY:   0,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	})

	return vs
}

/*
func rectVertices() []ebiten.Vertex {

	vs := []ebiten.Vertex{}

	cr := 1.0
	cg := 0.0
	cb := 0.0

	vs = append(vs, ebiten.Vertex{
		SrcX:   0,
		SrcY:   0,
		DstX:   float32(screenWidth),
		DstY:   float32(0.0),
		ColorR: float32(cr),
		ColorG: float32(cg),
		ColorB: float32(cb),
		ColorA: 1,
	})
	vs = append(vs, ebiten.Vertex{
		SrcX:   float32(screenWidth),
		SrcY:   float32(0.0),
		DstX:   float32(screenWidth),
		DstY:   float32(screenHeight),
		ColorR: float32(cr),
		ColorG: float32(cg),
		ColorB: float32(cb),
		ColorA: 1,
	})
	vs = append(vs, ebiten.Vertex{
		SrcX:   float32(screenWidth),
		SrcY:   float32(screenHeight),
		DstX:   float32(0.0),
		DstY:   float32(screenHeight),
		ColorR: float32(cr),
		ColorG: float32(cg),
		ColorB: float32(cb),
		ColorA: 1,
	})

	vs = append(vs, ebiten.Vertex{
		SrcX:   float32(0.0),
		SrcY:   float32(screenHeight),
		DstX:   0,
		DstY:   0,
		ColorR: 1,
		ColorG: 1,
		ColorB: 1,
		ColorA: 1,
	})

	return vs
}
*/

type Game struct {
	vertices []ebiten.Vertex
	ngon     int
}

func (g *Game) Update() error {

	if len(g.vertices) == 0 {
		g.vertices = genVertices(g.ngon)
		// g.vertices = rectVertices()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	vector.DrawFilledRect(screen, 0, 0, float32(screenWidth), float32(screenHeight), color.RGBA{0x00, 0x00, 0x00, 0xff}, false)
	screen.DrawImage(screenImage, nil)

	/*
		var x float32 = 10.0
		var y float32 = 10.0
		var w float32 = 200.0
		var h float32 = 200.0
		c := color.White

		vector.DrawFilledRect(screen, x, y, w, h, c, false)
	*/

	/*
		op := &ebiten.DrawTrianglesOptions{}
		op.Address = ebiten.AddressUnsafe
		indices := []uint16{}
		for i := 0; i < g.ngon; i++ {
			indices = append(indices, uint16(i), uint16(i+1)%uint16(g.ngon), uint16(g.ngon))
		}

		screen.DrawTriangles(g.vertices, indices, whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image), op)
	*/

	/*
		text.Draw(screen, screenText, bigFont, 10, 160, color.White)
		text.Draw(screen, "Questions: me@timthompson.com", normalFont,
			10, screenHeight-10, color.White)
	*/
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {

	wp := flag.Int("width", 800, "screen width")
	hp := flag.Int("height", 800, "screen height")
	tp := flag.String("text", "Normal Text", "text string")
	ip := flag.String("image", "sppro_attractscreen.png", "image filename")

	flag.Parse()

	// fmt.Printf("args=%v\n", flag.Args())
	// fmt.Printf("width: %d, height: %d, text: %s\n", *wp, *hp, *tp)

	screenWidth = int(*wp)
	screenHeight = int(*hp)
	screenText = *tp

	args := flag.Args()
	_ = args

	engine.InitLog("splash")

	imgpath := engine.ConfigFilePath(*ip)
	if engine.FileExists(imgpath) {
		engine.LogInfo("Loading image", "imgpath", imgpath)
		img, _, err := ebitenutil.NewImageFromFile(imgpath)
		if err != nil {
			log.Fatal(err)
		}
		screenImage = img
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("palette_splash")
	textinit()

	go func() {

		time.Sleep(time.Millisecond * 200)

		cmd := exec.Command("nircmdc.exe", "win", "-style", "stitle", "palette_splash", "0x00CA0000")
		err := cmd.Run()
		engine.LogIfError(err)

		cmd = exec.Command("nircmdc.exe", "win", "max", "stitle", "palette_splash")
		err = cmd.Run()
		engine.LogIfError(err)
	}()

	if err := ebiten.RunGame(&Game{ngon: 10}); err != nil {
		log.Fatal(err)
	}
}

var bigFont font.Face
var normalFont font.Face

func textinit() {

	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	var dpi float64 = 72

	normalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    normalFontSize,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}

	bigFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    bigFontSize,
		DPI:     dpi,
		Hinting: font.HintingFull, // Use quantization to save glyph cache images.
	})
	if err != nil {
		log.Fatal(err)
	}

	// Adjust the line height.
	bigFont = text.FaceWithLineHeight(bigFont, float64(bigLineHeight))
	normalFont = text.FaceWithLineHeight(normalFont, float64(normalLineHeight))
}
