package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	SessionNameSpecified   bool
	MultiplexerSessionName string
	NewSession             bool
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
	AuthenticatedBundle    []byte
	Multiplexer            *multiplexer.Manager
	MultiplexerName        string
	SessionName            string
	SessionNameSpecified   bool
	MultiplexerSessionName string
	NewSession             bool
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
		source = bundle.EmbeddedView
	}

	manager := options.Multiplexer
	if manager == nil {
		manager = multiplexer.NewManager(
			multiplexer.DefaultRegistry(),
		)
	}

	multiplexerName := options.MultiplexerName

	sessionName := strings.TrimSpace(options.SessionName)
	sessionNameSpecified := options.SessionNameSpecified ||
		sessionName != ""

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
		SessionNameSpecified:   sessionNameSpecified,
		MultiplexerSessionName: options.MultiplexerSessionName,
		NewSession:             options.NewSession,
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

func (a *App) RequestedNewSession() bool {
	return a.NewSession
}

func (a *App) ValidateAuthentication() error {
	started := time.Now()
	defer traceStartup("bundle verification", started)

	data, err := a.Bundle()
	if err != nil {
		return fmt.Errorf(
			"load embedded runtime bundle: %w",
			err,
		)
	}

	authenticated, err := bundle.OpenAuthenticated(
		data,
		a.AuthToken,
	)
	if err != nil {
		return fmt.Errorf(
			"authenticate runtime bundle: %w",
			err,
		)
	}

	a.AuthenticatedBundle = authenticated

	return nil
}

func (a *App) authenticatedBundle() ([]byte, error) {
	if len(a.AuthenticatedBundle) > 0 {
		return a.AuthenticatedBundle, nil
	}

	data, err := a.Bundle()
	if err != nil {
		return nil, fmt.Errorf(
			"load embedded runtime bundle: %w",
			err,
		)
	}

	authenticated, err := bundle.OpenAuthenticated(
		data,
		a.AuthToken,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"authenticate runtime bundle: %w",
			err,
		)
	}

	return authenticated, nil
}

func extractAuthenticatedRuntime(
	authenticated []byte,
	destination string,
	version string,
	cacheDir string,
) error {
	cacheStarted := time.Now()
	cached, err := bundle.OpenArchiveCache(authenticated, version, cacheDir)
	if err == nil {
		defer cached.File.Close()
		traceStartupDuration("decompress runtime", cached.DecompressionElapsed)
		cacheStage := "runtime archive cache build"
		if cached.Hit {
			cacheStage = "runtime archive cache hit"
		}
		traceStartup(cacheStage, cacheStarted)

		extractStarted := time.Now()
		err = bundle.ExtractArchiveCache(cached.File, destination)
		traceStartup("extract runtime", extractStarted)
		if err != nil {
			return fmt.Errorf("extract runtime bundle: %w", err)
		}
		return nil
	}

	// The cache is an optimization. If its directory is unavailable or cannot
	// be written, keep the original streaming launch path.
	decompressStarted := time.Now()
	archiveReader, err := bundle.DecompressAuthenticatedReader(authenticated)
	if err != nil {
		return fmt.Errorf("decompress runtime bundle: %w", err)
	}
	decompressSetup := time.Since(decompressStarted)
	defer archiveReader.Close()

	timedReader := &startupTimedReader{reader: archiveReader}
	extractStarted := time.Now()
	err = bundle.ExtractArchive(timedReader, destination)
	extractElapsed := time.Since(extractStarted)
	traceStartupDuration(
		"decompress runtime",
		decompressSetup+timedReader.elapsed,
	)
	traceStartupDuration(
		"extract runtime",
		extractElapsed-timedReader.elapsed,
	)
	if err != nil {
		return fmt.Errorf("extract runtime bundle: %w", err)
	}
	return nil
}

func (a *App) StartSession() (*runtime.Session, error) {
	cleanupStarted := time.Now()
	if err := runtime.CleanupStale(a.Paths); err != nil {
		return nil, fmt.Errorf(
			"clean stale runtime sessions: %w",
			err,
		)
	}
	traceStartup("stale-session cleanup", cleanupStarted)

	session, err := runtime.NewSession(a.Paths)
	if err != nil {
		return nil, fmt.Errorf(
			"create runtime session: %w",
			err,
		)
	}
	sessionName, err := sessionNameForRuntime(
		session,
		a.SessionName,
		a.SessionNameSpecified,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"generate runtime session name: %w",
			err,
		)
	}

	if err := session.SetName(sessionName); err != nil {
		return nil, fmt.Errorf(
			"set runtime session name: %w",
			err,
		)
	}
	prepareStarted := time.Now()
	if err := session.Prepare(); err != nil {
		return nil, fmt.Errorf(
			"prepare runtime session: %w",
			err,
		)
	}
	traceStartup("runtime setup", prepareStarted)

	cleanupOnError := func(err error) (*runtime.Session, error) {
		_ = session.Cleanup()
		return nil, err
	}

	authenticated, err := a.authenticatedBundle()
	if err != nil {
		return cleanupOnError(err)
	}

	if err := extractAuthenticatedRuntime(
		authenticated,
		session.Paths.Runtime,
		a.Version,
		filepath.Join(filepath.Dir(a.Paths.Runtime), ".cache"),
	); err != nil {
		return cleanupOnError(err)
	}

	return session, nil
}

