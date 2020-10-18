package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vizicist/palette/palette"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	palette.SetRootPath(os.Getenv("PALETTE"))

	palette.InitLogs(palette.ConfigValue("logfile"))
	palette.InitDebug(palette.ConfigValue("debug"))

	log.SetFlags(log.Ldate | log.Lmicroseconds)
	log.Print("palette.go begins")

	flag.Parse()

	palette.InitMIDI()

	go palette.StartNATSServer()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		palette.ListenForever()
	}()

	go palette.StartOSC("127.0.0.1:3333")
	go palette.StartNATSClient()
	go palette.StartMIDI()
	go palette.StartRealtime()
	go palette.StartCursorInput()

	wg.Wait()
	log.Printf("After wg.Wait()\n")
}
