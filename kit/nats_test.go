package kit

import "testing"

func TestNatsEnvValueUsesNatsURL(t *testing.T) {
	t.Setenv("NATS_URL", "nats://new.example:4222")
	t.Setenv("NATS_HUB_CLIENT_URL", "nats://old.example:4222")

	got, err := NatsEnvValue("NATS_URL")
	if err != nil {
		t.Fatalf("NatsEnvValue returned error: %v", err)
	}
	if got != "nats://new.example:4222" {
		t.Fatalf("NatsEnvValue = %q, want NATS_URL value", got)
	}
}

func TestNatsEnvValueFallsBackToLegacyHubURL(t *testing.T) {
	t.Setenv("NATS_URL", "")
	t.Setenv("NATS_HUB_CLIENT_URL", "nats://old.example:4222")

	got, err := NatsEnvValue("NATS_URL")
	if err != nil {
		t.Fatalf("NatsEnvValue returned error: %v", err)
	}
	if got != "nats://old.example:4222" {
		t.Fatalf("NatsEnvValue = %q, want legacy NATS_HUB_CLIENT_URL value", got)
	}
}
