package api

import sessionmeta "gitlab.com/mainops/uniShell/internal/session"

type Capability string

const (
	CapabilitySessions Capability = "sessions"
	CapabilityAttach   Capability = "attach"
	CapabilityDetach   Capability = "detach"
	CapabilityDestroy  Capability = "destroy"
)

type Session struct {
	Name       string
	NativeName string
	Runtime    string
	Endpoint   string
	ShellName  string
	ShellPath  string
	ShellArgs  []string
	Env        []string
	Options    Options
}

type Backend interface {
	Name() string
	Capabilities() map[Capability]bool
	Available() bool
	AvailableForSession(Session) bool

	Create(Session) error
	Attach(Session) error
	Detach(Session) error
	IsAlive(Session) bool
	Destroy(Session) error
}

// NativeNameCreator is optionally implemented by backends that can
// determine the native session name when creation does not receive one.
type NativeNameCreator interface {
	CreateWithNativeName(Session) (string, error)
}

// ProcessIdentityProvider is optionally implemented by backends that can
// identify the native multiplexer process that owns a managed session.
type ProcessIdentityProvider interface {
	ProcessIdentity(Session) (sessionmeta.ProcessIdentity, error)
}
