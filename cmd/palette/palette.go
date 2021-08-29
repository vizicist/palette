package main

import (
	"flag"
	"fmt"
	"log"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("palette")

	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		log.Fatal("Usage: palette {start|stop} {gui|engine|bidule|activate}\n")
	}
	nargs := len(args)
	if nargs == 0 {
		log.Printf("Expecting argument to start\n")
		return
	}

	switch args[0] {
	case "start", "stop":
		handle_startstop(args[0], args[1:])
	default:
		log.Fatalf("Unrecognized command: %s\n", args[0])
	}
}

func handle_activate() {

	log.Printf("Activating\n")

	addr := "127.0.0.1"
	resolumePort := 7000
	bidulePort := 3210

	resolumeClient := osc.NewClient(addr, resolumePort)
	start_layer(resolumeClient, 1)
	start_layer(resolumeClient, 2)
	start_layer(resolumeClient, 3)
	start_layer(resolumeClient, 4)

	biduleClient := osc.NewClient(addr, bidulePort)
	start_play(biduleClient, 1)
}

func start_play(biduleClient *osc.Client, onoff int) {
	msg := osc.NewMessage("/play")
	msg.Append(int32(onoff))
	// log.Printf("Sending %s\n", msg.String())
	err := biduleClient.Send(msg)
	if err != nil {
		log.Printf("start_play: err=%s\n", err)
	}
}

func start_layer(resolumeClient *osc.Client, layer int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/connect", layer)
	msg := osc.NewMessage(addr)
	msg.Append(int32(1))
	// log.Printf("Sending %s\n", msg.String())
	err := resolumeClient.Send(msg)
	if err != nil {
		log.Printf("start_layer: err=%s\n", err)
	}
}

func handle_startstop(startstop string, args []string) {

	nargs := len(args)
	if nargs == 0 {
		log.Printf("Expecting argument to start command\n")
		return
	}
	exe := ""
	fullexe := ""
	cmd := args[0]
	exearg := ""

	palette := engine.PaletteDir()

	switch cmd {

	case "all":
		handle_startstop(startstop, []string{"resolume"})
		handle_startstop(startstop, []string{"bidule"})
		handle_startstop(startstop, []string{"engine"})
		handle_startstop(startstop, []string{"gui"})
		// Give resolume and bidule some time to start
		time.Sleep(10000 * time.Millisecond)
		handle_startstop(startstop, []string{"activate"})
		return

	case "activate":
		if startstop == "start" {
			handle_activate()
		}
		return

	case "gui":
		if nargs > 1 {
			exearg = args[1]
		} else {
			exearg = "large" // default
		}
		exe = "palette_gui.exe"
		fullexe = filepath.Join(palette, "bin", "pyinstalled", exe)
		// don't return

	case "engine":
		exe = "palette_engine.exe"
		fullexe = filepath.Join(palette, "bin", exe)
		// don't return

	case "bidule":
		exe = "PlogueBidule_x64.exe"
		fullexe = "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_X64.exe"
		exearg = engine.ConfigFilePath("palette.bidule")
		if !engine.FileExists(fullexe) {
			log.Printf("No Bidule found\n")
			return
		}
		// don't return

	case "resolume":
		exe = "Avenue.exe"
		fullexe = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
		if !engine.FileExists(fullexe) {
			exe = "Arena.exe"
			fullexe = "C:\\Program Files\\Resolume Arena\\Arena.exe"
			if !engine.FileExists(fullexe) {
				log.Printf("No Resolume found\n")
				return
			}
		}
		// don't return

	default:
		log.Printf("Unknown target of start command.\n")
		return
	}

	switch startstop {
	case "start":
		stdoutWriter := engine.MakeFileWriter(engine.LogFilePath(cmd + ".stdout"))
		if stdoutWriter == nil {
			stdoutWriter = &engine.NoWriter{}
		}
		stderrWriter := engine.MakeFileWriter(engine.LogFilePath(cmd + ".stderr"))
		if stderrWriter == nil {
			stderrWriter = &engine.NoWriter{}
		}
		log.Printf("Starting %s\n", fullexe)
		engine.StartExecutable(fullexe, true, stdoutWriter, stderrWriter, exearg)
	case "stop":
		log.Printf("Stopping %s\n", exe)
		engine.StopExecutable(exe)

	default:
		log.Printf("Unknown syntax of start command.\n")
	}
}
