package multiplexer

import "errors"

var (
	ErrUnavailable = errors.New("multiplexer is unavailable")
	ErrUnsupported = errors.New("multiplexer capability is unsupported")
)

type Capability string

const (
	CapabilitySessions Capability = "sessions"
	CapabilityAttach   Capability = "attach"
	CapabilityDetach   Capability = "detach"
	CapabilityDestroy  Capability = "destroy"
)

type Session struct {
	Name string
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
