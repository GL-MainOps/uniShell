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
