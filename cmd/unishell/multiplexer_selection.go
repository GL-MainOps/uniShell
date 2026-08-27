package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var errMultiplexerSelectionCancelled = errors.New(
	"multiplexer selection cancelled",
)

const (
	multiplexerNone   = "none"
	multiplexerTmux   = "tmux"
	multiplexerZellij = "zellij"
)

func selectMultiplexer(
	ctx context.Context,
	requested string,
	in io.Reader,
	out io.Writer,
) (string, error) {
	requested = normalizeMultiplexerName(requested)

	if requested == "" ||
		requested == multiplexerNone ||
		requested == "disabled" {
		return multiplexerNone, nil
	}

	if requested == multiplexerTmux ||
		requested == multiplexerZellij {
		return requested, nil
	}

	fmt.Fprintf(
		out,
		"uniShell: multiplexer %q is not accepted.\n\n",
		requested,
	)
	fmt.Fprintln(out, "Available options:")
	fmt.Fprintln(out, "  1. tmux")
	fmt.Fprintln(out, "  2. zellij")
	fmt.Fprintln(out, "  3. none")
	fmt.Fprintln(out, "  q. quit")

	if file, ok := in.(*os.File); ok &&
		term.IsTerminal(int(file.Fd())) {
		return selectMultiplexerFromTerminal(
			ctx,
			file,
			out,
		)
	}

	return selectMultiplexerFromReader(
		ctx,
		in,
		out,
	)
}

func normalizeMultiplexerName(value string) string {
	return strings.ToLower(
		strings.TrimSpace(value),
	)
}

func selectMultiplexerFromReader(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
) (string, error) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(
			out,
			"\nSelect a multiplexer [1-3/q]: ",
		)

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
			return "", errMultiplexerSelectionCancelled

		case err := <-errCh:
			if errors.Is(err, io.EOF) {
				return "", errMultiplexerSelectionCancelled
			}

			return "", fmt.Errorf(
				"read multiplexer selection: %w",
				err,
			)

		case value := <-input:
			selected, valid, cancelled :=
				parseMultiplexerSelection(value)

			if cancelled {
				fmt.Fprintln(out)
				return "", errMultiplexerSelectionCancelled
			}

			if !valid {
				fmt.Fprintln(
					out,
					"Invalid selection. Choose [1-3/q].",
				)
				continue
			}

			return selected, nil
		}
	}
}

func selectMultiplexerFromTerminal(
	ctx context.Context,
	in *os.File,
	out io.Writer,
) (string, error) {
	fd := int(in.Fd())

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf(
			"enable interactive multiplexer selection: %w",
			err,
		)
	}

	defer func() {
		_ = term.Restore(fd, state)
	}()

	for {
		fmt.Fprint(
			out,
			"\nSelect a multiplexer [1-3/q]: ",
		)

		value, err := readTerminalLine(
			ctx,
			in,
			out,
		)
		if err != nil {
			return "", err
		}

		selected, valid, cancelled :=
			parseMultiplexerSelection(value)

		if cancelled {
			fmt.Fprintln(out)
			return "", errMultiplexerSelectionCancelled
		}

		if !valid {
			fmt.Fprintln(
				out,
				"\r\nInvalid selection. Choose [1-3/q].",
			)
			continue
		}

		fmt.Fprintln(out)

		return selected, nil
	}
}

func parseMultiplexerSelection(
	value string,
) (selected string, valid bool, cancelled bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "tmux":
		return multiplexerTmux, true, false

	case "2", "zellij":
		return multiplexerZellij, true, false

	case "3", "none", "disabled":
		return multiplexerNone, true, false

	case "q", "quit", "exit":
		return "", false, true

	default:
		return "", false, false
	}
}
