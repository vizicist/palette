package main

import (
	"flag"
	_ "image/png"
	"log"
	"time"

	"github.com/faiface/mainthread"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vizicist/palette/glhf"
	"github.com/vizicist/palette/isf"
	"github.com/vizicist/palette/kit"
	// _ "github.com/vizicist/palette/twinsys"
)

const windowWidth = 800
const windowHeight = 600

func main() {
	mainthread.Run(run)
}

// var printOnce sync.Once

func run() {

	kit.InitLog("isf")
	kit.InitMisc()
	kit.InitEngine()
	kit.StartEngine()

	log.Printf("Montage_ISF: starting\n")

	visible := flag.Bool("visible", true, "Window(s) are visible")
	_ = visible

	flag.Parse()

	realtime, err := isf.NewRealtime()
	if err != nil {
		log.Fatalf("NewRealtime: err=%s\n", err)
	}

	// kit.LoadParamEnums()
	// kit.LoadParamDefs()

	go realtime.Start()

	defer func() {
		mainthread.Call(func() {
			glfw.Terminate()
		})
	}()

	var win *glfw.Window
	var pipeline *isf.Pipeline

	// go twinsys.RunEbiten()
	// kit.LogInfo("Avoiding RunEbiten", "version", twinsys.Version)

	mainthread.Call(func() {

		glfw.Init()

		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, glfw.False)

		var err error
		win, err = glfw.CreateWindow(windowWidth, windowHeight, "Palette ISF", nil, nil)
		if err != nil {
			panic(err)
		}

		win.MakeContextCurrent()

		glhf.Init()

		/*
			printOnce.Do(func() {
				version := gl.GoStr(gl.GetString(gl.VERSION))
				log.Println("OpenGL version", version)
			})
		*/

		pipeline, err = isf.NewPipeline("A", windowWidth, windowHeight, true)
		if err != nil {
			log.Printf("pipeline error: %s\n", err)
			panic(err)
		}

		vizlet := pipeline.FirstVizlet()
		// vizlet.SetSpoutIn("vizspout.static")
		vizlet.SetSpoutIn("vizspout.moving")
		// v.Reactors["A"].SetTargetVizlet(vizlet)

	})

	frametime, _ := time.ParseDuration("16ms")
	tm := time.Now()
	previousTime := tm
	// elapsed := time.Duration(0)

	// mainthread.Call(func() {
	// 	twinsys.RunEbiten()
	// })

	shouldQuit := false
	for !shouldQuit {
		mainthread.Call(func() {
			if win.ShouldClose() {
				shouldQuit = true
			}

			pipeline.Do()

			win.SwapBuffers()
			glfw.PollEvents()

			previousTime = tm
			tm = time.Now()
			// elapsed = tm.Sub(previousTime)
			dt := tm.Sub(previousTime)
			if dt < frametime {
				tosleep := frametime - dt
				// log.Printf("Sleeping for %s\n", tosleep)
				time.Sleep(tosleep)
			}
		})
	}
	log.Printf("Venue has stopped\n")
}
