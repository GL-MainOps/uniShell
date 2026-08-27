package main

import (
	"errors"
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
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
		Version: version,
		Commit:  commit,
		Root:    options.RuntimeDir,
		Shell:   options.Shell,
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
	StartMultiplexerSession() (*app.Session, error)
	DiscoverMultiplexerSession() (*app.Session, error)
}

func runShell(application shellApplication, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf(
			"shell does not accept arguments",
		)
	}

	session, err := application.DiscoverMultiplexerSession()
	if err == nil {
		return session.Attach()
	}

	if !errors.Is(err, multiplexer.ErrSessionNotFound) {
		return fmt.Errorf(
			"discover multiplexer session: %w",
			err,
		)
	}

	session, err = application.StartMultiplexerSession()
	if err != nil {
		return fmt.Errorf(
			"start multiplexer session: %w",
			err,
		)
	}

	if err := session.Attach(); err != nil {
		_ = session.Cleanup()

		return fmt.Errorf(
			"attach new multiplexer session: %w",
			err,
		)
	}

	return nil
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
  unishell [command] [options]

Commands:
  shell       Start or attach to the uniShell environment
  install     Install the uniShell runtime
  update      Update the uniShell runtime
  clean       Remove the uniShell runtime
  detach      Detach from the uniShell multiplexer session
  doctor      Diagnose the uniShell environment
  version     Display version information
  help        Display this help message

Options:
  --shell NAME
              Select the shell to use
  --runtime-dir PATH
              Select the uniShell runtime directory`)
}
