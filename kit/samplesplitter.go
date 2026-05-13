package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const samplesplitterPort = 9876
const samplesplitterMidiPort = "16. Internal MIDI"

func SamplesplitterProcessInfo() *ProcessInfo {
	script := SamplesplitterScriptPath()
	if script == "" {
		LogWarn("SamplesplitterProcessInfo: samplesplitter.py not found")
		return EmptyProcessInfo()
	}

	python := "python3"
	arg := fmt.Sprintf("%q --port %d --midi-port %q --no-open", script, samplesplitterPort, samplesplitterMidiPort)
	if runtime.GOOS == "windows" {
		python = "py"
		arg = fmt.Sprintf("-3.11 %q --port %d --midi-port %q --no-open", script, samplesplitterPort, samplesplitterMidiPort)
	}

	pi := NewProcessInfo(python, python, arg, nil)
	pi.DirPath = filepath.Dir(script)
	return pi
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
			filepath.Join(cwd, "samplesplitter", "samplesplitter.py"),
			filepath.Clean(filepath.Join(cwd, "..", "samplesplitter", "samplesplitter.py")),
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
