package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/persistence"
	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
	"gitlab.com/mainops/uniShell/internal/shell"
	"gitlab.com/mainops/uniShell/internal/shell/profile"
)

var (
	version = "development"
	commit  = "unknown"
)

const (
	upgradeDirectLinkEnv = "UNISHELL_UPGRADE_DIRECT_LINK"
	gitlabReleasesURL    = "https://gitlab.com/mainops/uniShell/-/releases"
	upgradeBinaryName    = "unishell-slim"
	maximumUpgradeBytes  = 256 << 20
)

func newApplication(options cliOptions) (*app.App, error) {
	return app.New(app.Options{
		Version:                version,
		Commit:                 commit,
		Root:                   options.RuntimeDir,
		Persistent:             options.Persistent,
		Shell:                  options.Shell,
		ShellProfile:           options.ShellProfile,
		NoSharedRC:             options.NoSharedRC,
		MultiplexerName:        options.Multiplexer,
		SessionName:            options.SessionName,
		SessionNameSpecified:   options.SessionNameSpecified,
		MultiplexerSessionName: options.MultiplexerSessionName,
		NewSession:             options.NewSession,
	})
}

func environmentMap(entries []string) map[string]string {
	result := make(map[string]string, len(entries))

	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			continue
		}

		result[key] = value
	}

	return result
}

func interpolationEnvironment(
	sessionEnvironment map[string]string,
	runtimeDir string,
) (map[string]string, map[string]string) {
	if sessionEnvironment == nil {
		sessionEnvironment = make(map[string]string)
	}

	sessionEnvironment[shell.SessionRuntimeDirEnvName] = runtimeDir

	systemEnvironment := environmentMap(os.Environ())

	return sessionEnvironment, systemEnvironment
}

func main() {
	options, args, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}

	command, commandArgs := commandFromArgs(args)
	if command == "install" {
		root, rootErr := installRuntimeRoot(options)
		if rootErr == nil {
			rootErr = runInstall(root, commandArgs)
		}
		if rootErr != nil {
			printError(rootErr)
			os.Exit(exitCode(rootErr))
		}
		return
	}

	root, persistent, err := resolveRuntimeSelection(options)
	if err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}
	options.RuntimeDir = root
	options.Persistent = persistent
	if persistent {
		options, err = applyPersistentConfig(options, root)
		if err != nil {
			printError(err)
			os.Exit(exitCode(err))
		}
	}

	application, err := newApplication(options)
	if err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}
	if err := application.ValidateAuthentication(); err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}
	if err := run(application, args); err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}
}

func commandFromArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "shell", nil
	}
	return args[0], args[1:]
}

func installRuntimeRoot(options cliOptions) (string, error) {
	root := options.RuntimeDir
	if root == "" {
		root = strings.TrimSpace(os.Getenv("UNISHELL_RUNTIME_DIR"))
	}
	if root != "" {
		return filepath.Clean(root), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if pointed, err := persistence.RuntimePointer(home); err != nil {
		return "", err
	} else if pointed != "" {
		return pointed, nil
	}
	return filepath.Join(home, ".local", "unishell"), nil
}

func resolveRuntimeSelection(options cliOptions) (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve home directory: %w", err)
	}

	root := options.RuntimeDir
	if root == "" {
		root = strings.TrimSpace(os.Getenv("UNISHELL_RUNTIME_DIR"))
	}
	homeRuntime := filepath.Join(home, ".local", "unishell")
	if root == "" {
		if pointed, err := persistence.RuntimePointer(home); err != nil {
			return "", false, err
		} else if pointed != "" {
			root = pointed
		} else {
			root = homeRuntime
		}
	}
	root = filepath.Clean(root)

	executable, err := os.Executable()
	if err != nil {
		return "", false, fmt.Errorf("resolve uniShell executable path: %w", err)
	}
	installedBinary := samePath(executable, filepath.Join(home, ".local", "bin", "unishell"))
	configured := false
	if _, err := os.Stat(persistence.ConfigPath(root)); err == nil {
		configured = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("inspect persistent configuration: %w", err)
	}
	if root == homeRuntime && !configured {
		if _, err := os.Stat(persistence.ConfigPath(homeRuntime)); err == nil {
			configured = true
		}
	}
	persistent := installedBinary || configured
	if persistent {
		if installedBinary && !configured {
			if _, err := persistence.Create(root); err != nil {
				return "", false, err
			}
		}
		return root, true, nil
	}
	if root == homeRuntime && options.RuntimeDir == "" && os.Getenv("UNISHELL_RUNTIME_DIR") == "" {
		return "", false, nil
	}
	return root, false, nil
}

func samePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(leftAbs); err == nil {
		leftAbs = resolved
	}
	if resolved, err := filepath.EvalSymlinks(rightAbs); err == nil {
		rightAbs = resolved
	}
	return leftAbs == rightAbs
}

