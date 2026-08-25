package multiplexer

import (
	"errors"
	"fmt"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

var (
	ErrUnavailable     = errors.New("multiplexer is unavailable")
	ErrUnsupported     = errors.New("multiplexer capability is unsupported")
	ErrSessionNotFound = errors.New("multiplexer session not found")
)

type Capability = api.Capability

const (
	CapabilitySessions = api.CapabilitySessions
	CapabilityAttach   = api.CapabilityAttach
	CapabilityDetach   = api.CapabilityDetach
	CapabilityDestroy  = api.CapabilityDestroy
)

type Session = api.Session
type Backend = api.Backend

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
