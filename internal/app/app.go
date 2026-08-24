package app

import (
	"fmt"

	"gitlab.com/mainops/uniShell/internal/runtime"
)

// App contains the application-level state shared by uniShell commands.
type App struct {
	Version string
	Commit  string
	Paths   runtime.Paths
}

// New creates the application state.
func New(version, commit string) (*App, error) {
	paths, err := runtime.NewPaths(version)
	if err != nil {
		return nil, fmt.Errorf("initialize runtime paths: %w", err)
	}

	return &App{
		Version: version,
		Commit:  commit,
		Paths:   paths,
	}, nil
}
