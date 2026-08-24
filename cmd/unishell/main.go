package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/runtime"
)

var (
	version = "development"
	commit  = "unknown"
)

func main() {
	runtimeDir := flag.String(
		"runtime-dir",
		"",
		"uniShell runtime directory",
	)

	flag.Parse()

	application, err := app.New(app.Options{
		Version: version,
		Commit:  commit,
		Root:    *runtimeDir,
	})
	if err != nil {
		printError(err)
		os.Exit(1)
	}

	if err := run(application, flag.Args()); err != nil {
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

	case "doctor":
		return runDoctor(application, commandArgs)

	case "version":
		return runVersion(application, commandArgs)

	case "help", "--help", "-h":
		printHelp()
		return nil

	default:
		return fmt.Errorf("unknown command %q; use 'help' for usage", command)
	}
}

func runShell(app *app.App, args []string) error {
	fmt.Println("uniShell shell: not implemented")
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

func runClean(app *app.App, args []string) error {
	fmt.Println("uniShell clean: not implemented")
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
  unishell [command]

Commands:
  shell       Start the uniShell environment
  install     Install the uniShell runtime
  update      Update the uniShell runtime
  clean       Remove the uniShell runtime
  doctor      Diagnose the uniShell environment
  version     Display version information
  help        Display this help message`)
}
