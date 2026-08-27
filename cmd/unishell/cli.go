package main

import (
	"fmt"
	"os"
	"strings"
)

const multiplexerEnvName = "UNISHELL_MULTIPLEXER"

type cliOptions struct {
	RuntimeDir  string
	Shell       string
	Multiplexer string
}

func parseCLIArgs(args []string) (cliOptions, []string, error) {
	var options cliOptions
	var commandArgs []string

	options.Multiplexer = strings.TrimSpace(
		os.Getenv(multiplexerEnvName),
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--runtime-dir":
			if i+1 >= len(args) {
				return cliOptions{}, nil, fmt.Errorf(
					"--runtime-dir requires a path",
				)
			}

			options.RuntimeDir = args[i+1]
			i++

		case strings.HasPrefix(arg, "--runtime-dir="):
			value := strings.TrimPrefix(arg, "--runtime-dir=")

			if value == "" {
				return cliOptions{}, nil, fmt.Errorf(
					"--runtime-dir requires a path",
				)
			}

			options.RuntimeDir = value

		case arg == "--shell":
			if i+1 >= len(args) {
				return cliOptions{}, nil, fmt.Errorf(
					"--shell requires a shell name",
				)
			}

			options.Shell = args[i+1]
			i++

		case strings.HasPrefix(arg, "--shell="):
			value := strings.TrimPrefix(arg, "--shell=")

			if value == "" {
				return cliOptions{}, nil, fmt.Errorf(
					"--shell requires a shell name",
				)
			}

			options.Shell = value

		case arg == "--multiplexer":
			if i+1 >= len(args) {
				return cliOptions{}, nil, fmt.Errorf(
					"--multiplexer requires a value",
				)
			}

			options.Multiplexer = args[i+1]
			i++

		case strings.HasPrefix(arg, "--multiplexer="):
			value := strings.TrimPrefix(
				arg,
				"--multiplexer=",
			)

			if value == "" {
				return cliOptions{}, nil, fmt.Errorf(
					"--multiplexer requires a value",
				)
			}

			options.Multiplexer = value

		default:
			commandArgs = append(commandArgs, arg)
		}
	}

	return options, commandArgs, nil
}
