package config

import (
	"fmt"
	"strings"
)

type FishAdapter struct{}

func (FishAdapter) Name() string {
	return "fish"
}

func (FishAdapter) Render(cfg Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	var out strings.Builder

	out.WriteString(renderHeader("fish"))

	for _, name := range cfg.EnvironmentNames() {
		fmt.Fprintf(
			&out,
			"set -gx %s %s\n",
			name,
			renderFishValue(cfg.Environment[name]),
		)
	}

	if len(cfg.Path.Add) > 0 {
		entries := make([]string, 0, len(cfg.Path.Add))

		for _, entry := range cfg.Path.Add {
			entries = append(entries, renderFishValue(entry))
		}

		fmt.Fprintf(
			&out,
			"set -gx PATH %s $PATH\n",
			strings.Join(entries, " "),
		)
	}

	for _, name := range cfg.AliasNames() {
		fmt.Fprintf(
			&out,
			"alias %s %s\n",
			name,
			renderFishValue(cfg.Aliases[name]),
		)
	}

	return out.String(), nil
}
