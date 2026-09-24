package main

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed completions/unishell.bash completions/unishell.zsh completions/unishell.fish
var completionFiles embed.FS

func runCompletion(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("completion requires one shell: bash, zsh, or fish")
	}

	shellName := strings.ToLower(strings.TrimSpace(args[0]))
	filename := map[string]string{
		"bash": "completions/unishell.bash",
		"zsh":  "completions/unishell.zsh",
		"fish": "completions/unishell.fish",
	}[shellName]
	if filename == "" {
		return fmt.Errorf("unsupported completion shell %q; choose bash, zsh, or fish", args[0])
	}

	data, err := completionFiles.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read embedded %s completion: %w", shellName, err)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return fmt.Errorf("write %s completion: %w", shellName, err)
	}
	return nil
}
