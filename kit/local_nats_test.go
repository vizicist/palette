package kit

import "testing"

func TestEmbeddedNATSLeafURLUsesExplicitLeafURL(t *testing.T) {
	t.Setenv("NATS_LEAF_URL", "tls://leaf.example:7422")
	t.Setenv("NATS_URL", "tls://client.example:4222")

	got, err := EmbeddedNATSLeafURL()
	if err != nil {
		t.Fatalf("EmbeddedNATSLeafURL returned error: %v", err)
	}
	if got != "tls://leaf.example:7422" {
		t.Fatalf("EmbeddedNATSLeafURL = %q, want explicit leaf URL", got)
	}
}

func TestEmbeddedNATSLeafURLDerivesLeafPortFromNATSURL(t *testing.T) {
	t.Setenv("NATS_LEAF_URL", "")
	t.Setenv("NATS_URL", "tls://user:pass@photonsalon.com:4222")

	got, err := EmbeddedNATSLeafURL()
	if err != nil {
		t.Fatalf("EmbeddedNATSLeafURL returned error: %v", err)
	}
	if got != "tls://user:pass@photonsalon.com:7422" {
		t.Fatalf("EmbeddedNATSLeafURL = %q, want derived leaf URL", got)
	}
}
