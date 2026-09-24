package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/persistence"
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
	Persistent             bool
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
	Persistent             bool
	AuthTokenFromStore     bool
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

	token, tokenFromStore, err := credentials.ResolveForRuntime(
		paths.Root,
		options.Persistent,
	)
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
		Persistent:             options.Persistent,
		AuthTokenFromStore:     tokenFromStore,
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

func (a *App) SaveFirstLaunch(shellName, multiplexerName string) error {
	if !a.Persistent {
		return nil
	}
	return persistence.SaveFirstLaunch(a.Paths.Root, persistence.LaunchConfig{
		Shell:                  shellName,
		ShellProfile:           a.ShellProfile,
		NoSharedRC:             a.NoSharedRC,
		Multiplexer:            multiplexerName,
		SessionName:            a.SessionName,
		MultiplexerSessionName: a.MultiplexerSessionName,
		NewSession:             a.NewSession,
	})
}

func (a *App) ValidateAuthentication() error {
	if len(a.AuthenticatedBundle) > 0 {
		return nil
	}

	started := time.Now()
	defer traceStartup("bundle verification", started)

	data, err := a.Bundle()
	if err != nil {
		return fmt.Errorf(
			"load embedded runtime bundle: %w",
			err,
		)
	}

	authenticated, err := bundle.OpenAuthenticated(data, a.AuthToken)
	if errors.Is(err, credentials.ErrAuthenticationFailed) && a.AuthTokenFromStore {
		token, resolveErr := credentials.Resolve()
		if resolveErr != nil {
			return fmt.Errorf("saved token was rejected and no replacement token was available: %w", resolveErr)
		}
		a.AuthToken = token
		a.AuthTokenFromStore = false
		authenticated, err = bundle.OpenAuthenticated(data, token)
	}
	if err != nil {
		return fmt.Errorf("authenticate runtime bundle: %w", err)
	}

	if a.Persistent && !a.AuthTokenFromStore {
		if err := credentials.StoreToken(a.Paths.Root, a.AuthToken); err != nil {
			return fmt.Errorf("save encrypted authentication token: %w", err)
		}
		a.AuthTokenFromStore = true
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

func (a *App) PreparePersistentRuntime() error {
	if !a.Persistent {
		return fmt.Errorf("persistent runtime is not enabled")
	}
	authenticated, err := a.authenticatedBundle()
	if err != nil {
		return err
	}
	_, err = a.ensurePersistentBundle(authenticated)
	return err
}

func (a *App) ensurePersistentBundle(authenticated []byte) (string, error) {
	digest := sha256.Sum256(authenticated)
	fingerprint := hex.EncodeToString(digest[:12])
	installed := filepath.Join(a.Paths.Runtime, "installed-"+fingerprint)

	if info, err := os.Stat(installed); err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("persistent runtime path %q is not a directory", installed)
		}
		if _, err := os.Stat(filepath.Join(installed, "bin")); err != nil {
			return "", fmt.Errorf("persistent runtime %q is incomplete; run 'unishell clean' before reinstalling: %w", installed, err)
		}
		if _, err := os.Stat(filepath.Join(installed, "config")); err != nil {
			return "", fmt.Errorf("persistent runtime %q is incomplete; run 'unishell clean' before reinstalling: %w", installed, err)
		}
		marker, err := os.ReadFile(filepath.Join(installed, ".unishell-installed"))
		if err != nil {
			return "", fmt.Errorf("persistent runtime %q is incomplete; run 'unishell clean' before reinstalling: %w", installed, err)
		}
		wantMarker := fmt.Sprintf("bundle=%s\n", fingerprint)
		if string(marker) != wantMarker {
			return "", fmt.Errorf("persistent runtime %q has mismatched installation metadata", installed)
		}
		return installed, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect persistent runtime: %w", err)
	}

	if err := os.MkdirAll(a.Paths.Runtime, 0700); err != nil {
		return "", fmt.Errorf("create persistent runtime directory: %w", err)
	}
	if err := os.Mkdir(installed, 0700); err != nil {
		return "", fmt.Errorf("create persistent runtime directory: %w", err)
	}

	reader, err := bundle.DecompressAuthenticatedReader(authenticated)
	if err != nil {
		return "", fmt.Errorf("decompress persistent runtime bundle: %w", err)
	}
	extractErr := bundle.ExtractArchive(reader, installed)
	closeErr := reader.Close()
	if extractErr != nil {
		return "", fmt.Errorf("extract persistent runtime bundle: %w", extractErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close persistent runtime bundle: %w", closeErr)
	}

	marker := []byte(fmt.Sprintf("bundle=%s\n", fingerprint))
	if err := os.WriteFile(filepath.Join(installed, ".unishell-installed"), marker, 0600); err != nil {
		return "", fmt.Errorf("record persistent runtime version: %w", err)
	}
	return installed, nil
}

func (a *App) preparePersistentSessionFiles(session *runtime.Session, installed string) error {
	if err := os.Remove(session.Paths.Bin); err != nil {
		return fmt.Errorf("prepare persistent session binary path: %w", err)
	}
	if err := os.Symlink(filepath.Join(installed, "bin"), session.Paths.Bin); err != nil {
		return fmt.Errorf("link persistent runtime binaries: %w", err)
	}
	if err := copyRuntimeDirectory(filepath.Join(installed, "config"), session.Paths.Config); err != nil {
		return fmt.Errorf("copy persistent runtime configuration: %w", err)
	}
	return nil
}

func copyRuntimeDirectory(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported runtime config entry %q", path)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputCloseErr := input.Close()
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputCloseErr != nil {
			return inputCloseErr
		}
		if closeErr != nil {
			return closeErr
		}
		return nil
	})
}

