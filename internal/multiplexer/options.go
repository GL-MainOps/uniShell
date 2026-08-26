package multiplexer

import (
	"fmt"
	"os"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/shellargs"
)

const (
	tmuxOptionsEnvironmentVariable   = "UNISHELL_TMUX_OPTS"
	zellijOptionsEnvironmentVariable = "UNISHELL_ZELLIJ_OPTS"
)

// ParseOptionsFromEnvironment resolves multiplexer creation options from
// the supported environment variables.
//
// Empty or unset variables produce zero-value options.
func ParseOptionsFromEnvironment() (api.Options, error) {
	tmuxOptions, err := parseEnvironmentOptions(
		tmuxOptionsEnvironmentVariable,
	)
	if err != nil {
		return api.Options{}, err
	}

	zellijOptions, err := parseEnvironmentOptions(
		zellijOptionsEnvironmentVariable,
	)
	if err != nil {
		return api.Options{}, err
	}

	return api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: tmuxOptions,
		},
		Zellij: api.ZellijOptions{
			CreateArgs: zellijOptions,
		},
	}, nil
}

func parseEnvironmentOptions(
	name string,
) ([]string, error) {
	value := os.Getenv(name)
	if value == "" {
		return nil, nil
	}

	args, err := shellargs.Tokenize(value)
	if err != nil {
		return nil, fmt.Errorf(
			"parse %s: %w",
			name,
			err,
		)
	}

	return args, nil
}
