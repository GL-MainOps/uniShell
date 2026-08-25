package zellij

import (
	"os"
	"os/exec"
	"strings"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
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

func (b *Backend) Capabilities() map[api.Capability]bool {
	return map[api.Capability]bool{
		api.CapabilitySessions: true,
		api.CapabilityAttach:   true,
		api.CapabilityDetach:   true,
		api.CapabilityDestroy:  true,
	}
}

func (b *Backend) Available() bool {
	_, err := exec.LookPath(b.Binary)
	return err == nil
}

func (b *Backend) Create(session api.Session) error {
	args := []string{"--session"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}

func (b *Backend) Attach(session api.Session) error {
	args := []string{"attach"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}

func (b *Backend) Detach(session api.Session) error {
	_, err := b.Run(
		b.Binary,
		"action",
		"detach",
	)

	return err
}

func (b *Backend) IsAlive(session api.Session) bool {
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

func (b *Backend) Destroy(session api.Session) error {
	args := []string{"delete-session"}

	if session.Name != "" {
		args = append(args, session.Name)
	}

	_, err := b.Run(b.Binary, args...)
	return err
}
