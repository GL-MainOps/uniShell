package auth

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var ErrEmptyToken = errors.New("authentication token cannot be empty")

const environmentVariable = "UNISHELL_AUTH_TOKEN"

// Options controls how the authentication token is resolved.
type Options struct {
	Explicit string
}

// Resolve returns an authentication token using the configured precedence.
//
// Explicit CLI input takes precedence over the environment variable.
// If neither is available, an interactive hidden prompt is used.
func Resolve(options Options) (string, error) {
	if options.Explicit != "" {
		return validate(options.Explicit)
	}

	if token := os.Getenv(environmentVariable); token != "" {
		return validate(token)
	}

	return prompt()
}

func validate(token string) (string, error) {
	if token == "" {
		return "", ErrEmptyToken
	}

	return token, nil
}

func prompt() (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf(
			"authentication token is required; provide --auth or set %s",
			environmentVariable,
		)
	}

	fmt.Fprint(os.Stderr, "uniShell authentication token: ")

	token, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", fmt.Errorf("read authentication token: %w", err)
	}

	if len(token) == 0 {
		return "", ErrEmptyToken
	}

	return string(token), nil
}
