package config

import (
	"fmt"
	"strings"
)

type BashAdapter struct{}

func (BashAdapter) Name() string {
	return "bash"
}

func (BashAdapter) Render(cfg Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	var out strings.Builder

	out.WriteString(renderHeader("bash"))

	for _, name := range cfg.EnvironmentNames() {
		fmt.Fprintf(
			&out,
			"export %s=%s\n",
			name,
			renderPOSIXValue(cfg.Environment[name]),
		)
	}

	if len(cfg.Path.Add) > 0 {
		entries := make([]string, 0, len(cfg.Path.Add))

		for _, entry := range cfg.Path.Add {
			entries = append(entries, renderPOSIXValue(entry))
		}

		fmt.Fprintf(
			&out,
			"export PATH=%s:\"$PATH\"\n",
			strings.Join(entries, ":"),
		)
	}

	for _, name := range cfg.AliasNames() {
		fmt.Fprintf(
			&out,
			"alias %s=%s\n",
			name,
			renderPOSIXValue(cfg.Aliases[name]),
		)
	}

	return out.String(), nil
}
