package main

import (
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

func TestRunShellCreatesThenReattachesExistingMultiplexerSession(
	t *testing.T,
) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	backend := &lifecycleTestBackend{}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := app.New(app.Options{
		Version:         "1.0.0",
		Commit:          "test",
		Root:            root,
		Bundle:          lifecycleTestBundleSource(t),
		Multiplexer:     manager,
		MultiplexerName: "test",
		SessionName:     "default",
	})
	if err != nil {
		t.Fatalf("app.New() returned error: %v", err)
	}

	// First invocation must create and attach a new session.
	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"first runShell() returned error: %v",
			err,
		)
	}

	if backend.createCount != 1 {
		t.Fatalf(
			"backend Create() count = %d, want 1",
			backend.createCount,
		)
	}

	if backend.attachCount != 1 {
		t.Fatalf(
			"backend Attach() count after first invocation = %d, want 1",
			backend.attachCount,
		)
	}

	if backend.destroyCount != 0 {
		t.Fatalf(
			"backend Destroy() count after first invocation = %d, want 0",
			backend.destroyCount,
		)
	}

	// The runtime must remain available after a successful shell start.
	entries, err := os.ReadDir(application.Paths.Runtime)
	if err != nil {
		t.Fatalf(
			"read version runtime after first invocation: %v",
			err,
		)
	}

	if len(entries) != 1 {
		t.Fatalf(
			"version runtime entry count = %d, want 1",
			len(entries),
		)
	}

	// Second invocation must discover the existing session instead
	// of creating another one.
	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"second runShell() returned error: %v",
			err,
		)
	}

	if backend.createCount != 1 {
		t.Fatalf(
			"backend Create() count after second invocation = %d, want 1",
			backend.createCount,
		)
	}

	if backend.attachCount != 2 {
		t.Fatalf(
			"backend Attach() count after second invocation = %d, want 2",
			backend.attachCount,
		)
	}

	if backend.destroyCount != 0 {
		t.Fatalf(
			"backend Destroy() count after second invocation = %d, want 0",
			backend.destroyCount,
		)
	}

	// Explicit cleanup must destroy the multiplexer and remove
	// the runtime session.
	session, err := application.DiscoverMultiplexerSession()
	if err != nil {
		t.Fatalf(
			"DiscoverMultiplexerSession() returned error: %v",
			err,
		)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf(
			"session.Cleanup() returned error: %v",
			err,
		)
	}

	if backend.destroyCount != 1 {
		t.Fatalf(
			"backend Destroy() count after cleanup = %d, want 1",
			backend.destroyCount,
		)
	}

	entries, err = os.ReadDir(application.Paths.Runtime)
	if err == nil && len(entries) != 0 {
		t.Fatalf(
			"version runtime still contains %d entries after cleanup",
			len(entries),
		)
	}

	if err != nil && !os.IsNotExist(err) {
		t.Fatalf(
			"read version runtime after cleanup: %v",
			err,
		)
	}
}

type lifecycleTestBackend struct {
	createCount  int
	attachCount  int
	destroyCount int

	alive bool
}

func (b *lifecycleTestBackend) Name() string {
	return "test"
}

func (b *lifecycleTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *lifecycleTestBackend) Available() bool {
	return true
}

func (b *lifecycleTestBackend) Create(
	session multiplexer.Session,
) error {
	b.createCount++
	b.alive = true

	return nil
}

func (b *lifecycleTestBackend) Attach(
	session multiplexer.Session,
) error {
	b.attachCount++

	return nil
}

func (b *lifecycleTestBackend) Detach(
	session multiplexer.Session,
) error {
	return nil
}

func (b *lifecycleTestBackend) IsAlive(
	session multiplexer.Session,
) bool {
	return b.alive
}

func (b *lifecycleTestBackend) Destroy(
	session multiplexer.Session,
) error {
	b.destroyCount++
	b.alive = false

	return nil
}

func lifecycleTestBundleSource(t *testing.T) app.BundleSource {
	t.Helper()

	return func() ([]byte, error) {
		return os.ReadFile(
			filepath.Join(
				"..",
				"..",
				"internal",
				"bundle",
				"testdata",
				"runtime.bundle",
			),
		)
	}
}
