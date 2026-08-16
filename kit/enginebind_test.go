package kit

import "testing"

// The engine's HTTP server must not be on the network by default. The /api
// handler calls ExecuteAPIFromJSON with no token, session, or origin check, so
// anything that can reach the port drives the whole engine: read
// global.emailpassword, point global.resolumepath at an executable and start
// it, or forward a command to another Palette through /nats/api.
func TestEngineHTTPBindsLoopbackByDefault(t *testing.T) {
	t.Setenv(PaletteHTTPBindEnv, "")
	if got := engineHTTPBindAddress(); got != LocalAddress {
		t.Fatalf("engine HTTP binds %q by default, want %q", got, LocalAddress)
	}
}

// An installation that wants the GUI reachable from the venue LAN can still opt
// in, deliberately, from outside the API.
func TestEngineHTTPBindHonorsEnvOptIn(t *testing.T) {
	t.Setenv(PaletteHTTPBindEnv, "0.0.0.0")
	if got := engineHTTPBindAddress(); got != "0.0.0.0" {
		t.Fatalf("with %s=0.0.0.0 the bind address is %q", PaletteHTTPBindEnv, got)
	}
	t.Setenv(PaletteHTTPBindEnv, "192.168.1.50")
	if got := engineHTTPBindAddress(); got != "192.168.1.50" {
		t.Fatalf("a specific interface address was not honored, got %q", got)
	}
}