func applyPersistentConfig(options cliOptions, root string) (cliOptions, error) {
	config, err := persistence.Read(root)
	if errors.Is(err, os.ErrNotExist) {
		return options, nil
	}
	if err != nil {
		return cliOptions{}, err
	}
	launch := config.Launch
	if options.Shell == "" {
		options.Shell = launch.Shell
	}
	if options.ShellProfile == "" {
		options.ShellProfile = launch.ShellProfile
	}
	if !options.NoSharedRCSpecified && (launch.NoSharedRCSet || launch.Configured) {
		options.NoSharedRC = launch.NoSharedRC
	}
	if options.Multiplexer == "" {
		options.Multiplexer = launch.Multiplexer
	}
	if !options.SessionNameSpecified && launch.SessionName != "" {
		options.SessionName = launch.SessionName
		options.SessionNameSpecified = true
	}
	if options.MultiplexerSessionName == "" {
		options.MultiplexerSessionName = launch.MultiplexerSessionName
	}
	if !options.NewSession && launch.NewSession {
		options.NewSession = true
	}
	return options, nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		if code >= 0 {
			if code == 130 {
				return 0
			}

			return code
		}
	}

	return 1
}

func printError(err error) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) &&
		exitErr.ExitCode() == 130 {
		return
	}

	if errors.Is(err, credentials.ErrAuthenticationFailed) {
		fmt.Fprintln(os.Stderr, "Authentication Failed. Aborting...")
		return
	}

	var permissionErr *runtime.PermissionError

	if errors.As(err, &permissionErr) {
		fmt.Fprintf(
			os.Stderr,
			"uniShell: %v\n",
			permissionErr,
		)
		fmt.Fprintln(
			os.Stderr,
			"Fix the directory permissions or choose another runtime directory with --runtime-dir.",
		)
		return
	}

	fmt.Fprintf(os.Stderr, "uniShell: %v\n", err)
}

func run(application *app.App, args []string) error {
	command := "shell"
	commandArgs := args

	if len(args) > 0 {
		command = args[0]
		commandArgs = args[1:]
	}

	switch command {
	case "shell":
		return runShell(application, commandArgs)

	case "install":
		return runInstall(application.Paths.Root, commandArgs)

	case "update", "upgrade":
		return runUpdate(application, commandArgs)

	case "__refresh-runtime":
		if len(commandArgs) > 0 {
			return fmt.Errorf("internal runtime refresh does not accept arguments")
		}
		return application.PreparePersistentRuntime()

	case "clean":
		return runClean(application, commandArgs)

	case "list":
		return runList(application, commandArgs)

	case "detach":
		return runDetach(application, commandArgs)

	case "version":
		return runVersion(application, commandArgs)

	case "help", "--help", "-h":
		printHelp()
		return nil

	default:
		return fmt.Errorf(
			"unknown command %q; use 'help' for usage",
			command,
		)
	}
}

type firstLaunchRecorder interface {
	SaveFirstLaunch(shellName, multiplexerName string) error
}

func recordFirstLaunch(application any, shellName, multiplexerName string) error {
	recorder, ok := application.(firstLaunchRecorder)
	if !ok {
		return nil
	}
	return recorder.SaveFirstLaunch(shellName, multiplexerName)
}

type shellApplication interface {
	ValidateAuthentication() error
	StartSession() (*runtime.Session, error)
	StartMultiplexerSession() (*app.Session, error)
	DiscoverMultiplexerSession() (*app.Session, error)
	RequestedShell() string
	RequestedShellProfile() string
	RequestedNoSharedRC() bool
	RequestedMultiplexer() string
	PrepareMultiplexerSession() (*runtime.Session, error)
	CreateMultiplexerSessionResolved(
		*runtime.Session,
		string,
		shell.Shell,
		shell.Startup,
	) (*app.Session, error)
}

func runShell(application shellApplication, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf(
			"shell does not accept arguments",
		)
	}

	if err := application.ValidateAuthentication(); err != nil {
		return fmt.Errorf(
			"authenticate runtime bundle: %w",
			err,
		)
	}

	ctx, stop := shellSelectionContext()
	defer stop()

	multiplexerName, err := selectMultiplexer(
		ctx,
		application.RequestedMultiplexer(),
		os.Stdin,
		os.Stdout,
	)
	if err != nil {
		if errors.Is(
			err,
			errMultiplexerSelectionCancelled,
		) {
			return fmt.Errorf(
				"multiplexer selection cancelled",
			)
		}

		return fmt.Errorf(
			"select multiplexer: %w",
			err,
		)
	}

	if multiplexerName == multiplexerNone {
		return runDirectShell(
			application,
			ctx,
		)
	}

	return runMultiplexerShell(
		application,
		ctx,
		multiplexerName,
	)
}

func prepareShellStartup(
	application shellApplication,
	selected shell.Shell,
	runtimeDir string,
	sessionEnvironment map[string]string,
	systemEnvironment map[string]string,
) (shell.Startup, error) {

	profileName := application.RequestedShellProfile()
	includeShared := !application.RequestedNoSharedRC()

	if profileName == "" && !includeShared {
		return shell.Startup{}, nil
	}

	profileRoot := filepath.Join(
		runtimeDir,
		"config",
		"shell",
	)

	loader := profile.NewLoader(profileRoot)

	loaded, err := loader.Load(
		selected.Name,
		profileName,
		includeShared,
	)
	if err != nil {
		return shell.Startup{}, fmt.Errorf(
			"load shell profile %q: %w",
			profileName,
			err,
		)
	}

	startup, err := shell.PrepareProfileStartup(
		runtimeDir,
		selected.Name,
		profileName,
		loaded,
		includeShared,
		sessionEnvironment,
		systemEnvironment,
	)
	if err != nil {
		return shell.Startup{}, fmt.Errorf(
			"prepare shell profile startup: %w",
			err,
		)
	}

	return startup, nil
}

