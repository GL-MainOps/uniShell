package app

import (
	"fmt"
	"strings"

	"gitlab.com/mainops/uniShell/internal/runtime"
)

const (
	sessionNameIDLength            = 7
	multiplexerSessionNameIDLength = 3
)

func multiplexerSessionBaseName(
	specifiedName string,
) string {
	baseName := strings.TrimSpace(specifiedName)
	if baseName == "" {
		return "uS"
	}

	return baseName
}

func sessionNameForRuntime(
	runtimeSession *runtime.Session,
	specifiedName string,
	specified bool,
) (string, error) {
	if runtimeSession == nil {
		return "", fmt.Errorf("runtime session is nil")
	}

	if len(runtimeSession.ID) < sessionNameIDLength {
		return "", fmt.Errorf(
			"runtime session ID %q is shorter than %d characters",
			runtimeSession.ID,
			sessionNameIDLength,
		)
	}

	baseName := strings.TrimSpace(specifiedName)
	if !specified || baseName == "" {
		baseName = "unnamed"
	}

	return fmt.Sprintf(
		"%s@%s",
		baseName,
		runtimeSession.ID[:sessionNameIDLength],
	), nil
}

func multiplexerSessionNameForRuntime(
	runtimeSession *runtime.Session,
	specifiedName string,
) (string, error) {
	if runtimeSession == nil {
		return "", fmt.Errorf("runtime session is nil")
	}

	if len(runtimeSession.ID) < multiplexerSessionNameIDLength {
		return "", fmt.Errorf(
			"runtime session ID %q is shorter than %d characters",
			runtimeSession.ID,
			multiplexerSessionNameIDLength,
		)
	}

	baseName := multiplexerSessionBaseName(specifiedName)

	return fmt.Sprintf(
		"%s@%s",
		baseName,
		runtimeSession.ID[:multiplexerSessionNameIDLength],
	), nil
}
