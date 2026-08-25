package app

import (
	"errors"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

type sessionTestBackend struct {
	destroyErr error
	destroyed  bool
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
	return nil
}

func (b *sessionTestBackend) Detach(multiplexer.Session) error {
	return nil
}

func (b *sessionTestBackend) IsAlive(multiplexer.Session) bool {
	return true
}

func (b *sessionTestBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return b.destroyErr
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