func setSessionEnvironment(
	startup shell.Startup,
	sessionEnvironment map[string]string,
) shell.Startup {
	if startup.Env == nil {
		startup.Env = make(map[string]string)
	}

	for key, value := range sessionEnvironment {
		startup.Env[key] = value
	}

	return startup
}

func runDirectShell(
	application shellApplication,
	ctx context.Context,
) error {
	runtimeSession, err := application.StartSession()
	if err != nil {
		return fmt.Errorf(
			"prepare shell runtime: %w",
			err,
		)
	}

	cleanupRuntime := func(err error) error {
		if cleanupErr := runtimeSession.Cleanup(); cleanupErr != nil {
			return fmt.Errorf(
				"%w; cleanup runtime session: %v",
				err,
				cleanupErr,
			)
		}

		return err
	}

	selected, err := selectResolvedShell(
		ctx,
		runtimeSession.Paths.Bin,
		application.RequestedShell(),
		os.Stdin,
		os.Stdout,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"select shell: %w",
				err,
			),
		)
	}

	if err := runtimeSession.SetShellSelection(
		selected.Name,
		selected.Path,
		application.RequestedShellProfile(),
	); err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"record shell selection: %w",
				err,
			),
		)
	}

	sessionEnvironment, err := runtimeSession.Environment()
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"prepare session environment: %w",
				err,
			),
		)
	}

	sessionEnvironment, systemEnvironment :=
		interpolationEnvironment(
			sessionEnvironment,
			runtimeSession.Paths.Runtime,
		)

	startup, err := prepareShellStartup(
		application,
		selected,
		runtimeSession.Paths.Runtime,
		sessionEnvironment,
		systemEnvironment,
	)
	if err != nil {
		return cleanupRuntime(err)
	}

	startup = setSessionEnvironment(
		startup,
		sessionEnvironment,
	)

	command, err := shell.NewCommand(
		selected,
		runtimeSession.Paths.Bin,
		runtimeSession.Paths.Runtime,
		startup,
		nil,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"prepare shell command: %w",
				err,
			),
		)
	}

	if err := command.Run(); err != nil {
		return cleanupRuntime(err)
	}
	if err := recordFirstLaunch(application, selected.Name, multiplexerNone); err != nil {
		return cleanupRuntime(err)
	}

	return runtimeSession.Cleanup()
}

type multiSessionDiscovery interface {
	DiscoverMultiplexerSessions() ([]*multiplexer.ManagedSession, error)
}

type persistentSessionApplication interface {
	IsPersistent() bool
}

func applicationPersistent(application any) bool {
	persistent, ok := application.(persistentSessionApplication)
	return ok && persistent.IsPersistent()
}

type newSessionRequest interface {
	RequestedNewSession() bool
}

