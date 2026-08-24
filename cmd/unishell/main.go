package main

import (
	"fmt"
	"gitlab.com/mainops/uniShell/internal/app"
	"os"
)

var (
	version = "development"
	commit  = "unknown"
)

func main() {
	app, err := app.New(app.Options{
		Version: version,
		Commit:  commit,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "uniShell: %v\n", err)
		os.Exit(1)
	}

	if err := run(app, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "uniShell: %v\n", err)
		os.Exit(1)
	}
}

func run(app *app.App, args []string) error {
	command := "shell"
	commandArgs := args

	if len(args) > 0 {
		command = args[0]
		commandArgs = args[1:]
	}

	switch command {
	case "shell":
		return runShell(app, commandArgs)

	case "install":
		return runInstall(app, commandArgs)

	case "update":
		return runUpdate(app, commandArgs)

	case "clean":
		return runClean(app, commandArgs)

	case "doctor":
		return runDoctor(app, commandArgs)

	case "version":
		return runVersion(app, commandArgs)

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
