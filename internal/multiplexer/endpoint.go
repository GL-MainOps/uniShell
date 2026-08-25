package multiplexer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const defaultTmuxSocketName = "tmux.sock"

func ResolveTmuxSocket(runtimePath, configuredPath string) (string, error) {
	if runtimePath == "" {
		return "", errors.New("runtime path cannot be empty")
	}

	if configuredPath != "" {
		return configuredPath, nil
	}

	return filepath.Join(
		runtimePath,
		"multiplexer",
		defaultTmuxSocketName,
	), nil
}

func PrepareTmuxSocketPath(socketPath string) error {
	parent := filepath.Dir(socketPath)

	if err := os.MkdirAll(parent, 0700); err != nil {
		return fmt.Errorf(
			"create tmux socket directory %q: %w",
			parent,
			err,
		)
	}

	if err := os.Chmod(parent, 0700); err != nil {
		return fmt.Errorf(
			"set tmux socket directory permissions %q: %w",
			parent,
			err,
		)
	}

	return nil
}
