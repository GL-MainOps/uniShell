package config

import (
	"fmt"
	"strings"
)

type NushellAdapter struct{}

func (NushellAdapter) Name() string {
	return "nushell"
}

func (NushellAdapter) Render(cfg Config) (string, error) {
	if err := cfg.Validate(); err != nil {
		return "", err
	}

	var out strings.Builder

	out.WriteString(renderHeader("nushell"))

	for _, name := range cfg.EnvironmentNames() {
		fmt.Fprintf(
			&out,
			"$env.%s = %s\n",
			name,
			renderNushellValue(cfg.Environment[name]),
		)
	}

	if len(cfg.Path.Add) > 0 {
		entries := make([]string, 0, len(cfg.Path.Add))

		for _, entry := range cfg.Path.Add {
			entries = append(entries, renderNushellValue(entry))
		}

		fmt.Fprintf(
			&out,
			"$env.PATH = (%s | append $env.PATH)\n",
			strings.Join(entries, " | append "),
		)
	}

	for _, name := range cfg.AliasNames() {
		fmt.Fprintf(
			&out,
			"alias %s = %s\n",
			name,
			cfg.Aliases[name],
		)
	}

	return out.String(), nil
}

func renderNushellValue(value string) string {
	return "\"" + strings.ReplaceAll(
		strings.ReplaceAll(value, `\`, `\\`),
		`"`,
		`\"`,
	) + "\""
}