func runMultiplexerShell(
	application shellApplication,
	ctx context.Context,
	multiplexerName string,
) error {
	forceNew := false
	if requested, ok := application.(newSessionRequest); ok {
		forceNew = requested.RequestedNewSession()
	}

	if !forceNew {
		if discovery, ok := application.(multiSessionDiscovery); ok {
			sessions, err := discovery.DiscoverMultiplexerSessions()
			if err != nil {
				return fmt.Errorf("discover multiplexer sessions: %w", err)
			}

			live := make([]*app.Session, 0, len(sessions))
			hasMultiplexerIdentity := false
			for _, managed := range sessions {
				if managed == nil || managed.Backend == nil {
					continue
				}
				if managed.Metadata.Multiplexer != "" {
					hasMultiplexerIdentity = true
				}
				if managed.Backend.IsAlive(managed.Session) {
					live = append(live, &app.Session{Multiplexer: managed, Persistent: applicationPersistent(application)})
				}
			}

			// Older session providers may not expose the multiplexer identity in
			// their all-session result. Preserve their existing name-based lookup.
			if !hasMultiplexerIdentity {
				session, err := application.DiscoverMultiplexerSession()
				if err == nil {
					printReattachMessage(application, session)
					return session.Attach()
				}
				if !errors.Is(err, multiplexer.ErrSessionNotFound) {
					return fmt.Errorf("discover multiplexer session: %w", err)
				}
			} else {
				selected, err := chooseMultiplexerSession(
					ctx, multiplexerName, live, os.Stdin, os.Stdout,
				)
				if err != nil {
					if errors.Is(err, errMultiplexerSessionChoiceCancelled) {
						return fmt.Errorf("multiplexer session selection cancelled")
					}
					return err
				}
				if selected != nil {
					printReattachMessage(application, selected)
					if err := selected.Attach(); err != nil {
						return err
					}
					shellName := selected.Multiplexer.Metadata.ShellName
					if shellName == "" {
						shellName = application.RequestedShell()
					}
					return recordFirstLaunch(application, shellName, selected.Multiplexer.Metadata.Multiplexer)
				}
			}
		} else {
			session, err := application.DiscoverMultiplexerSession()
			if err == nil {
				printReattachMessage(application, session)
				if err := session.Attach(); err != nil {
					return err
				}
				shellName := session.Multiplexer.Metadata.ShellName
				if shellName == "" {
					shellName = application.RequestedShell()
				}
				return recordFirstLaunch(application, shellName, session.Multiplexer.Metadata.Multiplexer)
			}
			if !errors.Is(err, multiplexer.ErrSessionNotFound) {
				return fmt.Errorf("discover multiplexer session: %w", err)
			}
		}
	}

	runtimeSession, err := application.PrepareMultiplexerSession()
	if err != nil {
		return fmt.Errorf(
			"prepare multiplexer runtime: %w",
			err,
		)
	}

	cleanupRuntime := func(err error) error {
		if cleanupErr := runtimeSession.Cleanup(); cleanupErr != nil {
			return fmt.Errorf(
				"%w; cleanup runtime session: %v",
				err,
				cleanupErr,
			)
		}

		return err
	}

	selected, err := selectResolvedShell(
		ctx,
		runtimeSession.Paths.Bin,
		application.RequestedShell(),
		os.Stdin,
		os.Stdout,
	)
	if err != nil {
		if errors.Is(err, errShellSelectionCancelled) {
			return cleanupRuntime(
				fmt.Errorf(
					"shell selection cancelled",
				),
			)
		}

		return cleanupRuntime(
			fmt.Errorf(
				"select shell: %w",
				err,
			),
		)
	}

	sessionEnvironment, err := runtimeSession.Environment()
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"prepare session environment: %w",
				err,
			),
		)
	}

	sessionEnvironment, systemEnvironment :=
		interpolationEnvironment(
			sessionEnvironment,
			runtimeSession.Paths.Runtime,
		)

	startup, err := prepareShellStartup(
		application,
		selected,
		runtimeSession.Paths.Runtime,
		sessionEnvironment,
		systemEnvironment,
	)
	if err != nil {
		return cleanupRuntime(err)
	}

	startup = setSessionEnvironment(
		startup,
		sessionEnvironment,
	)

	session, err := application.CreateMultiplexerSessionResolved(
		runtimeSession,
		multiplexerName,
		selected,
		startup,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"create multiplexer session: %w",
				err,
			),
		)
	}

	if err := session.Attach(); err != nil {
		if cleanupErr := session.Cleanup(); cleanupErr != nil {
			return fmt.Errorf(
				"attach new multiplexer session: %w; cleanup session: %v",
				err,
				cleanupErr,
			)
		}

		return fmt.Errorf(
			"attach new multiplexer session: %w",
			err,
		)
	}

	if err := recordFirstLaunch(application, selected.Name, multiplexerName); err != nil {
		return err
	}
	return nil
}

var errMultiplexerSessionChoiceCancelled = errors.New(
	"multiplexer session selection cancelled",
)

func chooseMultiplexerSession(
	ctx context.Context,
	requested string,
	sessions []*app.Session,
	in io.Reader,
	out io.Writer,
) (*app.Session, error) {
	matching := make([]*app.Session, 0, len(sessions))
	for _, session := range sessions {
		if session != nil && session.Multiplexer != nil &&
			session.Multiplexer.Metadata.Multiplexer == requested {
			matching = append(matching, session)
		}
	}

	sort.SliceStable(matching, func(i, j int) bool {
		return matching[i].Multiplexer.Metadata.CreatedAt.After(
			matching[j].Multiplexer.Metadata.CreatedAt,
		)
	})
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].Multiplexer.Metadata.CreatedAt.After(
			sessions[j].Multiplexer.Metadata.CreatedAt,
		)
	})

	if len(matching) == 1 {
		return matching[0], nil
	}
	if len(sessions) == 0 {
		return nil, nil
	}

	candidates := matching
	if len(candidates) == 0 {
		candidates = sessions
		fmt.Fprintf(out, "An active %s session already exists.\n", candidates[0].Multiplexer.Metadata.Multiplexer)
		fmt.Fprintln(out, "Choose a session to attach to, or start a new session.")
	} else {
		fmt.Fprintf(out, "Multiple active %s sessions exist.\n", requested)
		fmt.Fprintln(out, "Choose a session to attach to, or start a new session.")
	}

	for i, session := range candidates {
		metadata := session.Multiplexer.Metadata
		name := metadata.MultiplexerSessionName
		if name == "" {
			name = metadata.Name
		}
		sessionID := filepath.Base(session.Multiplexer.Session.Runtime)
		if session.Multiplexer.Session.Runtime == "" || sessionID == "." {
			sessionID = metadata.ID
		}
		fmt.Fprintf(out, "  %d. %s — %s (id: %s)\n", i+1, metadata.Multiplexer, name, sessionID)
	}
	fmt.Fprintln(out, "  n. start a new session")
	fmt.Fprintln(out, "  q. cancel")

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "Select an existing session, new session, or quit [number/n/q]: ")
		input := make(chan string, 1)
		errCh := make(chan error, 1)
		go func() {
			if scanner.Scan() {
				input <- scanner.Text()
				return
			}
			err := scanner.Err()
			if err == nil {
				err = io.EOF
			}
			errCh <- err
		}()

		select {
		case <-ctx.Done():
			return nil, errMultiplexerSessionChoiceCancelled
		case err := <-errCh:
			if errors.Is(err, io.EOF) {
				return nil, errMultiplexerSessionChoiceCancelled
			}
			return nil, fmt.Errorf("read multiplexer session selection: %w", err)
		case value := <-input:
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "n" || value == "new" {
				return nil, nil
			}
			if value == "q" || value == "quit" || value == "exit" {
				return nil, errMultiplexerSessionChoiceCancelled
			}
			index, err := strconv.Atoi(value)
			if err == nil && index >= 1 && index <= len(candidates) {
				return candidates[index-1], nil
			}
			fmt.Fprintln(out, "Invalid selection. Choose one of the listed numbers, n, or q.")
		}
	}
}

