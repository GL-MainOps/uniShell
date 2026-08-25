package main

import (
	"errors"
	"testing"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

type shellHandoffBackend struct {
	attached  bool
	created   bool
	destroyed bool
}

func (b *shellHandoffBackend) Name() string {
	return "test"
}

func (b *shellHandoffBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *shellHandoffBackend) Available() bool {
	return true
}

func (b *shellHandoffBackend) Create(multiplexer.Session) error {
	b.created = true
	return nil
}

func (b *shellHandoffBackend) Attach(multiplexer.Session) error {
	b.attached = true
	return nil
}

func (b *shellHandoffBackend) Detach(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackend) IsAlive(multiplexer.Session) bool {
	return true
}

func (b *shellHandoffBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return nil
}

func TestRunShellUsesMultiplexerAsInteractiveEntryPoint(
	t *testing.T,
) {
	backend := &shellHandoffBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"runShell() returned error: %v",
			err,
		)
	}

	if !backend.attached {
		t.Fatal(
			"runShell() did not enter the existing multiplexer session",
		)
	}

	if backend.created {
		t.Fatal(
			"runShell() created a second multiplexer session",
		)
	}

	if backend.destroyed {
		t.Fatal(
			"runShell() destroyed the existing multiplexer session",
		)
	}
}

func TestRunShellDoesNotFallbackToStandaloneShell(
	t *testing.T,
) {
	backend := &shellHandoffBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	err := runShell(application, []string{"unexpected"})
	if err == nil {
		t.Fatal("runShell() accepted an unexpected shell argument")
	}

	if backend.attached {
		t.Fatal(
			"runShell() attached a session after rejecting the command",
		)
	}
}

func TestRunShellPropagatesMultiplexerAttachFailure(
	t *testing.T,
) {
	attachErr := errors.New("multiplexer attach failed")

	backend := &shellHandoffBackendWithError{
		attachErr: attachErr,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	err := runShell(application, nil)

	if !errors.Is(err, attachErr) {
		t.Fatalf(
			"runShell() error = %v, want %v",
			err,
			attachErr,
		)
	}
}

type shellHandoffBackendWithError struct {
	attachErr error
}

func (b *shellHandoffBackendWithError) Name() string {
	return "test"
}

func (b *shellHandoffBackendWithError) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *shellHandoffBackendWithError) Available() bool {
	return true
}

func (b *shellHandoffBackendWithError) Create(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackendWithError) Attach(multiplexer.Session) error {
	return b.attachErr
}

func (b *shellHandoffBackendWithError) Detach(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackendWithError) IsAlive(multiplexer.Session) bool {
	return true
}

func (b *shellHandoffBackendWithError) Destroy(multiplexer.Session) error {
	return nil
}
