//go:build windows

package kit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type spoutWindow struct {
	cmd    *exec.Cmd
	bounds string
}

var spoutWindows = struct {
	sync.Mutex
	windows  [2]spoutWindow
	watching bool
}{}

// The GUI sends the screen coordinates of its two reserved video surfaces.
// Keep the native windows in those surfaces while Performance mode is active.
func syncSpoutWindows(level int, args map[string]string) error {
	spoutWindows.Lock()
	defer spoutWindows.Unlock()
	if level != 2 {
		for i := range spoutWindows.windows {
			stopSpoutWindow(&spoutWindows.windows[i])
		}
		return nil
	}

	composition := args["composition"]
	preview := args["preview"]
	if composition == "" || preview == "" {
		return nil // Older API callers only update the visibility event.
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := filepath.Join(filepath.Dir(exe), "palette_spout_receiver.exe")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("Spout receiver at %s: %w", path, err)
	}
	var parsedBounds [2][]string
	for i, bounds := range []string{composition, preview} {
		parts := strings.Split(bounds, ",")
		if len(parts) != 4 {
			return fmt.Errorf("invalid Spout window bounds %q", bounds)
		}
		for j, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil || (j >= 2 && n <= 0) {
				return fmt.Errorf("invalid Spout window bounds %q", bounds)
			}
		}
		parsedBounds[i] = parts
	}
	for i, item := range []struct {
		bounds  string
		sender  string
		preview bool
	}{
		{composition, "Avenue - Screen 3", false},
		{preview, "Avenue - Screen 2", true},
	} {
		parts := parsedBounds[i]
		window := &spoutWindows.windows[i]
		if window.cmd != nil && window.bounds == item.bounds {
			continue
		}
		stopSpoutWindow(window)
		commandArgs := []string{"--sender", item.sender, "--x", parts[0], "--y", parts[1], "--width", parts[2], "--height", parts[3], "--always-on-top", "--performance-only"}
		if item.preview {
			commandArgs = append(commandArgs, "--preview-only")
		}
		cmd := exec.Command(path, commandArgs...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start Spout receiver for %s: %w", item.sender, err)
		}
		window.cmd = cmd
		window.bounds = item.bounds
		go func(window *spoutWindow, cmd *exec.Cmd) {
			err := cmd.Wait()
			spoutWindows.Lock()
			if window.cmd == cmd {
				window.cmd = nil
				window.bounds = ""
				if err != nil {
					LogWarn("Spout receiver exited", "sender", item.sender, "err", err)
				}
			}
			spoutWindows.Unlock()
		}(window, cmd)
	}
	if !spoutWindows.watching {
		spoutWindows.watching = true
		go watchSpoutGUI()
	}
	return nil
}

func stopSpoutWindow(window *spoutWindow) {
	if window.cmd != nil && window.cmd.Process != nil {
		// PyInstaller runs the GUI in a child process. Stopping only its
		// bootloader parent leaves a window and receiver behind.
		KillByPid(window.cmd.Process.Pid)
	}
	window.cmd = nil
	window.bounds = ""
}

func watchSpoutGUI() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	missing := 0
	for range ticker.C {
		spoutWindows.Lock()
		active := spoutWindows.windows[0].cmd != nil || spoutWindows.windows[1].cmd != nil
		if !active {
			spoutWindows.watching = false
			spoutWindows.Unlock()
			return
		}
		spoutWindows.Unlock()
		if FindWindowByTitleContains("Palette Control") {
			missing = 0
			continue
		}
		missing++
		if missing >= 3 {
			LogIfError(syncSpoutWindows(0, nil))
			LogIfError(writeSpoutPerformanceActive(false))
			missing = 0
		}
	}
}
