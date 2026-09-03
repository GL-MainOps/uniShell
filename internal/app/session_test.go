package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
	runtimepkg "gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

type sessionTestBackend struct {
	destroyErr error
	detachErr  error
	destroyed  bool
	attached   bool
	detached   bool
	alive      bool
}

func (b *sessionTestBackend) Name() string {
	return "test"
}

func (b *sessionTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *sessionTestBackend) Available() bool {
	return true
}

func (b *sessionTestBackend) Create(multiplexer.Session) error {
	return nil
}

func (b *sessionTestBackend) Attach(multiplexer.Session) error {
	b.attached = true
	return nil
}

func (b *sessionTestBackend) Detach(multiplexer.Session) error {
	b.detached = true
	return b.detachErr
}

func (b *sessionTestBackend) IsAlive(multiplexer.Session) bool {
	return b.alive
}

func (b *sessionTestBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return b.destroyErr
}

func newManagedTestSession(
	t *testing.T,
	backend *sessionTestBackend,
) (*Session, string) {
	t.Helper()

	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

	if err := multiplexer.WriteMetadata(
		runtimePath,
		multiplexer.Metadata{
			ID:                "test-session",
			PID:               os.Getpid(),
			ProcessStartTicks: runtimepkg.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeMultiplexer,
			Name:              "default",
			Multiplexer:       "test",
			Endpoint: filepath.Join(
				runtimePath,
				"multiplexer",
				"test.sock",
			),
		},
	); err != nil {
		t.Fatalf(
			"WriteMetadata() returned error: %v",
			err,
		)
	}

	return &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Metadata: multiplexer.Metadata{
				ID:          "test-session",
				Name:        "default",
				Multiplexer: "test",
				Endpoint: filepath.Join(
					runtimePath,
					"multiplexer",
					"test.sock",
				),
			},
			Backend: backend,
			Session: multiplexer.Session{
				Name: "default",
				Endpoint: filepath.Join(
					runtimePath,
					"multiplexer",
					"test.sock",
				),
				Runtime: runtimePath,
			},
		},
	}, runtimePath
}

func TestSessionCleanupDestroysMultiplexer(t *testing.T) {
	backend := &sessionTestBackend{}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/tmp/test.sock",
			},
		},
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	if !backend.destroyed {
		t.Fatal("Cleanup() did not destroy multiplexer session")
	}
}

func TestSessionCleanupReturnsMultiplexerError(t *testing.T) {
	wantErr := errors.New("destroy failed")

	backend := &sessionTestBackend{
		destroyErr: wantErr,
	}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
		},
	}

	err := session.Cleanup()

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"Cleanup() error = %v, want %v",
			err,
			wantErr,
		)
	}
}

func TestSessionCleanupContinuesRuntimeCleanupAfterMultiplexerFailure(
	t *testing.T,
) {
	backend := &sessionTestBackend{
		destroyErr: errors.New("destroy failed"),
	}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
		},
	}

	err := session.Cleanup()

	if err == nil {
		t.Fatal("Cleanup() returned nil error")
	}

	if !backend.destroyed {
		t.Fatal("Cleanup() did not attempt multiplexer destruction")
	}
}

func TestSessionAttachCallsMultiplexer(t *testing.T) {
	backend := &sessionTestBackend{
		alive: true,
	}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:       "default",
				NativeName: "native-default",
				Endpoint:   "/tmp/test.sock",
			},
		},
	}

	if err := session.Attach(); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	if !backend.attached {
		t.Fatal("Attach() did not call multiplexer backend")
	}
}

func TestSessionAttachPreservesLiveRuntime(t *testing.T) {
	backend := &sessionTestBackend{
		alive: true,
	}

	session, runtimePath := newManagedTestSession(
		t,
		backend,
	)

	if err := session.Attach(); err != nil {
		t.Fatalf(
			"Attach() returned error: %v",
			err,
		)
	}

	if !backend.attached {
		t.Fatal("Attach() did not call multiplexer backend")
	}

	if backend.destroyed {
		t.Fatal(
			"Attach() destroyed a live multiplexer session",
		)
	}

	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf(
			"live multiplexer runtime was removed: %v",
			err,
		)
	}
}

func TestSessionAttachCleansExitedRuntime(t *testing.T) {
	backend := &sessionTestBackend{
		alive: false,
	}

	session, runtimePath := newManagedTestSession(
		t,
		backend,
	)

	if err := session.Attach(); err != nil {
		t.Fatalf(
			"Attach() returned error: %v",
			err,
		)
	}

	if !backend.attached {
		t.Fatal("Attach() did not call multiplexer backend")
	}

	if backend.destroyed {
		t.Fatal(
			"Attach() destroyed an already exited multiplexer session",
		)
	}

	if _, err := os.Stat(runtimePath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"exited multiplexer runtime still exists, stat error = %v",
			err,
		)
	}
}

func TestSessionDetachCallsMultiplexer(t *testing.T) {
	backend := &sessionTestBackend{
		alive: true,
	}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:       "default",
				NativeName: "native-default",
				Endpoint:   "/tmp/test.sock",
			},
		},
	}

	if err := session.Detach(); err != nil {
		t.Fatalf("Detach() returned error: %v", err)
	}

	if !backend.detached {
		t.Fatal("Detach() did not call multiplexer backend")
	}

	if backend.destroyed {
		t.Fatal("Detach() destroyed multiplexer session")
	}
}

func TestSessionDetachReturnsMultiplexerError(t *testing.T) {
	wantErr := errors.New("detach failed")

	backend := &sessionTestBackend{
		detachErr: wantErr,
		alive:     true,
	}

	session := &Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
		},
	}

	err := session.Detach()

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"Detach() error = %v, want %v",
			err,
			wantErr,
		)
	}
}