// PrepareMultiplexerSession creates and extracts a multiplexer runtime.
//
// The returned runtime remains owned by the caller. The caller must either
// pass it to CreateMultiplexerSession and eventually clean it up, or clean
// it directly when startup is abandoned.
func (a *App) PrepareMultiplexerSession() (*runtime.Session, error) {
	cleanupStarted := time.Now()
	if err := runtime.CleanupStale(a.Paths); err != nil {
		return nil, fmt.Errorf(
			"clean stale runtime sessions: %w",
			err,
		)
	}
	traceStartup("stale-session cleanup", cleanupStarted)

	reconcileStarted := time.Now()
	if err := a.Multiplexer.Reconcile(
		a.Paths.Runtime,
	); err != nil {
		return nil, fmt.Errorf(
			"reconcile multiplexer sessions: %w",
			err,
		)
	}
	traceStartup("multiplexer reconciliation", reconcileStarted)

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

	sessionName, err := sessionNameForRuntime(
		runtimeSession,
		a.SessionName,
		a.SessionNameSpecified,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"generate runtime session name: %w",
			err,
		)
	}

	if err := runtimeSession.SetName(sessionName); err != nil {
		return nil, fmt.Errorf(
			"set multiplexer runtime session name: %w",
			err,
		)
	}

	prepareStarted := time.Now()
	if err := runtimeSession.Prepare(); err != nil {
		return nil, fmt.Errorf(
			"prepare multiplexer runtime session: %w",
			err,
		)
	}
	traceStartup("runtime setup", prepareStarted)

	cleanupOnError := func(err error) (*runtime.Session, error) {
		_ = runtimeSession.Cleanup()
		return nil, err
	}

	authenticated, err := a.authenticatedBundle()
	if err != nil {
		return cleanupOnError(err)
	}

	if err := extractAuthenticatedRuntime(
		authenticated,
		runtimeSession.Paths.Runtime,
		a.Version,
		filepath.Join(filepath.Dir(a.Paths.Runtime), ".cache"),
	); err != nil {
		return cleanupOnError(err)
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
		return nil, fmt.Errorf("multiplexer runtime session is nil")
	}
	selectedShell, err := shell.Resolve(shellName, runtimeSession.Paths.Bin)
	if err != nil {
		return nil, fmt.Errorf("resolve shell: %w", err)
	}
	return a.CreateMultiplexerSessionResolved(runtimeSession, multiplexerName, selectedShell, startup)
}

func (a *App) CreateMultiplexerSessionResolved(
	runtimeSession *runtime.Session,
	multiplexerName string,
	selectedShell shell.Shell,
	startup shell.Startup,
) (*Session, error) {
	if runtimeSession == nil {
		return nil, fmt.Errorf("multiplexer runtime session is nil")
	}

	multiplexerSessionName, err := multiplexerSessionNameForRuntime(
		runtimeSession,
		a.MultiplexerSessionName,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"generate multiplexer session name: %w",
			err,
		)
	}

	if err := runtimeSession.RecordShellSelection(
		selectedShell.Name,
		selectedShell.Path,
		a.ShellProfile,
	); err != nil {
		return nil, fmt.Errorf(
			"record shell selection: %w",
			err,
		)
	}

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
		multiplexerSessionName,
		"",
		runtimeSession.Paths.Runtime,
		selectedShell.Name,
		selectedShell.Path,
		startup.Args,
		environment,
		a.MultiplexerOptions,
		runtimeSession.Metadata(),
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
		multiplexerSessionBaseName(a.MultiplexerSessionName),
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		Multiplexer: managed,
	}, nil
}

// DiscoverMultiplexerSessions returns all managed multiplexer sessions
// beneath the application's version runtime directory.
func (a *App) DiscoverMultiplexerSessions() (
	[]*multiplexer.ManagedSession,
	error,
) {
	sessions, err := a.Multiplexer.DiscoverAll(
		a.Paths.Runtime,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"discover multiplexer sessions: %w",
			err,
		)
	}

	return sessions, nil
}
