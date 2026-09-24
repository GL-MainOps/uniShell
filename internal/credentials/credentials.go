package credentials

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var ErrEmptyToken = errors.New("authentication token cannot be empty")

const environmentVariable = "UNISHELL_AUTH_TOKEN"

// Resolve returns an authentication token.
//
// The environment variable is checked first. If it is unavailable,
// an interactive hidden prompt is used.
func Resolve() (string, error) {
	if token := os.Getenv(environmentVariable); token != "" {
		return validate(token)
	}

	return prompt()
}

// ResolveForRuntime prefers an explicit environment token, then the encrypted
// persistent token file, then an interactive prompt. The bool reports whether
// the returned value came from the local store so callers can retry with a
// freshly entered token if bundle validation fails.
func ResolveForRuntime(runtimeRoot string, persistent bool) (string, bool, error) {
	if token := os.Getenv(environmentVariable); token != "" {
		resolved, err := validate(token)
		return resolved, false, err
	}

	if persistent {
		token, err := ReadStoredToken(runtimeRoot)
		if err == nil {
			return token, true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "uniShell: could not read saved token: %v\n", err)
		}
	}

	token, err := prompt()
	return token, false, err
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
			"authentication token is required; provide %s",
			environmentVariable,
		)
	}

	fmt.Fprint(os.Stderr, "Enter Token: ")

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
