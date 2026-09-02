package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
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
	MultiplexerOptions     api.Options
	Shell                  string
	ShellProfile           string
	NoSharedRC             bool
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
	MultiplexerOptions     api.Options
	Shell                  string
	ShellProfile           string
	NoSharedRC             bool
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

	sessionName := options.SessionName
	if sessionName == "" {
		sessionName = "default"
	}

	multiplexerOptions := options.MultiplexerOptions

	if options.MultiplexerOptions.Tmux.CreateArgs == nil &&
		options.MultiplexerOptions.Zellij.CreateArgs == nil {
		multiplexerOptions, err =
			multiplexer.ParseOptionsFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf(
				"resolve multiplexer options: %w",
				err,
			)
		}
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
		MultiplexerOptions:     multiplexerOptions,
		Shell:                  options.Shell,
		ShellProfile:           options.ShellProfile,
		NoSharedRC:             options.NoSharedRC,
	}, nil
}

func (a *App) RequestedShell() string {
	return a.Shell
}

func (a *App) RequestedShellProfile() string {
	return a.ShellProfile
}

func (a *App) RequestedNoSharedRC() bool {
	return a.NoSharedRC
}

func (a *App) RequestedMultiplexer() string {
	return a.MultiplexerName
}

func (a *App) ValidateAuthentication() error {
	data, err := a.Bundle()
	if err != nil {
		return fmt.Errorf(
			"load embedded runtime bundle: %w",
			err,
		)
	}

	if _, err := bundle.Open(
		data,
		a.AuthToken,
	); err != nil {
		return fmt.Errorf(
			"authenticate runtime bundle: %w",
			err,
		)
	}

	return nil
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

// PrepareMultiplexerSession creates and extracts a multiplexer runtime.
//
// The returned runtime remains owned by the caller. The caller must either
// pass it to CreateMultiplexerSession and eventually clean it up, or clean
// it directly when startup is abandoned.
func (a *App) PrepareMultiplexerSession() (*runtime.Session, error) {
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

	cleanupOnError := func(err error) (*runtime.Session, error) {
		_ = runtimeSession.Cleanup()
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
		runtimeSession.Paths.Runtime,
	); err != nil {
		return cleanupOnError(
			fmt.Errorf(
				"extract runtime bundle: %w",
				err,
			),
		)
	}

	return runtimeSession, nil
}

func setEnvironment(
	env []string,
	key string,
	value string,
) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	found := false

	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if !found {
				result = append(
					result,
					prefix+value,
				)
				found = true
			}

			continue
		}

		result = append(result, entry)
	}

	if !found {
		result = append(result, prefix+value)
	}

	return result
}

func (a *App) CreateMultiplexerSession(
	runtimeSession *runtime.Session,
	multiplexerName string,
	shellName string,
	startup shell.Startup,
) (*Session, error) {
	if runtimeSession == nil {
		return nil, fmt.Errorf(
			"multiplexer runtime session is nil",
		)
	}

	selectedShell, err := shell.Resolve(
		shellName,
		runtimeSession.Paths.Bin,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve shell: %w",
			err,
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
		return nil, fmt.Errorf(
			"prepare multiplexer runtime: %w",
			err,
		)
	}

	endpoint := filepath.Join(
		multiplexerRuntime,
		multiplexerName+".sock",
	)

	environment, err := shell.NewEnvironmentForShell(
		runtimeSession.Paths.Bin,
		runtimeSession.Paths.Runtime,
		selectedShell,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"prepare shell environment: %w",
			err,
		)
	}

	for key, value := range startup.Env {
		environment = setEnvironment(
			environment,
			key,
			value,
		)
	}

	managedSession, err := a.Multiplexer.Create(
		multiplexerName,
		a.SessionName,
		a.MultiplexerSessionName,
		runtimeSession.Paths.Runtime,
		endpoint,
		selectedShell.Name,
		selectedShell.Path,
		startup.Args,
		environment,
		a.MultiplexerOptions,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create multiplexer session: %w",
			err,
		)
	}

	return &Session{
		Runtime:     runtimeSession,
		Multiplexer: managedSession,
	}, nil
}

func (a *App) StartMultiplexerSession() (*Session, error) {
	runtimeSession, err := a.PrepareMultiplexerSession()
	if err != nil {
		return nil, err
	}

	session, err := a.CreateMultiplexerSession(
		runtimeSession,
		a.MultiplexerName,
		a.Shell,
		shell.Startup{},
	)
	if err != nil {
		_ = runtimeSession.Cleanup()
		return nil, err
	}

	return session, nil
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
