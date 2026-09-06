package multiplexer

import (
	"errors"
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
