package app

import (
	"fmt"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
)

type BundleSource func() ([]byte, error)

type Options struct {
	Version     string
	Commit      string
	Root        string
	Bundle      BundleSource
	Multiplexer *multiplexer.Manager
}

type App struct {
	Version     string
	Commit      string
	AuthToken   string
	Paths       runtime.Paths
	Bundle      BundleSource
	Multiplexer *multiplexer.Manager
}

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

	source := options.Bundle
	if source == nil {
		source = bundle.Embedded
	}

	manager := options.Multiplexer
	if manager == nil {
		manager = multiplexer.NewManager(
			multiplexer.DefaultRegistry(),
		)
	}

	return &App{
		Version:     version,
		Commit:      options.Commit,
		AuthToken:   token,
		Paths:       paths,
		Bundle:      source,
		Multiplexer: manager,
	}, nil
}

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

	cleanupOnError := func(err error) (*runtime.Session, error) {
		_ = session.Cleanup()
		return nil, err
	}

	data, err := a.Bundle()
	if err != nil {
		return cleanupOnError(
			fmt.Errorf("load embedded runtime bundle: %w", err),
		)
	}

	archive, err := bundle.Open(data, a.AuthToken)
	if err != nil {
		return cleanupOnError(
			fmt.Errorf("open runtime bundle: %w", err),
		)
	}

	if err := bundle.ExtractArchive(archive, session.Paths.Runtime); err != nil {
		return cleanupOnError(
			fmt.Errorf("extract runtime bundle: %w", err),
		)
	}

	return session, nil
}
