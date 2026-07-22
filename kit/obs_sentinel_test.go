package kit

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setFakeObsConfigRoot points ObsConfigDir at a temp directory on whichever
// OS the test is running, and returns that directory.
func setFakeObsConfigRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	switch runtime.GOOS {
	case "darwin":
		t.Setenv("HOME", root)
		return filepath.Join(root, "Library", "Application Support")
	case "windows":
		t.Setenv("APPDATA", root)
		return root
	default:
		t.Setenv("XDG_CONFIG_HOME", root)
		return root
	}
}

func TestDeleteObsSentinelRemovesStaleDirectory(t *testing.T) {
	configRoot := setFakeObsConfigRoot(t)

	sentinelPath := filepath.Join(configRoot, "obs-studio", ".sentinel")
	if err := os.MkdirAll(sentinelPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sentinelPath, "run_stale"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := deleteObsSentinel(); err != nil {
		t.Fatalf("deleteObsSentinel() error = %v", err)
	}
	if _, err := os.Lstat(sentinelPath); !os.IsNotExist(err) {
		t.Fatalf("sentinel still exists after cleanup: %v", err)
	}
}

func TestDeleteObsSentinelAllowsAlreadyAbsentDirectory(t *testing.T) {
	setFakeObsConfigRoot(t)

	if err := deleteObsSentinel(); err != nil {
		t.Fatalf("deleteObsSentinel() error = %v", err)
	}
}

func TestDeleteObsSentinelRequiresConfigDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", "")
	default:
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")
	}

	if err := deleteObsSentinel(); err == nil {
		t.Fatal("deleteObsSentinel() error = nil, want missing-config-dir error")
	}
}

func TestStartRunningRunsPreLaunchHookBeforeProcessCreation(t *testing.T) {
	pm := NewProcessManager()
	wantErr := errors.New("sentinel cleanup failed")
	hookCalled := false
	pi := NewProcessInfo(
		"palette-prelaunch-test-process-that-is-not-running",
		filepath.Join(t.TempDir(), "must-not-be-started"),
		"",
		nil,
	)
	pi.BeforeLaunch = func() error {
		hookCalled = true
		return wantErr
	}
	pm.AddProcess("test", pi)

	err := pm.StartRunning("test")
	if !hookCalled {
		t.Fatal("StartRunning() did not call the pre-launch hook")
	}
	if err == nil || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("StartRunning() error = %v, want pre-launch error", err)
	}
}
