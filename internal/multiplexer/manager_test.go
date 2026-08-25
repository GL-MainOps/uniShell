package multiplexer

import (
	"errors"
	"os"
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

func TestManagerDiscoverFindsLiveSession(t *testing.T) {
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

	discovered, err := manager.Discover(
		runtimePath,
		"default",
	)
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	if discovered.Metadata.ID != created.Metadata.ID {
		t.Fatalf(
			"discovered ID = %q, want %q",
			discovered.Metadata.ID,
			created.Metadata.ID,
		)
	}

	if discovered.Metadata.Name != "default" {
		t.Fatalf(
			"discovered name = %q, want %q",
			discovered.Metadata.Name,
			"default",
		)
	}
}

func TestManagerDiscoverRejectsDifferentSessionName(t *testing.T) {
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

	_, err := manager.Discover(
		runtimePath,
		"work",
	)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDiscoverRejectsStaleMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := Metadata{
		ID:          "stale-session",
		Name:        "default",
		Multiplexer: "test",
		Endpoint:    "/tmp/test.endpoint",
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("WriteMetadata() returned error: %v", err)
	}

	_, err := manager.Discover(
		runtimePath,
		"default",
	)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDiscoverRejectsUnavailableBackend(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: false,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := Metadata{
		ID:          "session",
		Name:        "default",
		Multiplexer: "test",
		Endpoint:    "/tmp/test.endpoint",
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("WriteMetadata() returned error: %v", err)
	}

	_, err := manager.Discover(
		runtimePath,
		"default",
	)

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrUnavailable,
		)
	}
}

func TestManagerDiscoverByNameFindsSessionAcrossRuntimeDirectories(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	firstRuntime := filepath.Join(versionRuntime, "first")
	secondRuntime := filepath.Join(versionRuntime, "second")

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
		"other",
		firstRuntime,
		"/tmp/first.endpoint",
	); err != nil {
		t.Fatalf("Create(first) returned error: %v", err)
	}

	if _, err := manager.Create(
		"test",
		"default",
		secondRuntime,
		"/tmp/second.endpoint",
	); err != nil {
		t.Fatalf("Create(second) returned error: %v", err)
	}

	session, err := manager.DiscoverByName(
		versionRuntime,
		"default",
	)
	if err != nil {
		t.Fatalf(
			"DiscoverByName() returned error: %v",
			err,
		)
	}

	if session.Session.Runtime != secondRuntime {
		t.Fatalf(
			"runtime = %q, want %q",
			session.Session.Runtime,
			secondRuntime,
		)
	}
}

func TestManagerReconcilePreservesLiveSession(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
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
		sessionRuntime,
		"/tmp/test.endpoint",
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sessionRuntime, "payload"),
		[]byte("runtime"),
		0600,
	); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); err != nil {
		t.Fatalf(
			"live session was removed: %v",
			err,
		)
	}
}

func TestManagerReconcileRemovesDeadSession(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		sessionRuntime,
		"/tmp/test.endpoint",
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Create() marks the fake backend alive. Make it stale explicitly.
	backend.alive = false

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(
			"dead session runtime still exists, stat error = %v",
			err,
		)
	}
}

func TestManagerReconcileIgnoresUnavailableBackend(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
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
		sessionRuntime,
		"/tmp/test.endpoint",
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Simulate the backend becoming unavailable after the session exists.
	backend.available = false
	backend.alive = false

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); err != nil {
		t.Fatalf(
			"session was removed while backend was unavailable: %v",
			err,
		)
	}
}
