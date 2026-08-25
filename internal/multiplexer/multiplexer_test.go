package multiplexer

import "testing"

type testBackend struct {
	name      string
	available bool
}

func (b testBackend) Name() string {
	return b.name
}

func (b testBackend) Capabilities() map[Capability]bool {
	return map[Capability]bool{
		CapabilitySessions: true,
		CapabilityAttach:   true,
	}
}

func (b testBackend) Available() bool {
	return b.available
}

func (b testBackend) Create(Session) error {
	return nil
}

func (b testBackend) Attach(Session) error {
	return nil
}

func (b testBackend) Detach(Session) error {
	return nil
}

func (b testBackend) IsAlive(Session) bool {
	return true
}

func (b testBackend) Destroy(Session) error {
	return nil
}

func TestRegistryReturnsBackend(t *testing.T) {
	backend := testBackend{name: "test"}

	registry := NewRegistry(backend)

	got, ok := registry.Get("test")
	if !ok {
		t.Fatal("Get() did not find backend")
	}

	if got.Name() != "test" {
		t.Fatalf("backend name = %q, want %q", got.Name(), "test")
	}
}

func TestRegistryAvailableReturnsAvailableBackends(t *testing.T) {
	registry := NewRegistry(
		testBackend{name: "available", available: true},
		testBackend{name: "unavailable", available: false},
	)

	backends := registry.Available()

	if len(backends) != 1 {
		t.Fatalf("available backends = %d, want 1", len(backends))
	}

	if backends[0].Name() != "available" {
		t.Fatalf(
			"available backend = %q, want %q",
			backends[0].Name(),
			"available",
		)
	}
}
