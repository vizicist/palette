package gui

import (
	"fmt"
	"image/color"
	"log"
	"time"

	// Don't be tempted to use go-gl

	"github.com/goxjs/gl"
	"github.com/goxjs/glfw"
	"github.com/micaelAlastor/nanovgo"
	"github.com/nats-io/nats.go"
	"github.com/vizicist/palette/engine"
)

func midiHandler(msg *nats.Msg) {
	log.Printf("midiHandler: " + msg.Subject + " " + string(msg.Data))
}

func cursorHandler(msg *nats.Msg) {
	log.Printf("cursorHandler: " + msg.Subject + " " + string(msg.Data))
}

// Run xxx
func Run() {

	err := glfw.Init(gl.ContextWatcher)
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	// demo MSAA
	glfw.WindowHint(glfw.Samples, 4)

	glfwWindow, err := glfw.CreateWindow(600, 800, "Palette", nil, nil)
	if err != nil {
		panic(err)
	}

	glfwWindow.MakeContextCurrent()

	ctx, err := nanovgo.NewContext(0 /*nanovgo.AntiAlias | nanovgo.StencilStrokes | nanovgo.Debug*/)
	defer ctx.Delete()

	if err != nil {
		panic(err)
	}

	var screen *Screen

	glfw.SwapInterval(0)

	frametime, _ := time.ParseDuration("33ms") // 30fps
	tm := time.Now()
	previousTime := tm

	err = LoadFonts(ctx)
	if err != nil {
		log.Printf("gui.Run: err=%s\n", err)
	}

	framenum := 0

	for !glfwWindow.ShouldClose() {

		// t, _ := fps.UpdateGraph()

		fbWidth, fbHeight := glfwWindow.GetFramebufferSize()
		newWidth, newHeight := glfwWindow.GetSize()

		if screen == nil {
			screen = NewScreen("root", ctx)
			glfwWindow.SetKeyCallback(key)
			glfwWindow.SetMouseButtonCallback(screen.callbackForMousebutton)
			glfwWindow.SetMouseMovementCallback(screen.callbackForMousepos)
		}

		if newWidth != screen.rect.Dx() || newHeight != screen.rect.Dy() {

			screen.Resizex(newWidth, newHeight)
		}

		pixelRatio := float32(fbWidth) / float32(screen.rect.Dx())
		gl.Viewport(0, 0, fbWidth, fbHeight)

		// background color
		gl.ClearColor(0.8, 0.8, 0.8, 1)

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.Enable(gl.CULL_FACE)
		gl.Disable(gl.DEPTH_TEST)

		ctx.BeginFrame(screen.rect.Dx(), screen.rect.Dy(), pixelRatio)

		screen.CheckMouseInput()

		w := 40
		h := 40
		imagecolor := color.RGBA{100, 200, 200, 0xff} // cyan
		if (framenum/100)%2 == 0 {
			imagecolor = color.RGBA{200, 0, 0, 0xff}
		}

		for x := 0; x < w; x++ {
			for y := 0; y < h; y++ {
				screen.goimage.Set(x, y, imagecolor)
			}
		}

		screen.ggctx.Push()
		screen.ggctx.SetLineWidth(3.0)
		screen.ggctx.SetRGBA(0, 0, 1.0, 1.0)
		screen.ggctx.DrawLine(0, 0, 100, 100)
		screen.ggctx.Stroke()
		screen.ggctx.Pop()
		// im := screen.ggctx.Image()

		// screen.imageHandle = ctx.CreateImageFromGoImage(0, screen.goimage)
		ctx.UpdateImage(screen.imageHandle, screen.goimage.Pix)

		img := nanovgo.ImagePattern(0, 0, float32(fbWidth), float32(fbHeight), 0.0, screen.imageHandle, 1.0)

		ctx.Save()
		ctx.BeginPath()
		ctx.Circle(50, 50, 100)
		ctx.SetFillPaint(img)
		ctx.Fill()
		ctx.Stroke()

		// screen.Draw(ctx)

		ctx.Restore()

		ctx.EndFrame()

		gl.Enable(gl.DEPTH_TEST)
		glfwWindow.SwapBuffers()
		glfw.PollEvents()

		previousTime = tm
		tm = time.Now()
		dt := tm.Sub(previousTime)
		if dt < frametime {
			tosleep := frametime - dt
			// log.Printf("Sleeping for %s\n", tosleep)
			time.Sleep(tosleep)
		}

		framenum++
	}
	log.Printf("End of gui.Run()\n")
}

// LoadFonts xxx
func LoadFonts(ctx *nanovgo.Context) error {
	fonts := map[string]string{
		"icons":  "entypo.ttf",
		"lucida": "lucon.ttf",
	}
	hadErr := false
	for name, filename := range fonts {
		path := engine.ConfigFilePath(filename)
		f := ctx.CreateFont(name, path)
		if f == -1 {
			log.Printf("LoadFonts: could not add font name=%s path=%s", name, path)
			hadErr = true
		}
	}
	if hadErr {
		return fmt.Errorf("LoadFonts: unable to load all fonts")
	}
	return nil
}

func key(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// log.Printf("key=%d is ignored\n", key)
	/*
		if key == glfw.KeyEscape && action == glfw.Press {
			w.SetShouldClose(true)
		}
	*/
}
