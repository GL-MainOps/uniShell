package tmux

import (
	"fmt"
	"os"
	"os/exec"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/multiplexer/config"
)

type CommandRunner func(name string, args ...string) error

type Backend struct {
	Binary         string
	Run            CommandRunner
	RunQuiet       CommandRunner
	ConfigResolver *config.Resolver
}

func New() *Backend {
	return &Backend{
		Binary:         "tmux",
		ConfigResolver: config.NewResolver(),
		Run: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			return cmd.Run()
		},
		RunQuiet: func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			return cmd.Run()
		},
	}
}

func (b *Backend) Name() string {
	return "tmux"
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
	args, err := b.commandArgs(
		session,
		"new-session",
		"-d",
	)
	if err != nil {
		return err
	}

	for _, entry := range session.Env {
		args = append(args, "-e", entry)
	}

	if err := validateCreateArgs(
		session.Options.Tmux.CreateArgs,
	); err != nil {
		return err
	}

	args = append(
		args,
		session.Options.Tmux.CreateArgs...,
	)

	if session.NativeName != "" {
		args = append(
			args,
			"-s",
			session.NativeName,
		)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Attach(session api.Session) error {
	args, err := b.commandArgs(
		session,
		"attach-session",
	)
	if err != nil {
		return err
	}

	if session.NativeName != "" {
		args = append(
			args,
			"-t",
			session.NativeName,
		)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) Detach(session api.Session) error {
	args, err := b.commandArgs(
		session,
		"detach-client",
	)
	if err != nil {
		return err
	}

	if session.NativeName != "" {
		args = append(
			args,
			"-s",
			session.NativeName,
		)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) IsAlive(session api.Session) bool {
	args, err := b.commandArgs(
		session,
		"has-session",
	)
	if err != nil {
		return false
	}

	if session.NativeName != "" {
		args = append(
			args,
			"-t",
			session.NativeName,
		)
	}

	runner := b.RunQuiet
	if runner == nil {
		runner = func(name string, args ...string) error {
			cmd := exec.Command(name, args...)
			return cmd.Run()
		}
	}

	return runner(b.Binary, args...) == nil
}

func (b *Backend) Destroy(session api.Session) error {
	args, err := b.commandArgs(
		session,
		"kill-session",
	)
	if err != nil {
		return err
	}

	if session.NativeName != "" {
		args = append(
			args,
			"-t",
			session.NativeName,
		)
	}

	return b.Run(b.Binary, args...)
}

func (b *Backend) commandArgs(
	session api.Session,
	args ...string,
) ([]string, error) {
	if session.Endpoint == "" {
		return nil, fmt.Errorf(
			"tmux endpoint cannot be empty",
		)
	}

	result := make(
		[]string,
		0,
		len(args)+4,
	)

	configResolver := b.ConfigResolver
	if configResolver == nil {
		configResolver = config.NewResolver()
	}

	config, err := configResolver.Tmux(
		session.Runtime,
	)
	if err != nil {
		return nil, err
	}

	if config != "" {
		result = append(
			result,
			"-f",
			config,
		)
	}

	result = append(
		result,
		"-S",
		session.Endpoint,
	)

	result = append(result, args...)

	return result, nil
}

func validateCreateArgs(args []string) error {
	for index := 0; index < len(args); index++ {
		arg := args[index]

		switch arg {
		case "-S", "-L", "-c", "-D", "-N":
			return fmt.Errorf(
				"tmux create option %q is controlled by uniShell",
				arg,
			)
		}
	}

	return nil
}