func printReattachMessage(
	application shellApplication,
	session *app.Session,
) {
	if session == nil ||
		session.Multiplexer == nil {
		return
	}

	existingShell := session.Multiplexer.Metadata.ShellName
	if existingShell == "" {
		return
	}

	requestedShell := application.RequestedShell()

	if requestedShell == existingShell {
		return
	}

	fmt.Println("Existing uniShell session found.")
	fmt.Printf(
		"Requested shell: %s\n",
		requestedShell,
	)
	fmt.Printf(
		"Existing session shell: %s\n",
		existingShell,
	)
	fmt.Printf(
		"Attaching to the existing %s session.\n",
		existingShell,
	)
}

type sessionApplication interface {
	DiscoverMultiplexerSession() (*app.Session, error)
}

func runDetach(application sessionApplication, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf(
			"detach does not accept arguments",
		)
	}

	session, err := application.DiscoverMultiplexerSession()
	if err != nil {
		if errors.Is(err, multiplexer.ErrSessionNotFound) {
			return nil
		}

		return fmt.Errorf(
			"discover multiplexer session: %w",
			err,
		)
	}

	if err := session.Detach(); err != nil {
		return fmt.Errorf(
			"detach multiplexer session: %w",
			err,
		)
	}

	return nil
}

func runInstall(root string, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("install does not accept argument %q", args[0])
	}
	if root == "" {
		return fmt.Errorf("persistent runtime directory cannot be empty")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve persistent runtime directory: %w", err)
	}
	if _, err := persistence.Create(root); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	destination := filepath.Join(home, ".local", "bin", "unishell")
	if err := installExecutable(destination); err != nil {
		return err
	}
	if err := persistence.SetRuntimePointer(home, root); err != nil {
		return err
	}
	fmt.Printf("uniShell installed to %s\n", destination)
	fmt.Printf("Persistent runtime: %s\n", root)
	fmt.Printf("Configuration: %s\n", persistence.ConfigPath(root))
	return nil
}

func installExecutable(destination string) error {
	source, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current uniShell executable: %w", err)
	}
	if samePath(source, destination) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("create installed binary directory: %w", err)
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open current uniShell executable: %w", err)
	}
	defer input.Close()

	output, err := os.CreateTemp(filepath.Dir(destination), ".unishell-install-*.tmp")
	if err != nil {
		return fmt.Errorf("stage installed uniShell executable: %w", err)
	}
	tempPath := output.Name()
	defer os.Remove(tempPath)
	if err := output.Chmod(0755); err != nil {
		_ = output.Close()
		return fmt.Errorf("set installed uniShell permissions: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return fmt.Errorf("copy uniShell executable: %w", err)
	}
	if err := output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("sync installed uniShell executable: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close installed uniShell executable: %w", err)
	}
	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("install uniShell executable: %w", err)
	}
	return nil
}

func runUpdate(application *app.App, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("update does not accept argument %q", args[0])
	}
	if !application.Persistent {
		return fmt.Errorf("update requires a persistent installation; run 'unishell install' first")
	}

	tag, err := latestReleaseTag()
	if err != nil {
		return err
	}
	if strings.TrimPrefix(tag, "v") == strings.TrimPrefix(application.Version, "v") &&
		strings.TrimSpace(os.Getenv(upgradeDirectLinkEnv)) == "" {
		if err := application.PreparePersistentRuntime(); err != nil {
			return fmt.Errorf("refresh persistent runtime: %w", err)
		}
		fmt.Printf("uniShell %s is already the latest release; runtime is ready.\n", application.Version)
		return nil
	}

	manifest, err := downloadReleaseManifest(tag)
	if err != nil {
		return err
	}
	checksum, ok := checksumFor(manifest, upgradeBinaryName)
	if !ok {
		return fmt.Errorf("release %s does not publish checksum for %s", tag, upgradeBinaryName)
	}

	binaryURL := gitlabReleasesURL + "/" + tag + "/downloads/" + upgradeBinaryName
	if directURL := strings.TrimSpace(os.Getenv(upgradeDirectLinkEnv)); directURL != "" {
		parsed, err := url.Parse(directURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("%s must be an HTTPS URL", upgradeDirectLinkEnv)
		}
		binaryURL = directURL
	}

	stagedBinary, err := downloadUpgradeBinary(binaryURL, checksum)
	if err != nil {
		return err
	}
	defer os.Remove(stagedBinary)

	refresh := exec.Command(stagedBinary, "--runtime-dir", application.Paths.Root, "__refresh-runtime")
	refresh.Stdout = os.Stdout
	refresh.Stderr = os.Stderr
	refresh.Stdin = os.Stdin
	if err := refresh.Run(); err != nil {
		return fmt.Errorf("prepare updated persistent runtime: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	destination := filepath.Join(home, ".local", "bin", "unishell")
	if err := installBinaryFrom(stagedBinary, destination); err != nil {
		return err
	}
	fmt.Printf("Updated uniShell to %s.\n", tag)
	return nil
}

func latestReleaseTag() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Get(gitlabReleasesURL + "/permalink/latest")
	if err != nil {
		return "", fmt.Errorf("check latest uniShell release: %w", err)
	}
	defer response.Body.Close()
	location, err := response.Location()
	if err != nil {
		return "", fmt.Errorf("latest uniShell release did not redirect to a version: %w", err)
	}
	tag := filepath.Base(location.Path)
	if !validReleaseTag(tag) {
		return "", fmt.Errorf("latest uniShell release returned invalid tag %q", tag)
	}
	return tag, nil
}

