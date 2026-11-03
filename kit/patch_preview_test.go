package kit

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

func TestPatchPreviewSendsResolumeLayerOSC(t *testing.T) {
	InitLog("")
	oldPatches, oldWriter := Patchs, writeSpoutPreviewActive
	Patchs = map[string]*Patch{}
	var displayActive bool
	writeSpoutPreviewActive = func(active bool) error { displayActive = active; return nil }
	t.Cleanup(func() { Patchs, writeSpoutPreviewActive = oldPatches, oldWriter })
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	r := TheResolume()
	oldClient := r.resolumeClient
	r.resolumeClient = osc.NewClient("127.0.0.1", listener.LocalAddr().(*net.UDPAddr).Port)
	t.Cleanup(func() { r.resolumeClient = oldClient })
	for i, name := range []string{"A", "B", "C", "D"} {
		patch := &Patch{name: name}
		Patchs[name] = patch
		for _, bypass := range []bool{true, false} {
			if _, err := patch.API("preview", map[string]string{"onoff": fmt.Sprint(bypass)}); err != nil {
				t.Fatal(err)
			}
			if displayActive != bypass {
				t.Fatalf("Preview display active=%v, want %v", displayActive, bypass)
			}
			if err := listener.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.Fatal(err)
			}
			buf := make([]byte, 1024)
			n, _, err := listener.ReadFromUDP(buf)
			if err != nil {
				t.Fatal(err)
			}
			packet, err := osc.ParsePacket(string(buf[:n]))
			if err != nil {
				t.Fatal(err)
			}
			msg, ok := packet.(*osc.Message)
			if !ok {
				t.Fatalf("expected OSC message, got %T", packet)
			}
			want := int32(0)
			if bypass {
				want = 1
			}
			address := fmt.Sprintf("/composition/layers/%d/bypassed", i+1)
			if msg.Address != address || len(msg.Arguments) != 1 || msg.Arguments[0] != want {
				t.Fatalf("patch %s bypass=%v: got %s %v, want %s [%d]", name, bypass, msg.Address, msg.Arguments, address, want)
			}
			n, _, err = listener.ReadFromUDP(buf)
			if err != nil {
				t.Fatal(err)
			}
			packet, err = osc.ParsePacket(string(buf[:n]))
			if err != nil {
				t.Fatal(err)
			}
			msg, ok = packet.(*osc.Message)
			if !ok {
				t.Fatalf("expected OSC selection message, got %T", packet)
			}
			address = "/composition/select"
			if bypass {
				address = fmt.Sprintf("/composition/layers/%d/select", i+1)
			}
			if msg.Address != address || len(msg.Arguments) != 1 || msg.Arguments[0] != int32(1) {
				t.Fatalf("patch %s preview=%v: got %s %v, want %s [1]", name, bypass, msg.Address, msg.Arguments, address)
			}
		}
	}
}

func TestPatchPreviewRequiresBoolean(t *testing.T) {
	patch := &Patch{name: "A"}
	for _, value := range []string{"", "invalid"} {
		if _, err := patch.API("preview", map[string]string{"onoff": value}); err == nil {
			t.Fatalf("onoff=%q: expected error", value)
		}
	}
}