func (a *App) prepareRuntimeSession(mode runtime.SessionMode) (*runtime.Session, error) {
	if !a.Persistent {
		cleanupStarted := time.Now()
		if err := runtime.CleanupStale(a.Paths); err != nil {
			return nil, fmt.Errorf("clean stale runtime sessions: %w", err)
		}
		traceStartup("stale-session cleanup", cleanupStarted)
	}

	if mode == runtime.SessionModeMultiplexer && !a.Persistent {
		reconcileStarted := time.Now()
		if err := a.Multiplexer.Reconcile(a.Paths.Runtime); err != nil {
			return nil, fmt.Errorf("reconcile multiplexer sessions: %w", err)
		}
		traceStartup("multiplexer reconciliation", reconcileStarted)
	}

	authenticated, err := a.authenticatedBundle()
	if err != nil {
		return nil, err
	}
	installed := ""
	if a.Persistent {
		installed, err = a.ensurePersistentBundle(authenticated)
		if err != nil {
			return nil, err
		}
	}

	session, err := runtime.NewSessionWithMode(a.Paths, mode)
	if err != nil {
		return nil, fmt.Errorf("create runtime session: %w", err)
	}
	session.Persistent = a.Persistent
	sessionName, err := sessionNameForRuntime(session, a.SessionName, a.SessionNameSpecified)
	if err != nil {
		return nil, fmt.Errorf("generate runtime session name: %w", err)
	}
	if err := session.SetName(sessionName); err != nil {
		return nil, fmt.Errorf("set runtime session name: %w", err)
	}

	prepareStarted := time.Now()
	if err := session.Prepare(); err != nil {
		return nil, fmt.Errorf("prepare runtime session: %w", err)
	}
	traceStartup("runtime setup", prepareStarted)

	if a.Persistent {
		if err := a.preparePersistentSessionFiles(session, installed); err != nil {
			return nil, err
		}
	} else if err := extractAuthenticatedRuntime(
		authenticated,
		session.Paths.Runtime,
		a.Version,
		filepath.Join(filepath.Dir(a.Paths.Runtime), ".cache"),
	); err != nil {
		return nil, err
	}
	return session, nil
}

func (a *App) StartSession() (*runtime.Session, error) {
	return a.prepareRuntimeSession(runtime.SessionModeNormal)
}

// PrepareMultiplexerSession prepares an isolated managed multiplexer runtime.
// Persistent mode retains the session and shares the installed tool binaries.
func (a *App) PrepareMultiplexerSession() (*runtime.Session, error) {
	return a.prepareRuntimeSession(runtime.SessionModeMultiplexer)
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
		Persistent:  a.Persistent,
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
		Persistent:  a.Persistent,
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
