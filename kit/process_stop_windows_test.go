package kit

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestKillExecutableHelper(t *testing.T) {
	if os.Getenv("PALETTE_STOP_TEST_HELPER") != "1" {
		return
	}
	fmt.Println("ready")
	time.Sleep(time.Minute)
	os.Exit(0)
}

func TestKillExecutableStopsAllMatchesWithoutTaskkill(t *testing.T) {
	InitLog("")
	// Give the children a unique image name so the test cannot stop another
	// test runner or any of the user's applications.
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	image := filepath.Join(t.TempDir(), fmt.Sprintf("palette-stop-%d.exe", time.Now().UnixNano()))
	output, err := os.Create(image)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		t.Fatalf("copy helper: %v, %v", copyErr, closeErr)
	}
	t.Setenv("PATH", t.TempDir())
	for i := 0; i < 2; i++ {
		cmd := exec.Command(image, "-test.run=^TestKillExecutableHelper$")
		cmd.Env = append(os.Environ(), "PALETTE_STOP_TEST_HELPER=1")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("helper did not exit")
			}
		})
		ready := make(chan error, 1)
		go func() {
			line, err := bufio.NewReader(stdout).ReadString('\n')
			if err == nil && strings.TrimSpace(line) != "ready" {
				err = fmt.Errorf("unexpected helper output %q", line)
			}
			ready <- err
		}()
		select {
		case err := <-ready:
			if err != nil {
				t.Fatalf("helper failed to start: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("helper did not become ready")
		}
	}
	if err := KillExecutable(strings.ToUpper(image)); err != nil {
		t.Fatal(err)
	}
	if running, err := IsRunningExecutable(image); err != nil || running {
		t.Fatalf("after stop: running=%v, err=%v", running, err)
	}
	// Stopping an already stopped application must still succeed.
	if err := KillExecutable(image); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestKillExecutableRejectsEmptyName(t *testing.T) {
	InitLog("")
	if err := KillExecutable(""); err == nil {
		t.Fatal("empty name must return an error")
	}
}
