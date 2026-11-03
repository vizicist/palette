//go:build windows

package kit

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSpoutProcessTreeHelper(t *testing.T) {
	if os.Getenv("PALETTE_SPOUT_TREE_CHILD") == "1" {
		time.Sleep(time.Minute)
		return
	}
	if os.Getenv("PALETTE_SPOUT_TREE_PARENT") != "1" {
		return
	}
	child := exec.Command(os.Args[0], "-test.run=^TestSpoutProcessTreeHelper$")
	child.Env = append(os.Environ(), "PALETTE_SPOUT_TREE_CHILD=1")
	child.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	fmt.Println(child.Process.Pid)
	time.Sleep(time.Minute)
}

func TestStopSpoutWindowKillsBootloaderAndChild(t *testing.T) {
	InitLog("")
	cmd := exec.Command(os.Args[0], "-test.run=^TestSpoutProcessTreeHelper$")
	cmd.Env = append(os.Environ(), "PALETTE_SPOUT_TREE_PARENT=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	childPIDText, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(childPIDText))
	if err != nil {
		t.Fatal(err)
	}
	defer KillByPid(childPID)

	window := &spoutWindow{cmd: cmd}
	stopSpoutWindow(window)
	_ = cmd.Wait()
	if window.cmd != nil {
		t.Fatal("receiver remained tracked after stop")
	}
	for _, pid := range []int{cmd.Process.Pid, childPID} {
		deadline := time.Now().Add(5 * time.Second)
		for IsPidRunning(pid) && time.Now().Before(deadline) {
			time.Sleep(50 * time.Millisecond)
		}
		if IsPidRunning(pid) {
			t.Errorf("process %d survived receiver stop", pid)
		}
	}
}
