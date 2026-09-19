package tmux

import (
	"errors"
	"path/filepath"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

const defaultSocketName = "tmux.sock"

func ResolveEndpoint(runtimePath string, _ api.Options) (string, error) {
	if runtimePath == "" {
		return "", errors.New("runtime path cannot be empty")
	}

	return filepath.Join(runtimePath, "multiplexer", defaultSocketName), nil
}
