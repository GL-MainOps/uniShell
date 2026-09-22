package zellij

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

const zellijSessionConfigName = "session-config.kdl"

func sessionConfigPath(session api.Session) string {
	if session.Runtime == "" {
		return ""
	}

	return filepath.Join(
		session.Runtime,
		"config",
		"zellij",
		zellijSessionConfigName,
	)
}

func prepareSessionConfig(
	session api.Session,
	baseConfigPath string,
) (string, error) {
	if session.Runtime == "" {
		return baseConfigPath, nil
	}

	path := sessionConfigPath(session)

	var content []byte

	if baseConfigPath != "" {
		var err error

		content, err = os.ReadFile(baseConfigPath)
		if err != nil {
			return "", fmt.Errorf(
				"read zellij config %q: %w",
				baseConfigPath,
				err,
			)
		}
	}

	updated := overrideDefaultShell(
		string(content),
		sessionShellConfigValue(session),
	)

	if err := os.MkdirAll(
		filepath.Dir(path),
		0700,
	); err != nil {
		return "", fmt.Errorf(
			"create zellij session config directory: %w",
			err,
		)
	}

	if err := os.WriteFile(
		path,
		[]byte(updated),
		0600,
	); err != nil {
		return "", fmt.Errorf(
			"write zellij session config %q: %w",
			path,
			err,
		)
	}

	return path, nil
}

func sessionShellConfigValue(session api.Session) string {
	return strconv.Quote(
		sessionShellPath(session),
	)
}

func sessionShellPath(session api.Session) string {
	return filepath.Join(
		session.Runtime,
		"scripts",
		zellijShellScript,
	)
}

func overrideDefaultShell(
	content string,
	value string,
) string {
	lines := strings.Split(content, "\n")
	replaced := false

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "//") ||
			!strings.HasPrefix(trimmed, "default_shell") {
			continue
		}

		fields := strings.Fields(trimmed)

		if len(fields) == 0 ||
			fields[0] != "default_shell" {
			continue
		}

		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		lines[index] =
			indent + "default_shell " + value
		replaced = true
	}

	if !replaced {
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		content += "default_shell " + value + "\n"
		return content
	}

	return strings.Join(lines, "\n")
}
