package api

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

	Create(Session) error
	Attach(Session) error
	Detach(Session) error
	IsAlive(Session) bool
	Destroy(Session) error
}
