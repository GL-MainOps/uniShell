package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Startup struct {
	Args []string
	Env  map[string]string
}

func BuildStartup(
	shellName string,
	startupPath string,
	runtimeDir string,
) (Startup, error) {
	if shellName == "" {
		return Startup{}, errors.New(
			"shell name cannot be empty",
		)
	}

	if startupPath == "" {
		return Startup{}, errors.New(
			"shell startup path cannot be empty",
		)
	}

	if runtimeDir == "" {
		return Startup{}, errors.New(
			"shell startup runtime directory cannot be empty",
		)
	}

	switch shellName {
	case "bash":
		return Startup{
			Args: []string{
				"--noprofile",
				"--rcfile",
				startupPath,
			},
		}, nil

	case "zsh":
		return buildZshStartup(
			startupPath,
			runtimeDir,
		)

	case "fish":
		return Startup{
			Args: []string{
				"--no-config",
				"--init-command",
				fmt.Sprintf(
					"source %s",
					shellQuote(startupPath),
				),
			},
		}, nil

	case "nushell":
		return Startup{
			Args: []string{
				"--config",
				startupPath,
			},
		}, nil

	default:
		return Startup{}, fmt.Errorf(
			"unsupported shell: %q",
			shellName,
		)
	}
}

func buildZshStartup(
	startupPath string,
	runtimeDir string,
) (Startup, error) {
	zdotdir := filepath.Join(
		runtimeDir,
		"config",
		"shell",
		"zsh",
	)

	if err := os.MkdirAll(zdotdir, 0700); err != nil {
		return Startup{}, fmt.Errorf(
			"create zsh startup directory: %w",
			err,
		)
	}

	zshrc := filepath.Join(zdotdir, ".zshrc")

	content := fmt.Sprintf(
		"source %s\n",
		shellQuote(startupPath),
	)

	if err := os.WriteFile(
		zshrc,
		[]byte(content),
		0600,
	); err != nil {
		return Startup{}, fmt.Errorf(
			"write zsh startup file: %w",
			err,
		)
	}

	return Startup{
		Args: []string{"-d"},
		Env: map[string]string{
			"ZDOTDIR": zdotdir,
		},
	}, nil
}

func shellQuote(value string) string {
	return "'" + replaceSingleQuotes(value) + "'"
}

func replaceSingleQuotes(value string) string {
	result := ""

	for _, part := range splitSingleQuotes(value) {
		if result != "" {
			result += "'\"'\"'"
		}

		result += part
	}

	return result
}

func splitSingleQuotes(value string) []string {
	parts := []string{""}

	for _, char := range value {
		if char == '\'' {
			parts = append(parts, "")
			continue
		}

		parts[len(parts)-1] += string(char)
	}

	return parts
}
