package multiplexer

import (
	"errors"
	"fmt"
)

var (
	ErrUnavailable     = errors.New("multiplexer is unavailable")
	ErrUnsupported     = errors.New("multiplexer capability is unsupported")
	ErrSessionNotFound = errors.New("multiplexer session not found")
)

type Capability string

const (
	CapabilitySessions Capability = "sessions"
	CapabilityAttach   Capability = "attach"
	CapabilityDetach   Capability = "detach"
	CapabilityDestroy  Capability = "destroy"
)

type SessionState string

const (
	StateUnknown SessionState = "unknown"
	StateAlive   SessionState = "alive"
	StateStopped SessionState = "stopped"
)

type Session struct {
	Name     string
	Runtime  string
	Endpoint string
}

type Backend interface {
	Name() string
	Capabilities() map[Capability]bool
	Available() bool

	Create(Session) error
	Attach(Session) error
	Detach(Session) error
	IsAlive(Session) bool
	Destroy(Session) error
}

func RequireCapability(
	backend Backend,
	capability Capability,
) error {
	if !backend.Capabilities()[capability] {
		return fmt.Errorf(
			"%s: %w: %s",
			backend.Name(),
			ErrUnsupported,
			capability,
		)
	}

	return nil
}

type Registry struct {
	backends map[string]Backend
}

func NewRegistry(backends ...Backend) *Registry {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	for _, backend := range backends {
		registry.backends[backend.Name()] = backend
	}

	return registry
}

func (r *Registry) Get(name string) (Backend, bool) {
	backend, ok := r.backends[name]
	return backend, ok
}

func (r *Registry) Available() []Backend {
	result := make([]Backend, 0, len(r.backends))

	for _, backend := range r.backends {
		if backend.Available() {
			result = append(result, backend)
		}
	}

	return result
}
