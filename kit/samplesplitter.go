package kit

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

const samplesplitterPort = 9876
const samplesplitterMidiPort = "16. Internal MIDI"
const samplesplitterStatusTimeout = 500 * time.Millisecond

func SamplesplitterProcessInfo() *ProcessInfo {
	exe := SamplesplitterExecutablePath()
	if exe == "" {
		LogWarn("SamplesplitterProcessInfo: samplesplitter executable not found")
		return EmptyProcessInfo()
	}

	dir := SamplesplitterRuntimeDir(exe)
	mp3Dir := SamplesplitterMP3Dir(dir)
	arg := fmt.Sprintf("--dir %q --port %d --midi-port %q --no-open", mp3Dir, samplesplitterPort, samplesplitterMidiPort)
	pi := NewProcessInfo(filepath.Base(exe), exe, arg, nil)
	pi.DirPath = dir
	return pi
}

func SamplesplitterExecutablePath() string {
	candidates := []string{}
	if currentExe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(currentExe)
		candidates = append(candidates,
			filepath.Join(exeDir, SamplesplitterExe),
			filepath.Clean(filepath.Join(exeDir, "..", "bin", SamplesplitterExe)),
		)
	}
	if paletteDir := PaletteDir(); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "bin", SamplesplitterExe))
	}
	if paletteDir := os.Getenv("PALETTE"); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "bin", SamplesplitterExe))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter", SamplesplitterExe),
			filepath.Join(cwd, SamplesplitterExe),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", "..", "samplesplitter", SamplesplitterExe)),
			filepath.Clean(filepath.Join(cwd, "..", "..", SamplesplitterExe)),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func SamplesplitterRuntimeDir(exe string) string {
	exeDir := filepath.Dir(exe)
	candidates := []string{
		filepath.Clean(filepath.Join(exeDir, "..", "samplesplitter")),
		filepath.Clean(filepath.Join(exeDir, "..", "cmd", "samplesplitter")),
		exeDir,
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter"),
			filepath.Join(cwd, "samplesplitter"),
		)
	}
	for _, candidate := range candidates {
		if FileExists(filepath.Join(candidate, "static", "index.html")) {
			return candidate
		}
	}
	return exeDir
}

func SamplesplitterMP3Dir(runtimeDir string) string {
	candidates := []string{
		filepath.Join(runtimeDir, "mp3s"),
		filepath.Clean(filepath.Join(runtimeDir, "..", "data_default", "samplesplitter", "mp3s")),
		filepath.Clean(filepath.Join(runtimeDir, "..", "..", "data_default", "samplesplitter", "mp3s")),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "data_default", "samplesplitter", "mp3s"),
			filepath.Clean(filepath.Join(cwd, "..", "data_default", "samplesplitter", "mp3s")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "data_default", "samplesplitter", "mp3s")),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return filepath.Join(runtimeDir, "mp3s")
}

func SamplesplitterScriptPath() string {
	candidates := []string{}
	if currentExe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(currentExe)
		candidates = append(candidates,
			filepath.Clean(filepath.Join(exeDir, "..", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(exeDir, "..", "..", "samplesplitter", "samplesplitter.py")),
		)
	}
	if paletteDir := PaletteDir(); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "samplesplitter", "samplesplitter.py"))
	}
	if paletteDir := os.Getenv("PALETTE"); paletteDir != "" {
		candidates = append(candidates, filepath.Join(paletteDir, "samplesplitter", "samplesplitter.py"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "cmd", "samplesplitter", "samplesplitter.py"),
			filepath.Join(cwd, "samplesplitter", "samplesplitter.py"),
			filepath.Clean(filepath.Join(cwd, "..", "cmd", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "cmd", "samplesplitter", "samplesplitter.py")),
			filepath.Clean(filepath.Join(cwd, "..", "..", "samplesplitter", "samplesplitter.py")),
		)
	}
	for _, candidate := range candidates {
		if FileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func samplesplitterWebIsListening() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", samplesplitterPort), samplesplitterStatusTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
