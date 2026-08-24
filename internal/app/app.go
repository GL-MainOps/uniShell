package app

import (
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/runtime"
)

const defaultRuntimeRoot = "/var/tmp/.lesscache"

// App contains application-level state shared by uniShell commands.
type App struct {
	Version   string
	Commit    string
	Paths     runtime.Paths
	AuthToken string
}

// Options controls application initialization.
type Options struct {
	Version string
	Commit  string
	Root    string
}

// New creates authenticated application state.
func New(options Options) (*App, error) {
	root := options.Root

	if root == "" {
		root = os.Getenv("UNISHELL_RUNTIME_DIR")
	}

	if root == "" {
		root = defaultRuntimeRoot
	}

	token, err := credentials.Resolve()

	if err != nil {
		return nil, fmt.Errorf("authentication: %w", err)
	}

	paths, err := runtime.NewPaths(root, options.Version)
	if err != nil {
		return nil, fmt.Errorf("initialize runtime paths: %w", err)
	}

	return &App{
		Version:   options.Version,
		Commit:    options.Commit,
		Paths:     paths,
		AuthToken: token,
	}, nil
}
