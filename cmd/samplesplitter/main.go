package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/vizicist/palette/pkg/samplesplitter"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

func main() {
	config := samplesplitter.DefaultConfig()
	flag.StringVar(&config.MP3Dir, "dir", samplesplitter.DefaultMP3Dir(), "Ignored; MP3 directory is always %USERPROFILE%\\mp3s")
	flag.IntVar(&config.Port, "port", samplesplitter.DefaultPort, "HTTP port")
	flag.IntVar(&config.BaseNote, "base-note", samplesplitter.DefaultBaseNote, "MIDI base note")
	flag.StringVar(&config.MIDIPortName, "midi-port", config.MIDIPortName, "MIDI input port to listen to on startup")
	noOpen := flag.Bool("no-open", false, "Accepted for Python CLI compatibility; browser opening is not implemented yet")
	flag.Parse()
	_ = noOpen
	config.MP3Dir = samplesplitter.DefaultMP3Dir()

	if err := config.Normalize(); err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(config.MP3Dir)
	if err != nil || !info.IsDir() {
		log.Fatalf("directory not found: %s", config.MP3Dir)
	}

	staticDir := filepath.Join(mustCwd(), "static")
	service, err := samplesplitter.NewService(samplesplitter.ServiceOptions{
		Config:       config,
		StaticDir:    staticDir,
		EnableMIDI:   true,
		EnableHTTP:   false,
		SelectOutput: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer service.Close()
	if err := service.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	server := samplesplitter.ServerFromService(service, staticDir)

	addr := fmt.Sprintf("localhost:%d", config.Port)
	fmt.Printf("Sample Splitter Go port running at http://%s\n", addr)
	fmt.Printf("MP3 directory: %s\n", config.MP3Dir)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}

func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return cwd
}
