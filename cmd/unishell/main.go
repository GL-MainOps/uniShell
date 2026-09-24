package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
	"gitlab.com/mainops/uniShell/internal/shell"
	"gitlab.com/mainops/uniShell/internal/shell/profile"
)

var (
	version = "development"
	commit  = "unknown"
)

func newApplication(options cliOptions) (*app.App, error) {
	return app.New(app.Options{
		Version:                version,
		Commit:                 commit,
		Root:                   options.RuntimeDir,
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

	application, err := newApplication(options)
	if err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}

	if err := run(application, args); err != nil {
		printError(err)
		os.Exit(exitCode(err))
	}
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
		return runInstall(application, commandArgs)

	case "update":
		return runUpdate(application, commandArgs)

	case "clean":
		return runClean(application, commandArgs)

	case "detach":
		return runDetach(application, commandArgs)

	case "doctor":
		return runDoctor(application, commandArgs)

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

	return runtimeSession.Cleanup()
}

type multiSessionDiscovery interface {
	DiscoverMultiplexerSessions() ([]*multiplexer.ManagedSession, error)
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
					live = append(live, &app.Session{Multiplexer: managed})
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
					return selected.Attach()
				}
			}
		} else {
			session, err := application.DiscoverMultiplexerSession()
			if err == nil {
				printReattachMessage(application, session)
				return session.Attach()
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

func runInstall(app *app.App, args []string) error {
	fmt.Println("uniShell install: not implemented")
	return nil
}

func runUpdate(app *app.App, args []string) error {
	fmt.Println("uniShell update: not implemented")
	return nil
}

type cleanOptions struct {
	Target string
}

func parseCleanArgs(args []string) (cleanOptions, error) {
	var options cleanOptions

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
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

func runClean(
	application cleanApplication,
	args []string,
) error {
	reader := bufio.NewReader(os.Stdin)

	options, err := parseCleanArgs(args)
	if err != nil {
		return err
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

func runDoctor(app *app.App, args []string) error {
	fmt.Println("uniShell doctor: not implemented")
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
  install     Install the uniShell runtime (not implemented)
  update      Update the uniShell runtime (not implemented)
  clean       Remove the current uniShell multiplexer runtime
  detach      Detach from the current uniShell multiplexer session
  doctor      Diagnose the uniShell environment (not implemented)
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
      Default runtime root when --runtime-dir is not specified.

  UNISHELL_TMUX_OPTS
      Additional tmux creation options.

  UNISHELL_ZELLIJ_OPTS
      Additional Zellij creation options.

Behavior:
  With no multiplexer selected, uniShell prepares an isolated runtime,
  starts the selected enhanced shell directly, and removes the session
  runtime when the shell exits.

  With tmux or Zellij selected, uniShell creates or reattaches to the
  multiplexer session. Detaching preserves the session runtime so it can
  be reattached later. The runtime is removed only after the multiplexer
  session has actually exited.

  The uniShell runtime root itself is preserved; cleanup removes only
  the session-specific runtime.

Examples:
  unishell
  unishell --shell zsh
  unishell --multiplexer tmux
  unishell --multiplexer zellij
  unishell --multiplexer none
  UNISHELL_MULTIPLEXER=tmux unishell`)
}
