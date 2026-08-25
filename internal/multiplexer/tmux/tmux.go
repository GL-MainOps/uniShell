package tmux

import (
	"fmt"
	"os"
	"os/exec"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

type CommandRunner func(name string, args ...string) error

type Backend struct {
	Binary     string
	SocketPath string
	Run        CommandRunner
}

func New(socketPath string) *Backend {
	return &Backend{
		Binary:     "tmux",
		SocketPath: socketPath,
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
	args := b.commandArgs("new-session", "-d")

	if session.Name != "" {
		args = append(args, "-s", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Attach(session multiplexer.Session) error {
	args := b.commandArgs("attach-session")

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Detach(session multiplexer.Session) error {
	args := b.commandArgs("detach-client")

	if session.Name != "" {
		args = append(args, "-s", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) IsAlive(session multiplexer.Session) bool {
	args := b.commandArgs("has-session")

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...) == nil
}

func (b *Backend) Destroy(session multiplexer.Session) error {
	args := b.commandArgs("kill-session")

	if session.Name != "" {
		args = append(args, "-t", session.Name)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) commandArgs(args ...string) []string {
	if b.SocketPath == "" {
		return append([]string(nil), args...)
	}

	result := make([]string, 0, len(args)+2)

	result = append(result, "-S", b.SocketPath)
	result = append(result, args...)

	return result
}

func (b *Backend) Validate() error {
	if b.SocketPath == "" {
		return fmt.Errorf("tmux socket path cannot be empty")
	}

	return nil
}
