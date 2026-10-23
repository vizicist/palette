package kit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
)

// The env file holds the hub NATS password and the OBS and YouTube
// credentials, so it must be readable by its owner only, and an existing
// world-readable one must be tightened the next time it is saved.
func TestWriteEnvFileIsOwnerOnlyAndRoundTrips(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits")
	}
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("NATS_URL=nats://old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"NATS_URL":     "nats://user:p@ss word@hub.example:4222",
		"OBS_PASSWORD": "a=b#c",
	}
	if err := WriteEnvFile(path, want); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("env file mode is %o, want 0600", perm)
	}

	got, err := godotenv.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q after round trip, want %q", k, got[k], v)
		}
	}
	if _, stale := got["NATS_URL_OLD"]; stale || got["NATS_URL"] == "nats://old" {
		t.Error("old contents survived the rewrite")
	}
}
