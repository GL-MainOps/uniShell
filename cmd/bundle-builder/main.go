package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"

	"gitlab.com/mainops/uniShell/internal/bundle"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "bundle-builder: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("bundle-builder", flag.ContinueOnError)

	input := flags.String(
		"input",
		"",
		"directory containing runtime assets",
	)

	output := flags.String(
		"output",
		"",
		"output encrypted bundle path",
	)

	generate := flags.String(
		"generate",
		"",
		"generated Go source file containing the encrypted bundle",
	)

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *input == "" {
		return errors.New("input directory is required")
	}

	if *output == "" {
		return errors.New("output bundle path is required")
	}

	if *generate == "" {
		return errors.New("generated Go source path is required")
	}

	password, err := resolvePassword()
	if err != nil {
		return err
	}

	data, err := bundle.Create(*input, password)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	if err := writeOutput(*output, data); err != nil {
		return err
	}

	if err := generateBundleSource(*generate, data); err != nil {
		return err
	}

	return nil
}

func resolvePassword() (string, error) {
	if token := os.Getenv("UNISHELL_AUTH_TOKEN"); token != "" {
		return token, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", errors.New(
			"UNISHELL_AUTH_TOKEN is required when stdin is not interactive",
		)
	}

	fmt.Fprint(os.Stderr, "Bundle Token: ")

	token, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", fmt.Errorf("read bundle token: %w", err)
	}

	if len(token) == 0 {
		return "", errors.New("bundle token cannot be empty")
	}

	return string(token), nil
}

func writeOutput(path string, data []byte) error {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil {
		return fmt.Errorf("create output bundle: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write output bundle: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close output bundle: %w", err)
	}

	return nil
}
