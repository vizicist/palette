package main

import (
	"flag"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vizicist/palette/engine"
)

func main() {

	signal.Ignore(syscall.SIGHUP)
	signal.Ignore(syscall.SIGINT)

	engine.InitLogs()
	engine.InitDebug()

	log.SetFlags(log.Ldate | log.Lmicroseconds)
	log.Print("Palette is started")

	flag.Parse()

	engine.InitMIDI()

	go engine.StartNATSServer()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine.ListenForever()
	}()

	go engine.StartOSC("127.0.0.1:3333")
	go engine.StartNATSClient()
	go engine.StartMIDI()
	go engine.StartRealtime()
	go engine.StartCursorInput()

	wg.Wait()
	log.Printf("After wg.Wait()\n")
}
