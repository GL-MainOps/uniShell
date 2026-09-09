package app

import (
	"fmt"
	"strings"

	"gitlab.com/mainops/uniShell/internal/runtime"
)

const sessionNameIDLength = 7

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
