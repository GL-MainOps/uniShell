package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	"gitlab.com/mainops/uniShell/internal/shell"
)

type BundleSource func() ([]byte, error)

type Options struct {
	Version                string
	Commit                 string
	Root                   string
	Bundle                 BundleSource
	Multiplexer            *multiplexer.Manager
	MultiplexerName        string
	SessionName            string
	MultiplexerSessionName string
}

type App struct {
	Version                string
	Commit                 string
	AuthToken              string
	Paths                  runtime.Paths
	Bundle                 BundleSource
	Multiplexer            *multiplexer.Manager
	MultiplexerName        string
	SessionName            string
	MultiplexerSessionName string
}

func New(options Options) (*App, error) {
	version := options.Version
	if version == "" {
		return nil, fmt.Errorf(
			"application version cannot be empty",
		)
	}

	paths, err := runtime.NewPaths(
		options.Root,
		version,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"initialize runtime paths: %w",
			err,
		)
	}

	token, err := credentials.Resolve()
	if err != nil {
		return nil, fmt.Errorf(
			"resolve authentication token: %w",
			err,
		)
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

	multiplexerName := options.MultiplexerName
	if multiplexerName == "" {
		multiplexerName = multiplexer.DefaultName
	}

	sessionName := options.SessionName
	if sessionName == "" {
		sessionName = "default"
	}

	return &App{
		Version:                version,
		Commit:                 options.Commit,
		AuthToken:              token,
		Paths:                  paths,
		Bundle:                 source,
		Multiplexer:            manager,
		MultiplexerName:        multiplexerName,
		SessionName:            sessionName,
		MultiplexerSessionName: options.MultiplexerSessionName,
	}, nil
}

func (a *App) StartSession() (*runtime.Session, error) {
	if err := runtime.CleanupStale(a.Paths); err != nil {
		return nil, fmt.Errorf(
			"clean stale runtime sessions: %w",
			err,
		)
	}

	session, err := runtime.NewSession(a.Paths)
	if err != nil {
		return nil, fmt.Errorf(
			"create runtime session: %w",
			err,
		)
	}

	if err := session.Prepare(); err != nil {
		return nil, fmt.Errorf(
			"prepare runtime session: %w",
			err,
		)
	}

	cleanupOnError := func(err error) (*runtime.Session, error) {
		_ = session.Cleanup()
		return nil, err
	}

	data, err := a.Bundle()
	if err != nil {
		return cleanupOnError(
			fmt.Errorf(
				"load embedded runtime bundle: %w",
				err,
			),
		)
	}

	archive, err := bundle.Open(
		data,
		a.AuthToken,
	)
	if err != nil {
		return cleanupOnError(
			fmt.Errorf(
				"open runtime bundle: %w",
				err,
			),
		)
	}

	if err := bundle.ExtractArchive(
		archive,
		session.Paths.Runtime,
	); err != nil {
		return cleanupOnError(
			fmt.Errorf(
				"extract runtime bundle: %w",
				err,
			),
		)
	}

	return session, nil
}

func (a *App) StartMultiplexerSession() (*Session, error) {
	if err := runtime.CleanupStale(a.Paths); err != nil {
		return nil, fmt.Errorf(
			"clean stale runtime sessions: %w",
			err,
		)
	}

	if err := a.Multiplexer.Reconcile(
		a.Paths.Runtime,
	); err != nil {
		return nil, fmt.Errorf(
			"reconcile multiplexer sessions: %w",
			err,
		)
	}

	runtimeSession, err := runtime.NewSessionWithMode(
		a.Paths,
		runtime.SessionModeMultiplexer,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create multiplexer runtime session: %w",
			err,
		)
	}

	if err := runtimeSession.Prepare(); err != nil {
		return nil, fmt.Errorf(
			"prepare multiplexer runtime session: %w",
			err,
		)
	}

	cleanupRuntime := func(err error) (*Session, error) {
		_ = runtimeSession.Cleanup()
		return nil, err
	}

	data, err := a.Bundle()
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"load embedded runtime bundle: %w",
				err,
			),
		)
	}

	archive, err := bundle.Open(
		data,
		a.AuthToken,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"open runtime bundle: %w",
				err,
			),
		)
	}

	if err := bundle.ExtractArchive(
		archive,
		runtimeSession.Paths.Runtime,
	); err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"extract runtime bundle: %w",
				err,
			),
		)
	}

	multiplexerRuntime := filepath.Join(
		runtimeSession.Paths.Runtime,
		"multiplexer",
	)

	if err := os.MkdirAll(
		multiplexerRuntime,
		0700,
	); err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"prepare multiplexer runtime: %w",
				err,
			),
		)
	}

	endpoint := filepath.Join(
		multiplexerRuntime,
		a.MultiplexerName+".sock",
	)

	environment, err := shell.NewEnvironment(
		runtimeSession.Paths.Bin,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"prepare shell environment: %w",
				err,
			),
		)
	}

	managedSession, err := a.Multiplexer.Create(
		a.MultiplexerName,
		a.SessionName,
		a.MultiplexerSessionName,
		runtimeSession.Paths.Runtime,
		endpoint,
		environment,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"create multiplexer session: %w",
				err,
			),
		)
	}

	return &Session{
		Runtime:     runtimeSession,
		Multiplexer: managedSession,
	}, nil
}

func (a *App) DiscoverMultiplexerSession() (*Session, error) {
	managed, err := a.Multiplexer.DiscoverByName(
		a.Paths.Runtime,
		a.SessionName,
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		Multiplexer: managed,
	}, nil
}
