package main

import (
	"fmt"
	"os"
)

var (
	version = "development"
	commit  = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "uniShell: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "shell"

	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "shell":
		return runShell(args[1:])

	case "install":
		return runInstall(args[1:])

	case "update":
		return runUpdate(args[1:])

	case "clean":
		return runClean(args[1:])

	case "doctor":
		return runDoctor(args[1:])

	case "version":
		return runVersion(args[1:])

	case "help", "--help", "-h":
		printHelp()
		return nil

	default:
		return fmt.Errorf("unknown command %q; use 'help' for usage", command)
	}
}

func runShell(args []string) error {
	fmt.Println("uniShell shell: not implemented")
	return nil
}

func runInstall(args []string) error {
	fmt.Println("uniShell install: not implemented")
	return nil
}

func runUpdate(args []string) error {
	fmt.Println("uniShell update: not implemented")
	return nil
}

func runClean(args []string) error {
	fmt.Println("uniShell clean: not implemented")
	return nil
}

func runDoctor(args []string) error {
	fmt.Println("uniShell doctor: not implemented")
	return nil
}

func runVersion(args []string) error {
	fmt.Printf("uniShell %s (%s)\n", version, commit)
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
