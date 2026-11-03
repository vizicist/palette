package kit

import (
	"fmt"
	"net"
	"testing"

	"github.com/hypebeast/go-osc/osc"
)

func TestGUILevelLimitsSpoutWindowsToPerformance(t *testing.T) {
	oldWriter := writeSpoutPerformanceActive
	var states []bool
	writeSpoutPerformanceActive = func(active bool) error {
		states = append(states, active)
		return nil
	}
	t.Cleanup(func() { writeSpoutPerformanceActive = oldWriter })
	for _, level := range []int{0, 1, 2, 1, 0, 2} {
		if _, err := globalGUILevel("guilevel", map[string]string{"level": fmt.Sprint(level)}); err != nil {
			t.Fatal(err)
		}
		if states[len(states)-1] != (level == 2) {
			t.Fatalf("Spout windows have wrong visibility at GUI level %d", level)
		}
	}
	for _, level := range []string{"", "wrong", "-1", "3"} {
		if _, err := globalGUILevel("guilevel", map[string]string{"level": level}); err == nil {
			t.Fatalf("invalid GUI level %q was accepted", level)
		}
	}
	if len(states) != 6 {
		t.Fatal("invalid GUI levels changed native display visibility")
	}
}

func TestSpoutPreviewDisplayFollowsPreviewModeRatherThanDemoPlayback(t *testing.T) {
	oldPatches, oldWriter := Patchs, writeSpoutPreviewActive
	a, b := &Patch{name: "A"}, &Patch{name: "B"}
	Patchs = map[string]*Patch{"A": a, "B": b}
	var active bool
	writeSpoutPreviewActive = func(value bool) error { active = value; return nil }
	t.Cleanup(func() { Patchs, writeSpoutPreviewActive = oldPatches, oldWriter })
	check := func(want bool) {
		t.Helper()
		if err := updateSpoutPreviewDisplay(); err != nil {
			t.Fatal(err)
		}
		if active != want {
			t.Fatalf("display active=%v, want %v", active, want)
		}
	}
	check(false)
	a.preview.setEnabled(true)
	check(true)
	// Taking over with real gestures stops the demo, but the layer stays visible.
	a.preview.realInput()
	check(true)
	b.preview.setEnabled(true)
	a.preview.setEnabled(false)
	check(true)
	b.preview.setEnabled(false)
	check(false)
}

func TestPatchPreviewRollsBackIfDisplaySignalFails(t *testing.T) {
	InitLog("")
	oldPatches, oldWriter := Patchs, writeSpoutPreviewActive
	patch := &Patch{name: "A"}
	Patchs = map[string]*Patch{"A": patch}
	var writes []bool
	writeSpoutPreviewActive = func(active bool) error {
		writes = append(writes, active)
		if active {
			return fmt.Errorf("test signal failure")
		}
		return nil
	}
	t.Cleanup(func() { Patchs, writeSpoutPreviewActive = oldPatches, oldWriter })
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	resolume := TheResolume()
	oldClient := resolume.resolumeClient
	resolume.resolumeClient = osc.NewClient("127.0.0.1", listener.LocalAddr().(*net.UDPAddr).Port)
	t.Cleanup(func() { resolume.resolumeClient = oldClient })
	if _, err := patch.API("preview", map[string]string{"onoff": "true"}); err == nil {
		t.Fatal("expected the display error to reach the caller")
	}
	if patch.preview.isEnabled() || len(writes) != 2 || !writes[0] || writes[1] {
		t.Fatalf("Preview did not roll back: enabled=%v writes=%v", patch.preview.isEnabled(), writes)
	}
}