func validReleaseTag(tag string) bool {
	if !strings.HasPrefix(tag, "v") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func downloadReleaseManifest(tag string) ([]byte, error) {
	manifestURL := gitlabReleasesURL + "/" + tag + "/downloads/SHA256SUMS"
	request, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create release manifest request: %w", err)
	}
	client := secureHTTPClient(2 * time.Minute)
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download release checksum manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download release checksum manifest: HTTP %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read release checksum manifest: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("release checksum manifest is empty")
	}
	return data, nil
}

func checksumFor(manifest []byte, name string) (string, bool) {
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != name || len(fields[0]) != sha256.Size*2 {
			continue
		}
		if _, err := hex.DecodeString(fields[0]); err == nil {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

func downloadUpgradeBinary(binaryURL, expectedChecksum string) (string, error) {
	request, err := http.NewRequest(http.MethodGet, binaryURL, nil)
	if err != nil {
		return "", fmt.Errorf("create binary download request: %w", err)
	}
	client := secureHTTPClient(5 * time.Minute)
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download uniShell release binary: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download uniShell release binary: HTTP %s", response.Status)
	}

	file, err := os.CreateTemp("", ".unishell-update-*")
	if err != nil {
		return "", fmt.Errorf("stage uniShell update: %w", err)
	}
	path := file.Name()
	if err := file.Chmod(0700); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("protect staged uniShell update: %w", err)
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maximumUpgradeBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("save uniShell release binary: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close staged uniShell update: %w", closeErr)
	}
	if written == 0 || written > maximumUpgradeBytes {
		_ = os.Remove(path)
		return "", fmt.Errorf("uniShell release binary has an invalid size")
	}
	staged, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("verify staged uniShell update: %w", err)
	}
	hasher := sha256.New()
	_, hashErr := io.Copy(hasher, staged)
	closeErr = staged.Close()
	if hashErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("hash staged uniShell update: %w", hashErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close staged uniShell update: %w", closeErr)
	}
	actualText := hex.EncodeToString(hasher.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actualText), []byte(expectedChecksum)) != 1 {
		_ = os.Remove(path)
		return "", fmt.Errorf("uniShell release binary checksum does not match the official manifest")
	}
	return path, nil
}

func secureHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			if request.URL.Scheme != "https" {
				return fmt.Errorf("refusing insecure release redirect")
			}
			return nil
		},
	}
}

func installBinaryFrom(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("create installed binary directory: %w", err)
	}
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open downloaded uniShell executable: %w", err)
	}
	defer input.Close()
	output, err := os.CreateTemp(filepath.Dir(destination), ".unishell-update-*.tmp")
	if err != nil {
		return fmt.Errorf("stage installed uniShell update: %w", err)
	}
	tempPath := output.Name()
	defer os.Remove(tempPath)
	if err := output.Chmod(0755); err != nil {
		_ = output.Close()
		return fmt.Errorf("set updated uniShell permissions: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return fmt.Errorf("copy updated uniShell executable: %w", err)
	}
	if err := output.Sync(); err != nil {
		_ = output.Close()
		return fmt.Errorf("sync updated uniShell executable: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close updated uniShell executable: %w", err)
	}
	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("install updated uniShell executable: %w", err)
	}
	return nil
}

type listApplication interface {
	ListSessions() ([]*app.CleanSession, error)
}

func runList(application listApplication, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("list does not accept argument %q", args[0])
	}

	sessions, err := application.ListSessions()
	if err != nil {
		return fmt.Errorf("list managed uniShell sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No managed uniShell sessions found.")
		return nil
	}

	fmt.Println("SESSION ID\tSESSION NAME\tSESSION TYPE")
	for _, session := range sessions {
		if session == nil {
			continue
		}
		sessionType := string(session.Metadata.Mode)
		switch session.Metadata.Mode {
		case sessionmeta.ModeNormal:
			sessionType = "direct"
		case sessionmeta.ModeMultiplexer:
			sessionType = "multiplexer"
		}
		fmt.Printf("%s\t%s\t%s\n", session.Metadata.ID, session.Metadata.Name, sessionType)
	}
	return nil
}

type cleanOptions struct {
	Target    string
	Installed bool
}

