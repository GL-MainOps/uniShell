package zellij

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/multiplexer/config"
)

type CommandRunner func(
	name string,
	args []string,
	env []string,
) error

type QuietCommandRunner func(
	name string,
	args []string,
	env []string,
) ([]byte, error)

type Backend struct {
	Binary         string
	Run            CommandRunner
	RunQuiet       QuietCommandRunner
	ConfigResolver *config.Resolver
}

func New() *Backend {
	return &Backend{
		Binary:         "zellij",
		ConfigResolver: config.NewResolver(),
		Run: func(
			name string,
			args []string,
			env []string,
		) error {
			cmd := exec.Command(name, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Env = env

			return cmd.Run()
		},
		RunQuiet: func(
			name string,
			args []string,
			env []string,
		) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Env = env

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
	_, err := b.create(session)
	return err
}

func (b *Backend) CreateWithNativeName(
	session api.Session,
) (string, error) {
	return b.create(session)
}

func (b *Backend) create(
	session api.Session,
) (string, error) {
	args := make([]string, 0, 8)

	configResolver := b.ConfigResolver
	if configResolver == nil {
		configResolver = config.NewResolver()
	}

	configPath, err := configResolver.Zellij(
		session.Runtime,
	)
	if err != nil {
		return "", err
	}

	if configPath != "" {
		args = append(
			args,
			"--config",
			configPath,
		)
	}

	args = append(
		args,
		"attach",
		"--create-background",
		"--close-on-exit",
	)

	if err := validateCreateArgs(
		session.Options.Zellij.CreateArgs,
	); err != nil {
		return "", err
	}

	if session.ShellPath == "" {
		return "", fmt.Errorf(
			"zellij shell path cannot be empty",
		)
	}

	args = append(
		args,
		session.Options.Zellij.CreateArgs...,
	)

	if session.NativeName != "" {
		args = append(
			args,
			session.NativeName,
			"--",
			session.ShellPath,
		)

		if err := b.Run(
			b.Binary,
			args,
			session.Env,
		); err != nil {
			return "", err
		}

		return session.NativeName, nil
	}

	nativeName, err := generateNativeName()
	if err != nil {
		return "", fmt.Errorf(
			"generate zellij native session name: %w",
			err,
		)
	}

	args = append(
		args,
		nativeName,
		"--",
		session.ShellPath,
	)

	if err := b.Run(
		b.Binary,
		args,
		session.Env,
	); err != nil {
		return "", err
	}

	return nativeName, nil
}

func generateNativeName() (string, error) {
	const size = 16

	data := make([]byte, size)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return "unishell-" + hex.EncodeToString(data), nil
}

func (b *Backend) Attach(session api.Session) error {
	args := []string{"attach"}

	if session.NativeName != "" {
		args = append(
			args,
			session.NativeName,
		)
	}

	return b.Run(
		b.Binary,
		args,
		nil,
	)
}

func (b *Backend) Detach(session api.Session) error {
	return b.Run(
		b.Binary,
		[]string{
			"action",
			"detach",
		},
		nil,
	)
}

func (b *Backend) IsAlive(session api.Session) bool {
	runner := b.RunQuiet
	if runner == nil {
		runner = func(
			name string,
			args []string,
			env []string,
		) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Env = env

			return cmd.Output()
		}
	}

	output, err := runner(
		b.Binary,
		[]string{"list-sessions", "--short"},
		nil,
	)
	if err != nil {
		return false
	}

	if session.NativeName == "" {
		return len(output) > 0
	}

	for _, line := range strings.Split(
		string(output),
		"\n",
	) {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == session.NativeName {
			return true
		}
	}

	return false
}

func (b *Backend) Destroy(session api.Session) error {
	args := []string{"delete-session"}

	if session.NativeName != "" {
		args = append(
			args,
			session.NativeName,
		)
	}

	return b.Run(
		b.Binary,
		args,
		nil,
	)
}

func validateCreateArgs(args []string) error {
	for index := 0; index < len(args); index++ {
		arg := args[index]

		switch arg {
		case "--session-name",
			"--attach-to-session",
			"--config",
			"--config-dir":
			return fmt.Errorf(
				"zellij create option %q is controlled by uniShell",
				arg,
			)
		}
	}

	return nil
}
