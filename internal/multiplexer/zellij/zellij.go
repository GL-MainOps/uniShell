package zellij

import (
	"os"
	"os/exec"
	"strings"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

type CommandRunner func(name string, args ...string) ([]byte, error)

type Backend struct {
	Binary string
	Run    CommandRunner
}

func New() *Backend {
	return &Backend{
		Binary: "zellij",
		Run: func(name string, args ...string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd.Output()
		},
	}
}

func (b *Backend) Name() string {
	return "zellij"
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
	args := []string{"--session"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}

func (b *Backend) Attach(session multiplexer.Session) error {
	args := []string{"attach"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}

func (b *Backend) Detach(session multiplexer.Session) error {
	_, err := b.Run(
		b.Binary,
		"action",
		"detach",
	)

	return err
}

func (b *Backend) IsAlive(session multiplexer.Session) bool {
	output, err := b.Run(
		b.Binary,
		"list-sessions",
	)
	if err != nil {
		return false
	}

	if session.Name == "" {
		return len(output) > 0
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == session.Name {
			return true
		}
	}

	return false
}

func (b *Backend) Destroy(session multiplexer.Session) error {
	args := []string{"delete-session"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}
