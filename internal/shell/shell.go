package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrShellUnavailable = errors.New("shell is unavailable")

type Command struct {
	Path string
	Args []string
	Env  []string
}

func Resolve() (string, error) {
	if shell := os.Getenv("SHELL"); shell != "" {
		if isExecutable(shell) {
			return shell, nil
		}
	}

	for _, candidate := range []string{
		"/bin/sh",
		"/bin/bash",
		"/bin/zsh",
	} {
		if isExecutable(candidate) {
			return candidate, nil
		}
	}

	return "", ErrShellUnavailable
}

func NewEnvironment(
	runtimeBin string,
) ([]string, error) {
	if runtimeBin == "" {
		return nil, errors.New(
			"runtime bin path cannot be empty",
		)
	}

	path := buildPATH(
		runtimeBin,
		os.Getenv("PATH"),
	)

	env := os.Environ()

	return setEnvironment(
		env,
		"PATH",
		path,
	), nil
}

func NewCommand(
	shellPath string,
	runtimeBin string,
) (Command, error) {
	if shellPath == "" {
		return Command{}, errors.New(
			"shell path cannot be empty",
		)
	}

	env, err := NewEnvironment(runtimeBin)
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
		result = append(
			result,
			prefix+value,
		)
	}

	return result
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
