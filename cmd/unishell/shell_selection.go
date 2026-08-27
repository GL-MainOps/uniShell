package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"golang.org/x/term"

	"gitlab.com/mainops/uniShell/internal/shell"
)

var errShellSelectionCancelled = errors.New(
	"shell selection cancelled",
)

type shellSelectionInput struct {
	value string
	err   error
}

func selectShell(
	ctx context.Context,
	runtimeBin string,
	requested string,
	in io.Reader,
	out io.Writer,
) (string, error) {
	selected, err := shell.Resolve(
		requested,
		runtimeBin,
	)
	if err == nil {
		return selected.Name, nil
	}

	fmt.Fprintf(
		out,
		"uniShell: shell %q is unavailable or unsupported.\n\n",
		requested,
	)

	available := availableShells(runtimeBin)
	if len(available) == 0 {
		return "", fmt.Errorf(
			"no supported shells are available; install or provide one of: %s",
			strings.Join(shell.SupportedShells(), ", "),
		)
	}

	printAvailableShells(out, available)

	if file, ok := in.(*os.File); ok &&
		term.IsTerminal(int(file.Fd())) {
		return selectShellFromTerminal(
			ctx,
			file,
			out,
			available,
		)
	}

	return selectShellFromReader(
		ctx,
		in,
		out,
		available,
	)
}

func printAvailableShells(
	out io.Writer,
	available []shell.Shell,
) {
	fmt.Fprintln(out, "Available shells:")

	for index, candidate := range available {
		fmt.Fprintf(
			out,
			"  %d. %s\n",
			index+1,
			candidate.Name,
		)
	}

	fmt.Fprintln(out, "  q. Exit")
}

func selectShellFromReader(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	available []shell.Shell,
) (string, error) {
	scanner := bufio.NewScanner(in)
	input := make(chan shellSelectionInput, 1)

	readInput := func() {
		if scanner.Scan() {
			input <- shellSelectionInput{
				value: scanner.Text(),
			}
			return
		}

		input <- shellSelectionInput{
			err: scanner.Err(),
		}
	}

	go readInput()

	for {
		fmt.Fprint(out, "\nSelect a shell: ")

		select {
		case <-ctx.Done():
			return "", errShellSelectionCancelled

		case result := <-input:
			if result.err != nil {
				if errors.Is(
					result.err,
					io.EOF,
				) {
					return "", errShellSelectionCancelled
				}

				return "", fmt.Errorf(
					"read shell selection: %w",
					result.err,
				)
			}

			selected, cancelled, valid := parseShellSelection(
				result.value,
				available,
			)

			if cancelled {
				fmt.Fprintln(out)
				return "", errShellSelectionCancelled
			}

			if !valid {
				fmt.Fprintln(
					out,
					"Invalid selection. Choose one of the listed shells or 'q' to exit.",
				)

				go readInput()
				continue
			}

			return selected, nil
		}
	}
}

func selectShellFromTerminal(
	ctx context.Context,
	in *os.File,
	out io.Writer,
	available []shell.Shell,
) (string, error) {
	fd := int(in.Fd())

	state, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf(
			"enable interactive shell selection: %w",
			err,
		)
	}

	defer func() {
		_ = term.Restore(fd, state)
	}()

	for {
		fmt.Fprint(out, "\nSelect a shell: ")

		value, err := readTerminalLine(ctx, in, out)
		if err != nil {
			return "", err
		}

		selected, cancelled, valid := parseShellSelection(
			value,
			available,
		)

		if cancelled {
			fmt.Fprintln(out)
			return "", errShellSelectionCancelled
		}

		if !valid {
			fmt.Fprintln(
				out,
				"\r\nInvalid selection. Choose one of the listed shells or 'q' to exit.",
			)
			continue
		}

		fmt.Fprintln(out)

		return selected, nil
	}
}

func readTerminalLine(
	ctx context.Context,
	in *os.File,
	out io.Writer,
) (string, error) {
	buffer := make([]byte, 1)
	line := make([]byte, 0, 16)

	for {
		select {
		case <-ctx.Done():
			return "", errShellSelectionCancelled

		default:
		}

		n, err := in.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", errShellSelectionCancelled
			}

			return "", fmt.Errorf(
				"read terminal input: %w",
				err,
			)
		}

		if n == 0 {
			continue
		}

		switch buffer[0] {
		case 0x03:
			// Ctrl+C.
			return "", errShellSelectionCancelled

		case '\r', '\n':
			return string(line), nil

		case 0x7f, 0x08:
			// Backspace.
			if len(line) == 0 {
				continue
			}

			line = line[:len(line)-1]
			fmt.Fprint(out, "\b \b")

		case 0x04:
			// Ctrl+D on an empty line behaves as cancellation.
			if len(line) == 0 {
				return "", errShellSelectionCancelled
			}

		default:
			if buffer[0] < 0x20 {
				continue
			}

			line = append(line, buffer[0])
			fmt.Fprintf(out, "%c", buffer[0])
		}
	}
}

func parseShellSelection(
	value string,
	available []shell.Shell,
) (selected string, cancelled bool, valid bool) {
	value = strings.TrimSpace(value)

	if strings.EqualFold(value, "q") ||
		strings.EqualFold(value, "quit") ||
		strings.EqualFold(value, "exit") {
		return "", true, false
	}

	index, err := strconv.Atoi(value)
	if err != nil ||
		index < 1 ||
		index > len(available) {
		return "", false, false
	}

	return available[index-1].Name, false, true
}

func availableShells(runtimeBin string) []shell.Shell {
	result := make([]shell.Shell, 0)

	for _, name := range shell.SupportedShells() {
		selected, err := shell.Resolve(name, runtimeBin)
		if err != nil {
			continue
		}

		result = append(result, selected)
	}

	return result
}

func shellSelectionContext() (
	context.Context,
	context.CancelFunc,
) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)
}
