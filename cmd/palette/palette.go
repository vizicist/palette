package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hypebeast/go-osc/osc"
	"github.com/vizicist/palette/engine"
)

func usage() {
	usage := "Usage: palette {start|stop|sendlogs} {engine|gui|bidule|all|activate}\n"
	os.Stderr.WriteString(usage)
	log.Fatal(usage)
}

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLog("palette")

	flag.Parse()

	args := flag.Args()
	nargs := len(args)

	recipient := engine.ConfigValue("emailto")
	login := engine.ConfigValue("emaillogin")
	password := engine.ConfigValue("emailpassword")

	switch nargs {

	case 1:
		switch args[0] {
		case "sendlogs":
			engine.SendLogs(recipient, login, password)
		default:
			usage()
		}
		return

	case 2:

		body := ""
		if recipient != "" && login != "" && password != "" {
			body = fmt.Sprintf("host=%s palette executed args = %v", engine.Hostname(), args)
			engine.SendMail(recipient, login, password, body, "")
		}

		switch args[0] {

		case "start":
			handle_startstop(args[0], args[1:])
			// Always send logs after starting all
			if args[1] == "all" {
				engine.SendLogs(recipient, login, password)
			}

		case "stop":
			handle_startstop(args[0], args[1:])

		default:
			usage()
		}

	default:
		usage()
	}
}

// handle_activate sends OSC messages to start the layers in Resolume,
// and make sure the audio is on in Bidule.
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
	msg := osc.NewMessage("/play")
	msg.Append(int32(1)) // turn it on
	// log.Printf("Sending %s\n", msg.String())
	err := biduleClient.Send(msg)
	if err != nil {
		log.Printf("handle_activate: osc to Bidule, err=%s\n", err)
	}
}

func start_layer(resolumeClient *osc.Client, layer int) {
	addr := fmt.Sprintf("/composition/layers/%d/clips/1/connect", layer)
	msg := osc.NewMessage(addr)
	msg.Append(int32(1))
	// log.Printf("Sending %s\n", msg.String())
	err := resolumeClient.Send(msg)
	if err != nil {
		log.Printf("start_layer: osc to Resolume err=%s\n", err)
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

	case "all", "allsmall":
		handle_startstop(startstop, []string{"resolume"})
		handle_startstop(startstop, []string{"bidule"})
		handle_startstop(startstop, []string{"engine"})

		// Give resolume time to start, before starting
		// GUI and activating things.
		if startstop == "start" {
			time.Sleep(12 * time.Second)
			handle_startstop(startstop, []string{"activate"})
		}

		// Start the GUI.
		if cmd == "allsmall" {
			handle_startstop(startstop, []string{"guismall"})
		} else {
			handle_startstop(startstop, []string{"gui"})
		}

		// Sometimes the audio in bidule is still turned off
		// (perhaps when it hasn't completed its startup by
		// the time the first activate is sent), so
		// send a few more activates to make sure
		if startstop == "start" {
			time.Sleep(24 * time.Second)
			handle_startstop(startstop, []string{"activate"})
			time.Sleep(24 * time.Second)
			handle_startstop(startstop, []string{"activate"})
			time.Sleep(24 * time.Second)
			handle_startstop(startstop, []string{"activate"})
			time.Sleep(24 * time.Second)
			handle_startstop(startstop, []string{"activate"})
		}

		return

	case "activate":
		if startstop == "start" {
			handle_activate()
		}
		return

	case "gui":
		exe = "palette_gui.exe"
		fullexe = filepath.Join(palette, "bin", "pyinstalled", exe)
		exearg = "large" // default
		// don't return

	case "guismall":
		exe = "palette_gui.exe"
		fullexe = filepath.Join(palette, "bin", "pyinstalled", exe)
		exearg = "small"
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
