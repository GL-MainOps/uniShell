package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const sessionRuntimeDirEnvName = "UNISHELL_SESSION_RUNTIME_DIR"

var ErrShellUnavailable = errors.New("shell is unavailable")

type Command struct {
	Path string
	Args []string
	Env  []string
}

func Resolve() (string, error) {
	if configured := os.Getenv("SHELL"); configured != "" {
		if path, ok := resolveExecutable(configured); ok {
			return path, nil
		}
	}

	for _, candidate := range []string{
		"sh",
		"bash",
		"zsh",
		"fish",
		"ksh",
		"dash",
	} {
		if path, ok := resolveExecutable(candidate); ok {
			return path, nil
		}
	}

	return "", ErrShellUnavailable
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

	shellPath, err := Resolve()
	if err != nil {
		return nil, err
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
		shellPath,
	)

	env = setEnvironment(
		env,
		sessionRuntimeDirEnvName,
		sessionRuntime,
	)

	return env, nil
}

func NewCommand(
	shellPath string,
	runtimeBin string,
	sessionRuntime string,
) (Command, error) {
	if shellPath == "" {
		return Command{}, errors.New(
			"shell path cannot be empty",
		)
	}

	env, err := NewEnvironment(
		runtimeBin,
		sessionRuntime,
	)
	if err != nil {
		return Command{}, err
	}

	env = setEnvironment(
		env,
		"SHELL",
		shellPath,
	)

	return Command{
		Path: shellPath,
		Args: []string{shellPath},
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