func parseCleanArgs(args []string) (cleanOptions, error) {
	var options cleanOptions

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--installed":
			options.Installed = true

		case arg == "--target":
			if i+1 >= len(args) {
				return cleanOptions{}, fmt.Errorf(
					"--target requires a session name",
				)
			}

			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return cleanOptions{}, fmt.Errorf(
					"--target requires a session name",
				)
			}

			options.Target = value
			i++

		case strings.HasPrefix(arg, "--target="):
			value := strings.TrimSpace(
				strings.TrimPrefix(arg, "--target="),
			)

			if value == "" {
				return cleanOptions{}, fmt.Errorf(
					"--target requires a session name",
				)
			}

			options.Target = value

		default:
			return cleanOptions{}, fmt.Errorf(
				"clean does not accept argument %q",
				arg,
			)
		}
	}

	if options.Installed && options.Target != "" {
		return cleanOptions{}, fmt.Errorf("clean --installed cannot be combined with --target")
	}

	return options, nil
}

type cleanApplication interface {
	DiscoverCleanSessions() ([]*app.CleanSession, error)
	TerminateNormalSession(*app.CleanSession) error
	CleanupMultiplexerSession(*app.CleanSession) error
}

var errCleanSelectionCancelled = errors.New(
	"clean session selection cancelled",
)

var errCleanSelectionAll = errors.New(
	"all clean sessions selected",
)

func selectCleanSession(
	sessions []*app.CleanSession,
	reader *bufio.Reader,
) (*app.CleanSession, error) {
	fmt.Println("Managed uniShell sessions:")
	fmt.Println()

	for index, session := range sessions {
		fmt.Printf(
			"%d) %s\n",
			index+1,
			formatCleanSessionLabel(session.Metadata),
		)
	}

	if len(sessions) > 1 {
		fmt.Println("a) ALL")
	}

	fmt.Println("q) quit")

	for {
		fmt.Fprint(
			os.Stdout,
			"\nEnter session number: ",
		)

		response, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, errCleanSelectionCancelled
			}

			return nil, err
		}

		value := strings.TrimSpace(response)

		if strings.EqualFold(value, "q") {
			return nil, errCleanSelectionCancelled
		}

		if len(sessions) > 1 &&
			strings.EqualFold(value, "a") {
			return nil, errCleanSelectionAll
		}

		index, err := strconv.Atoi(value)
		if err != nil ||
			index < 1 ||
			index > len(sessions) {
			continue
		}

		return sessions[index-1], nil
	}
}

func confirmCleanSession(
	name string,
	reader *bufio.Reader,
) (bool, error) {
	fmt.Printf(
		"Are you sure you want to clean session %q? [y/N]: ",
		name,
	)

	response, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes", nil
}

func confirmCleanAllSessions(
	reader *bufio.Reader,
) (bool, error) {
	fmt.Print(
		"Are you sure you want to clean ALL managed uniShell sessions? [y/N]: ",
	)

	response, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes", nil
}

type persistentCleanApplication interface {
	IsPersistent() bool
	PersistentRoot() string
	CleanPersistentRuntime() error
}

func confirmCleanPersistentRuntime(reader *bufio.Reader, root string) (bool, error) {
	fmt.Printf("Remove all installed uniShell runtime data, configuration, and saved credentials under %q? The executable will remain. [y/N]: ", root)
	response, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return false, nil
	}

	fmt.Printf("Type the runtime directory %q to confirm deletion: ", root)
	response, err = reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(response) == root, nil
}

