package kit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteObsSentinelRemovesStaleDirectory(t *testing.T) {
	appData := t.TempDir()
	t.Setenv("APPDATA", appData)

	sentinelPath := filepath.Join(appData, "obs-studio", ".sentinel")
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
	t.Setenv("APPDATA", t.TempDir())

	if err := deleteObsSentinel(); err != nil {
		t.Fatalf("deleteObsSentinel() error = %v", err)
	}
}

func TestDeleteObsSentinelRequiresAppData(t *testing.T) {
	t.Setenv("APPDATA", "")

	if err := deleteObsSentinel(); err == nil {
		t.Fatal("deleteObsSentinel() error = nil, want APPDATA error")
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
