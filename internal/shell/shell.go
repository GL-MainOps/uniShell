package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	SessionRuntimeDirEnvName = "UNISHELL_SESSION_RUNTIME_DIR"
	ShellEnvName             = "UNISHELL_SHELL"
)

var (
	ErrShellUnavailable = errors.New("shell is unavailable")
	ErrShellUnsupported = errors.New("shell is unsupported")
)

type Shell struct {
	Name   string
	Path   string
	Source Source
}

type Source string

const (
	SourceBundled Source = "bundled"
	SourceHost    Source = "host"
)

type Command struct {
	Path string
	Args []string
	Env  []string
}

type Handoff struct {
	Path string
	Args []string
}

var supportedShells = []string{
	"bash",
	"zsh",
	"fish",
	"nushell",
}

func SupportedShells() []string {
	return append([]string(nil), supportedShells...)
}

// Resolve selects the shell according to:
//
//  1. explicit shell name
//  2. UNISHELL_SHELL
//  3. $SHELL basename
//  4. bash
//
// Shell executable resolution is separate:
//
//   - bash: host PATH
//   - zsh/fish/nushell: bundled runtime binary, then host PATH
func Resolve(explicit, runtimeBin string) (Shell, error) {
	requested, explicitRequest := requestedShell(explicit)

	if requested == "" {
		requested = "bash"
	}

	if !isSupported(requested) {
		if explicitRequest {
			return Shell{}, fmt.Errorf(
				"%w: %q",
				ErrShellUnsupported,
				requested,
			)
		}

		return Shell{}, fmt.Errorf(
			"%w: %q",
			ErrShellUnsupported,
			requested,
		)
	}

	path, source, ok := resolveShell(requested, runtimeBin)
	if !ok {
		return Shell{}, fmt.Errorf(
			"%w: %q",
			ErrShellUnavailable,
			requested,
		)
	}

	return Shell{
		Name:   requested,
		Path:   path,
		Source: source,
	}, nil
}

func ResolveFromEnvironment(runtimeBin string) (Shell, error) {
	return Resolve("", runtimeBin)
}

func NewEnvironment(
	runtimeBin string,
	sessionRuntime string,
) ([]string, error) {
	if runtimeBin == "" {
		return nil, errors.New(
			"runtime bin path cannot be empty",
		)
	}

	if sessionRuntime == "" {
		return nil, errors.New(
			"session runtime path cannot be empty",
		)
	}

	selected, err := ResolveFromEnvironment(runtimeBin)
	if err != nil {
		return nil, err
	}

	return NewEnvironmentForShell(
		runtimeBin,
		sessionRuntime,
		selected,
	)
}

func NewEnvironmentForShell(
	runtimeBin string,
	sessionRuntime string,
	selected Shell,
) ([]string, error) {
	if runtimeBin == "" {
		return nil, errors.New(
			"runtime bin path cannot be empty",
		)
	}

	if sessionRuntime == "" {
		return nil, errors.New(
			"session runtime path cannot be empty",
		)
	}

	if selected.Name == "" || selected.Path == "" {
		return nil, errors.New(
			"selected shell is incomplete",
		)
	}

	path := buildPATH(
		runtimeBin,
		os.Getenv("PATH"),
	)

	env := os.Environ()

	env = setEnvironment(
		env,
		"PATH",
		path,
	)

	env = setEnvironment(
		env,
		"SHELL",
		selected.Path,
	)

	env = setEnvironment(
		env,
		SessionRuntimeDirEnvName,
		sessionRuntime,
	)

	return env, nil
}

func NewCommand(
	selected Shell,
	runtimeBin string,
	sessionRuntime string,
	startup Startup,
	handoff *Handoff,
) (Command, error) {
	if selected.Path == "" {
		return Command{}, errors.New(
			"shell path cannot be empty",
		)
	}

	env, err := NewEnvironmentForShell(
		runtimeBin,
		sessionRuntime,
		selected,
	)
	if err != nil {
		return Command{}, err
	}

	for key, value := range startup.Env {
		env = setEnvironment(env, key, value)
	}

	args := []string{selected.Path}
	args = append(args, startup.Args...)

	if handoff != nil {
		if handoff.Path == "" {
			return Command{}, errors.New(
				"handoff path cannot be empty",
			)
		}

		args = append(
			args,
			handoff.Path,
		)
		args = append(
			args,
			handoff.Args...,
		)
	}

	return Command{
		Path: selected.Path,
		Args: args,
		Env:  env,
	}, nil
}

func (c Command) Run() error {
	cmd := exec.Command(c.Path, c.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = c.Env

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run shell: %w", err)
	}

	return nil
}

func requestedShell(explicit string) (string, bool) {
	if explicit != "" {
		return normalizeShellName(explicit), true
	}

	if configured := strings.TrimSpace(
		os.Getenv(ShellEnvName),
	); configured != "" {
		return normalizeShellName(configured), true
	}

	if configured := strings.TrimSpace(
		os.Getenv("SHELL"),
	); configured != "" {
		return normalizeShellName(configured), false
	}

	return "bash", false
}

func normalizeShellName(value string) string {
	return strings.ToLower(
		filepath.Base(
			strings.TrimSpace(value),
		),
	)
}

func isSupported(name string) bool {
	for _, candidate := range supportedShells {
		if candidate == name {
			return true
		}
	}

	return false
}

func resolveShell(
	name string,
	runtimeBin string,
) (string, Source, bool) {
	if name != "bash" && runtimeBin != "" {
		bundled := filepath.Join(runtimeBin, name)

		if isExecutable(bundled) {
			return bundled, SourceBundled, true
		}
	}

	host, err := exec.LookPath(name)
	if err != nil {
		return "", "", false
	}

	return host, SourceHost, true
}

func buildPATH(runtimeBin, existing string) string {
	if existing == "" {
		return runtimeBin
	}

	return runtimeBin +
		string(os.PathListSeparator) +
		existing
}

func setEnvironment(
	env []string,
	key string,
	value string,
) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	found := false

	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if !found {
				result = append(
					result,
					prefix+value,
				)
				found = true
			}

			continue
		}

		result = append(result, entry)
	}

	if !found {
		result = append(result, prefix+value)
	}

	return result
}

func resolveExecutable(path string) (string, bool) {
	if strings.ContainsRune(path, os.PathSeparator) {
		if !isExecutable(path) {
			return "", false
		}

		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", false
		}

		return absolute, true
	}

	resolved, err := exec.LookPath(path)
	if err != nil {
		return "", false
	}

	return resolved, true
}

func isExecutable(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	return info.Mode().Perm()&0111 != 0
}