func runClean(
	application cleanApplication,
	args []string,
) error {
	reader := bufio.NewReader(os.Stdin)

	options, err := parseCleanArgs(args)
	if err != nil {
		return err
	}
	if options.Installed {
		persistent, ok := application.(persistentCleanApplication)
		if !ok || !persistent.IsPersistent() {
			return fmt.Errorf("clean --installed requires a persistent installation")
		}
		confirmed, err := confirmCleanPersistentRuntime(reader, persistent.PersistentRoot())
		if err != nil {
			return fmt.Errorf("read persistent clean confirmation: %w", err)
		}
		if !confirmed {
			return nil
		}
		return persistent.CleanPersistentRuntime()
	}

	sessions, err := application.DiscoverCleanSessions()
	if err != nil {
		return fmt.Errorf(
			"discover clean sessions: %w",
			err,
		)
	}

	if len(sessions) == 0 {
		fmt.Println("No managed uniShell sessions found.")
		return nil
	}

	var targets []*app.CleanSession
	allSelected := false

	if options.Target != "" {
		for _, candidate := range sessions {
			if candidate.Metadata.Name == options.Target {
				targets = []*app.CleanSession{
					candidate,
				}
				break
			}
		}

		if len(targets) == 0 {
			return fmt.Errorf(
				"managed session %q not found",
				options.Target,
			)
		}
	} else {
		target, selectionErr := selectCleanSession(
			sessions,
			reader,
		)
		if selectionErr != nil {
			if errors.Is(
				selectionErr,
				errCleanSelectionCancelled,
			) {
				return nil
			}

			if errors.Is(
				selectionErr,
				errCleanSelectionAll,
			) {
				targets = sessions
				allSelected = true
			} else {
				return fmt.Errorf(
					"select clean session: %w",
					selectionErr,
				)
			}
		} else {
			targets = []*app.CleanSession{
				target,
			}
		}
	}

	if allSelected {
		confirmed, err := confirmCleanAllSessions(reader)
		if err != nil {
			return fmt.Errorf(
				"read clean confirmation: %w",
				err,
			)
		}

		if !confirmed {
			return nil
		}
	} else {
		confirmed, err := confirmCleanSession(
			targets[0].Metadata.Name,
			reader,
		)
		if err != nil {
			return fmt.Errorf(
				"read clean confirmation: %w",
				err,
			)
		}

		if !confirmed {
			return nil
		}
	}

	revalidatedSessions, err := application.DiscoverCleanSessions()
	if err != nil {
		return fmt.Errorf(
			"revalidate clean sessions: %w",
			err,
		)
	}

	revalidatedTargets := make(
		[]*app.CleanSession,
		0,
		len(targets),
	)

	for _, target := range targets {
		var revalidatedTarget *app.CleanSession

		for _, candidate := range revalidatedSessions {
			if candidate.Metadata.ID == target.Metadata.ID &&
				candidate.RuntimeDir == target.RuntimeDir {
				revalidatedTarget = candidate
				break
			}
		}

		if revalidatedTarget == nil {
			return fmt.Errorf(
				"managed session %q no longer exists",
				target.Metadata.Name,
			)
		}

		revalidatedTargets = append(
			revalidatedTargets,
			revalidatedTarget,
		)
	}

	var cleanupErrors []error

	for _, target := range revalidatedTargets {
		switch target.Metadata.Mode {
		case sessionmeta.ModeNormal:
			if err := application.TerminateNormalSession(
				target,
			); err != nil {
				cleanupErrors = append(
					cleanupErrors,
					fmt.Errorf(
						"terminate clean session %q: %w",
						target.Metadata.Name,
						err,
					),
				)
			}

		case sessionmeta.ModeMultiplexer:
			if err := application.CleanupMultiplexerSession(
				target,
			); err != nil {
				cleanupErrors = append(
					cleanupErrors,
					fmt.Errorf(
						"cleanup multiplexer session %q: %w",
						target.Metadata.Name,
						err,
					),
				)
			}

		default:
			cleanupErrors = append(
				cleanupErrors,
				fmt.Errorf(
					"clean session %q uses unsupported termination mode %q",
					target.Metadata.Name,
					target.Metadata.Mode,
				),
			)
		}
	}

	if len(cleanupErrors) > 0 {
		return errors.Join(cleanupErrors...)
	}

	return nil
}

func runVersion(app *app.App, args []string) error {
	fmt.Printf("uniShell %s (%s)\n", app.Version, app.Commit)
	return nil
}

func printHelp() {
	fmt.Println(`uniShell - portable Linux shell environment

Usage:
  unishell [options] [command]

Commands:
  shell       Start or attach to the uniShell environment
  install     Install persistent uniShell under ~/.local
  update/upgrade Update the installed binary and persistent runtime
  clean       Clean a session; use --installed to remove the installed runtime
  list        List managed direct and multiplexer sessions
  detach      Detach from the current uniShell multiplexer session
  version     Display version information
  help        Display this help message

Options:
  --shell NAME
      Select the shell to use.

  --runtime-dir PATH
      Select the uniShell runtime root directory.

  --multiplexer NAME
      Select the multiplexer to use.

      Accepted values:
        tmux
        zellij
        none
        disabled

      If --multiplexer is omitted, uniShell starts a normal
      enhanced shell without a multiplexer.

      An invalid value starts an interactive selection:
        1. tmux
        2. zellij
        3. none
        4. quit

      Ctrl+C or selecting quit safely cancels startup.

  --new-session
      Start a new multiplexer session without attaching to or prompting
      about existing sessions.

  --no-shared-rc / --shared-rc
      Explicitly skip or load shared shell configuration, overriding the
      persistent config value.

Clean subcommand:
  unishell clean [--target SESSION]
      Clean one managed session or interactively select sessions.
  unishell clean --installed
      Delete the persistent runtime after two confirmations.
  unishell list
      Print session ID, name, and type.

Environment:
  UNISHELL_SHELL
      Shell selection fallback when --shell is not specified.

  UNISHELL_MULTIPLEXER
      Multiplexer selection fallback when --multiplexer is not specified.

      Accepted values are:
        tmux
        zellij
        none
        disabled

  UNISHELL_RUNTIME_DIR
      Runtime root when --runtime-dir is not specified; defaults to
      ~/.local/unishell for persistent installs.

  UNISHELL_TMUX_OPTS
      Additional tmux creation options.

  UNISHELL_ZELLIJ_OPTS
      Additional Zellij creation options.

  UNISHELL_UPGRADE_DIRECT_LINK
      Optional HTTPS URL to use for the update binary download.

Behavior:
  By default, uniShell uses an ephemeral runtime and removes its session
  runtime when the shell exits. After 'unishell install', launches use a
  persistent runtime under ~/.local/unishell and retain session data.

  With tmux or Zellij selected, uniShell creates or reattaches to the
  multiplexer session. Detaching preserves the session runtime so it can
  be reattached later.

  In persistent mode, 'unishell clean --installed' asks twice before removing
  runtime data, saved credentials, and configuration. The binary remains.

Examples:
  unishell
  unishell --shell zsh
  unishell --multiplexer tmux
  unishell --multiplexer zellij
  unishell --multiplexer none
  unishell install
  unishell update
  unishell upgrade
  unishell clean
  unishell list
  UNISHELL_MULTIPLEXER=tmux unishell`)
}
