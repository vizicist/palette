package main

import (
	"flag"
	_ "image/png"
	"log"
	"sync"
	"time"

	"github.com/faiface/mainthread"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/vizicist/palette/glhf"
	"github.com/vizicist/palette/isf"
)

const windowWidth = 800
const windowHeight = 600

func main() {
	mainthread.Run(run)
}

var printOnce sync.Once

func run() {

	log.Printf("main() started")

	visible := flag.Bool("visible", true, "Window(s) are visible")
	_ = visible

	flag.Parse()

	v, err := isf.NewVenue("defaultvenue")
	if err != nil {
		log.Fatalf("NewVenue: err=%s\n", err)
	}

	go v.Start()

	defer func() {
		mainthread.Call(func() {
			glfw.Terminate()
		})
	}()

	var win *glfw.Window
	var pipeline *isf.Pipeline

	mainthread.Call(func() {

		glfw.Init()

		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
		glfw.WindowHint(glfw.Resizable, glfw.False)

		var err error
		win, err = glfw.CreateWindow(windowWidth, windowHeight, "VizVenue", nil, nil)
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
