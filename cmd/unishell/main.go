package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	"gitlab.com/mainops/uniShell/internal/shell"
)

var (
	version = "development"
	commit  = "unknown"
)

func main() {
	options, args, err := parseCLIArgs(os.Args[1:])
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	application, err := app.New(app.Options{
		Version:         version,
		Commit:          commit,
		Root:            options.RuntimeDir,
		Shell:           options.Shell,
		MultiplexerName: options.Multiplexer,
	})
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if err := run(application, args); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func printError(err error) {
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
	RequestedMultiplexer() string
	PrepareMultiplexerSession() (*runtime.Session, error)
	CreateMultiplexerSession(
		*runtime.Session,
		string,
		string,
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

	selected, err := selectShell(
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

	resolved, err := shell.Resolve(
		selected,
		runtimeSession.Paths.Bin,
	)
	if err != nil {
		return cleanupRuntime(
			fmt.Errorf(
				"resolve shell: %w",
				err,
			),
		)
	}

	command, err := shell.NewCommand(
		resolved,
		runtimeSession.Paths.Bin,
		runtimeSession.Paths.Runtime,
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

func runMultiplexerShell(
	application shellApplication,
	ctx context.Context,
	multiplexerName string,
) error {
	session, err := application.DiscoverMultiplexerSession()
	if err == nil {
		printReattachMessage(application, session)

		return session.Attach()
	}

	if !errors.Is(err, multiplexer.ErrSessionNotFound) {
		return fmt.Errorf(
			"discover multiplexer session: %w",
			err,
		)
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

	selected, err := selectShell(
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

	session, err = application.CreateMultiplexerSession(
		runtimeSession,
		multiplexerName,
		selected,
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

type cleanApplication interface {
	DiscoverMultiplexerSession() (*app.Session, error)
}

func runClean(application cleanApplication, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("clean does not accept arguments")
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

	if err := session.Cleanup(); err != nil {
		return fmt.Errorf(
			"clean multiplexer session: %w",
			err,
		)
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
