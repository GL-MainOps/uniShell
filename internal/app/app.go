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

// Options controls application initialization.
type Options struct {
	Version string
	Commit  string
	Root    string
}

// New creates the application state.
func New(options Options) (*App, error) {
	paths, err := runtime.NewPaths(options.Root, options.Version)
	if err != nil {
		return nil, fmt.Errorf("initialize runtime paths: %w", err)
	}

	return &App{
		Version: options.Version,
		Commit:  options.Commit,
		Paths:   paths,
	}, nil
}
