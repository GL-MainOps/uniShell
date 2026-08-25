package tmux

import (
	"os"
	"os/exec"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

type CommandRunner func(name string, args ...string) error

type Backend struct {
	Binary string
	Run    CommandRunner
}

func New() *Backend {
	return &Backend{
		Binary: "tmux",
		Run: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

func (b *Backend) Name() string {
	return "tmux"
}

func (b *Backend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *Backend) Available() bool {
	_, err := exec.LookPath(b.Binary)
	return err == nil
}

func (b *Backend) Create(session multiplexer.Session) error {
	args := []string{"new-session", "-d"}

	if session.Name != "" {
		args = append(args, "-s", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Attach(session multiplexer.Session) error {
	args := []string{"attach-session"}

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Detach(session multiplexer.Session) error {
	args := []string{"detach-client"}

	if session.Name != "" {
		args = append(args, "-s", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) IsAlive(session multiplexer.Session) bool {
	args := []string{"has-session"}

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...) == nil
}

func (b *Backend) Destroy(session multiplexer.Session) error {
	args := []string{"kill-session"}

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...)
}
