package multiplexer

import (
	"errors"
	"path/filepath"
	"testing"
)

type managerTestBackend struct {
	name      string
	available bool
	alive     bool

	created   bool
	destroyed bool
}

func (b *managerTestBackend) Name() string {
	return b.name
}

func (b *managerTestBackend) Capabilities() map[Capability]bool {
	return map[Capability]bool{
		CapabilitySessions: true,
		CapabilityAttach:   true,
		CapabilityDetach:   true,
		CapabilityDestroy:  true,
	}
}

func (b *managerTestBackend) Available() bool {
	return b.available
}

func (b *managerTestBackend) Create(Session) error {
	b.created = true
	b.alive = true
	return nil
}

func (b *managerTestBackend) Attach(Session) error {
	return nil
}

func (b *managerTestBackend) Detach(Session) error {
	return nil
}

func (b *managerTestBackend) IsAlive(Session) bool {
	return b.alive
}

func (b *managerTestBackend) Destroy(Session) error {
	b.destroyed = true
	b.alive = false
	return nil
}

func TestManagerCreateWritesMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	session, err := manager.Create(
		"test",
		"default",
		runtimePath,
		"/tmp/test.endpoint",
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if !backend.created {
		t.Fatal("backend Create() was not called")
	}

	if session.Metadata.ID == "" {
		t.Fatal("session ID is empty")
	}

	if session.Metadata.Name != "default" {
		t.Fatalf(
			"session name = %q, want %q",
			session.Metadata.Name,
			"default",
		)
	}

	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if metadata.ID != session.Metadata.ID {
		t.Fatalf(
			"metadata ID = %q, want %q",
			metadata.ID,
			session.Metadata.ID,
		)
	}
}

func TestManagerAttachRequiresLiveSession(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	created, err := manager.Create(
		"test",
		"default",
		runtimePath,
		"/tmp/test.endpoint",
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	attached, err := manager.Attach(runtimePath)
	if err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	if attached.Metadata.ID != created.Metadata.ID {
		t.Fatalf(
			"attached ID = %q, want %q",
			attached.Metadata.ID,
			created.Metadata.ID,
		)
	}

	backend.alive = false

	_, err = manager.Attach(runtimePath)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Attach() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDestroyRemovesMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		runtimePath,
		"/tmp/test.endpoint",
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := manager.Destroy(runtimePath); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	if !backend.destroyed {
		t.Fatal("backend Destroy() was not called")
	}

	if _, err := ReadMetadata(runtimePath); err == nil {
		t.Fatal("ReadMetadata() succeeded after Destroy()")
	}
}
