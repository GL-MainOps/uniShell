package app

import (
	"fmt"

	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/runtime"
)

const defaultVersion = "development"

// Options controls application construction.
type Options struct {
	Version string
	Commit  string
	Root    string
}

// App contains the application-level configuration and runtime state.
type App struct {
	Version   string
	Commit    string
	AuthToken string
	Paths     runtime.Paths
}

// New creates a new application instance.
func New(options Options) (*App, error) {
	version := options.Version
	if version == "" {
		return nil, fmt.Errorf("application version cannot be empty")
	}

	paths, err := runtime.NewPaths(options.Root, version)
	if err != nil {
		return nil, fmt.Errorf("initialize runtime paths: %w", err)
	}

	token, err := credentials.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve authentication token: %w", err)
	}

	return &App{
		Version:   version,
		Commit:    options.Commit,
		AuthToken: token,
		Paths:     paths,
	}, nil
}

// StartSession removes stale sessions and creates a new isolated runtime
// session.
func (a *App) StartSession() (*runtime.Session, error) {
	if err := runtime.CleanupStale(a.Paths); err != nil {
		return nil, fmt.Errorf("clean stale runtime sessions: %w", err)
	}

	session, err := runtime.NewSession(a.Paths)
	if err != nil {
		return nil, fmt.Errorf("create runtime session: %w", err)
	}

	if err := session.Prepare(); err != nil {
		return nil, fmt.Errorf("prepare runtime session: %w", err)
	}

	return session, nil
}
