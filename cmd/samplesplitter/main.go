package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vizicist/palette/cmd/samplesplitter/internal/samplesplitter"
)

func main() {
	config := samplesplitter.DefaultConfig()
	flag.StringVar(&config.MP3Dir, "dir", "mp3s", "Directory containing MP3 files")
	flag.IntVar(&config.Port, "port", samplesplitter.DefaultPort, "HTTP port")
	flag.IntVar(&config.BaseNote, "base-note", samplesplitter.DefaultBaseNote, "MIDI base note")
	flag.StringVar(&config.MIDIPortName, "midi-port", config.MIDIPortName, "MIDI input port to listen to on startup")
	noOpen := flag.Bool("no-open", false, "Accepted for Python CLI compatibility; browser opening is not implemented yet")
	flag.Parse()
	_ = noOpen

	if err := config.Normalize(); err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(config.MP3Dir)
	if err != nil || !info.IsDir() {
		log.Fatalf("directory not found: %s", config.MP3Dir)
	}

	config.FFmpegPath = findFFmpeg()
	analyzer := samplesplitter.Analyzer{FFmpegPath: config.FFmpegPath}
	state := samplesplitter.NewState(config)
	state.LoadSigilDefaults(analyzer, rand.New(rand.NewSource(time.Now().UnixNano())))
	state.LoadFirstIfEmpty(analyzer)

	audioManager, err := samplesplitter.NewAudioManager(config.FFmpegPath, state)
	if err != nil {
		state.SetAudioStatus(false, err)
		log.Printf("Audio disabled: %v", err)
	} else {
		defer audioManager.Close()
		state.SetAudioStatus(true, nil)
		_, defaultID, _ := audioManager.Devices()
		if name, err := audioManager.SetOutput(defaultID); err == nil {
			fmt.Printf("Audio output: %s\n", name)
		}
	}

	midiManager, err := samplesplitter.NewMIDIManager(state, audioManager)
	if err != nil {
		state.SetMIDIStatus("", err)
		log.Printf("MIDI disabled: %v", err)
	} else {
		defer midiManager.Close()
		if config.MIDIPortName != "" {
			if resolved, err := midiManager.Start(config.MIDIPortName); err != nil {
				log.Printf("MIDI input %q unavailable: %v", config.MIDIPortName, err)
			} else {
				fmt.Printf("MIDI input: %s\n", resolved)
			}
		}
	}

	staticDir := filepath.Join(mustCwd(), "static")
	server := samplesplitter.Server{
		State:     state,
		Analyzer:  analyzer,
		StaticDir: staticDir,
		MIDI:      midiManager,
		Audio:     audioManager,
	}

	addr := fmt.Sprintf("localhost:%d", config.Port)
	fmt.Printf("Sample Splitter Go port running at http://%s\n", addr)
	fmt.Printf("MP3 directory: %s\n", config.MP3Dir)
	log.Fatal(http.ListenAndServe(addr, server.Handler()))
}

func findFFmpeg() string {
	cwd := mustCwd()
	name := "ffmpeg"
	if filepath.Separator == '\\' {
		name = "ffmpeg.exe"
	}
	local := filepath.Join(cwd, "ffmpeg", "bin", name)
	if _, err := os.Stat(local); err == nil {
		return local
	}
	return "ffmpeg"
}

func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return cwd
}
