package multiplexer

import (
	"gitlab.com/mainops/uniShell/internal/multiplexer/tmux"
	"gitlab.com/mainops/uniShell/internal/multiplexer/zellij"
)

func DefaultRegistry() *Registry {
	return NewRegistry(
		tmux.New(),
		zellij.New(),
	)
}
