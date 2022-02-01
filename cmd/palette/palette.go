package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	word0 := ""
	word1 := ""
	if nargs > 0 {
		word0 = args[0]
	}
	if nargs > 1 {
		word1 = args[1]
	}

	sendlogs := false
	sendmail := false

	switch word0 {

	case "sendlogs":
		sendlogs = true

	case "start":

		sendmail = true
		handle_start(word1)
		sendlogs = (word1 == "all") // Always send logs after starting all

	case "stop":
		sendmail = true
		handle_stop(word1)

	default:
		usage()
	}

	if sendmail {
		body := fmt.Sprintf("host=%s palette executed args = %v", engine.Hostname(), args)
		engine.SendMail(body)
	}

	if sendlogs {
		engine.SendLogs()
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

func handle_start(cmd string) {

	// exe := ""
	fullexe := ""
	exearg := ""

	palette := engine.PaletteDir()

	switch cmd {

	case "all", "allsmall":
		handle_start("resolume_only")
		handle_start("bidule")
		handle_start("engine")

		// Give resolume time to start, before starting
		// GUI and activating things.
		time.Sleep(12 * time.Second)
		handle_start("activate")

		// Start the GUI.
		if cmd == "allsmall" {
			handle_start("guismall")
		} else {
			handle_start("gui")
		}

		// Sometimes the audio in bidule is still turned off
		// (perhaps when it hasn't completed its startup by
		// the time the first activate is sent), so
		// send a few more activates to make sure
		for i := 0; i < 4; i++ {
			time.Sleep(24 * time.Second)
			handle_start("activate")
		}

		return

	case "activate":
		handle_activate()
		return

	case "gui":
		fullexe = filepath.Join(palette, "bin", "pyinstalled", "palette_gui.exe")
		// don't return

	case "engine":
		fullexe = filepath.Join(palette, "bin", "palette_engine.exe")
		// don't return

	case "bidule":
		fullexe = engine.ConfigValue("bidule")
		if fullexe == "" {
			fullexe = "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_X64.exe"
		}
		exearg = engine.ConfigFilePath("palette.bidule")
		if !engine.FileExists(fullexe) {
			log.Printf("No Bidule found, looking for %s\n", fullexe)
			return
		}
		// don't return

	case "resolume":
		handle_start("resolume_only")

		// Give resolume time to start, before starting
		// GUI and activating things.
		time.Sleep(12 * time.Second)
		handle_start("activate")

	case "resolume_only":
		fullexe = engine.ConfigValue("resolume")
		if fullexe != "" && !engine.FileExists(fullexe) {
			log.Printf("No Resolume found, looking for %s\n", fullexe)
			return
		}
		if fullexe == "" {
			fullexe = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
			if !engine.FileExists(fullexe) {
				fullexe = "C:\\Program Files\\Resolume Arena\\Arena.exe"
				if !engine.FileExists(fullexe) {
					log.Printf("No Resolume found in default locations\n")
					return
				}
			}
		}
		// don't return

	default:
		log.Printf("Unknown target of start command.\n")
		return
	}

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
}

func handle_stop(cmd string) {

	exeToStop := ""

	palette := engine.PaletteDir()

	switch cmd {

	case "all", "allsmall":
		handle_stop("resolume_only")
		handle_stop("bidule")
		handle_stop("engine")

		if cmd == "allsmall" {
			handle_stop("guismall")
		} else {
			handle_stop("gui")
		}

	case "activate":
		// nothing to stop

	case "gui":
		exeToStop = filepath.Join(palette, "bin", "pyinstalled", "palette_gui.exe")

	case "guismall":
		exeToStop = filepath.Join(palette, "bin", "pyinstalled", "palette_gui.exe")

	case "engine":
		exeToStop = filepath.Join(palette, "bin", "palette_engine.exe")

	case "bidule":
		exeToStop = engine.ConfigValue("bidule")
		if exeToStop == "" {
			exeToStop = "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_X64.exe"
		}
		if !engine.FileExists(exeToStop) {
			log.Printf("No Bidule found, looking for %s\n", exeToStop)
			return
		}

	case "resolume", "resolume_only":
		exeToStop = engine.ConfigValue("resolume")
		if exeToStop != "" && !engine.FileExists(exeToStop) {
			log.Printf("No Resolume found, looking for %s\n", exeToStop)
			return
		}
		if exeToStop == "" {
			exeToStop = "C:\\Program Files\\Resolume Avenue\\Avenue.exe"
			if !engine.FileExists(exeToStop) {
				exeToStop = "C:\\Program Files\\Resolume Arena\\Arena.exe"
				if !engine.FileExists(exeToStop) {
					log.Printf("No Resolume found in default locations\n")
					return
				}
			}
		}

	default:
		log.Printf("Unknown target of stop command.\n")
		return
	}

	if exeToStop != "" {
		log.Printf("Stopping %s\n", cmd)
		// isolate the last part of exe path
		justexe := exeToStop
		lastslash := strings.LastIndexAny(exeToStop, "/\\")
		if lastslash >= 0 {
			justexe = exeToStop[lastslash+1:]
		}
		engine.StopExecutable(justexe)
	}

}
